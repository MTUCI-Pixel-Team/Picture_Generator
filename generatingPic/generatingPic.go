package generatingPic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	URL                   = "wss://ws-api.runware.ai/v1"
	SEND_CHAN_MAX_SIZE    = 128
	RECEIVE_CHAN_MAX_SIZE = 128
	TIMEOUT               = 15 * time.Second
	RECONNECTED_DELAY     = 3 * time.Second
	RECONNECT_TIMEOUT     = 10 * time.Second
	MAX_RETRIES           = 3
	WRITE_TIMEOUT         = 10 * time.Second
	READ_TIMEOUT          = 30 * time.Second
)

type ReqMessage struct {
	TaskType       string   `json:"taskType,omitempty"`
	TaskUUID       string   `json:"taskUUID,omitempty"`
	OutputType     []string `json:"outputType,omitempty"`
	OutputFormat   string   `json:"outputFormat,omitempty"`
	PositivePrompt string   `json:"positivePrompt,omitempty"`
	NegativePrompt string   `json:"negativePrompt,omitempty"`
	Height         int      `json:"height,omitempty"`
	Width          int      `json:"width,omitempty"`
	Model          string   `json:"model,omitempty"`
	Steps          int      `json:"steps,omitempty"`
	CFGScale       float64  `json:"CFGScale,omitempty"`
	NumberResults  int      `json:"numberResults,omitempty"`
	ApiKey         string   `json:"apiKey,omitempty"`
}

type RespData struct {
	TaskType              string `json:"taskType"`
	TaskUUID              string `json:"taskUUID"`
	ImageUUID             string `json:"imageUUID"`
	NSFWContent           bool   `json:"NSFWContent"`
	ConnectionSessionUUID string `json:"connectionSessionUUID"`
	ImageURL              string `json:"imageURL"`
}

type RespError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Parameter string `json:"parameter"`
	Type      string `json:"type"`
	TaskUUID  string `json:"taskUUID"`
}

type RespMessage struct {
	Data []RespData  `json:"data"`
	Err  []RespError `json:"errors"`
}

type WSClient struct {
	User struct {
		ID   uint
		UUID string
	}
	url            string
	apiKey         string
	socket         *websocket.Conn
	SendMsgChan    chan ReqMessage
	ReceiveMsgChan chan []byte
	ErrChan        chan error
	Done           chan struct{}
	CloseChan      chan struct{}
	socketMutex    sync.Mutex
	wg             sync.WaitGroup
	reconnecting   atomic.Bool
	closeOnce      sync.Once
	closed         atomic.Bool
}

func NewWSClient(apiKey string, userID uint) *WSClient {
	return &WSClient{
		User: struct {
			ID   uint
			UUID string
		}{ID: userID, UUID: ""},
		url:          URL,
		apiKey:       apiKey,
		socketMutex:  sync.Mutex{},
		reconnecting: atomic.Bool{},
		closeOnce:    sync.Once{},
		closed:       atomic.Bool{},
	}
}

func (ws *WSClient) Start(ctx context.Context) {
	ws.SendMsgChan = make(chan ReqMessage, SEND_CHAN_MAX_SIZE)
	ws.ReceiveMsgChan = make(chan []byte, RECEIVE_CHAN_MAX_SIZE)
	ws.ErrChan = make(chan error)
	ws.Done = make(chan struct{})
	ws.closed.Store(false)

	ws.socketMutex.Lock()
	if ws.socket == nil {
		ws.socketMutex.Unlock()
		fmt.Println("Connecting")
		err := ws.connect(ctx)
		if err != nil {
			log.Printf("Failed to connect: %v", err)
			ws.ErrChan <- err
			return
		}
	} else {
		ws.socketMutex.Unlock()
		ws.restart(ctx)
	}
}

func (ws *WSClient) connect(ctx context.Context) error {
	ws.socketMutex.Lock()
	defer ws.socketMutex.Unlock()
	if ws.socket != nil {
		return errors.New("socket already exists")
	}

	socket, _, err := websocket.DefaultDialer.Dial(ws.url, nil)
	if err != nil {
		return fmt.Errorf("dial error: %w", err)
	}
	fmt.Println("socket created")
	ws.socket = socket

	authCredentials := ReqMessage{
		TaskType: "authentication",
		ApiKey:   ws.apiKey,
	}
	formattaedReq := []ReqMessage{authCredentials}

	jsnoStr, err := json.Marshal(formattaedReq)
	if err != nil {
		return fmt.Errorf("failed to marshal auth request: %w", err)
	}
	fmt.Println(string(jsnoStr))

	err = ws.socket.WriteMessage(websocket.TextMessage, jsnoStr)
	if err != nil {
		return fmt.Errorf("failed to send auth request: %w", err)
	}
	_, resp, err := ws.socket.ReadMessage()
	if err != nil {
		return fmt.Errorf("failed to read auth response: %w", err)
	}
	response := new(RespMessage)
	if err := json.Unmarshal(resp, response); err != nil {
		return fmt.Errorf("failed to unmarshal auth response: %w", err)
	}
	if len(response.Err) > 0 {
		return fmt.Errorf("error response: %s", response.Err[0].Message)
	}
	if len(response.Data) == 0 {
		return errors.New("empty data in auth response")
	}
	ws.User.UUID = response.Data[0].ConnectionSessionUUID

	log.Println("Auth response:", string(resp))
	ws.wg.Add(3)
	go ws.sendPump(ctx)
	go ws.receivePump(ctx)
	go ws.errorHandler(ctx)

	return nil
}

func (ws *WSClient) restart(ctx context.Context) error {
	if !ws.reconnecting.CompareAndSwap(false, true) {
		return errors.New("reconnection already in progress")
	}
	defer ws.reconnecting.Store(false)

	_, cancel := context.WithTimeout(ctx, TIMEOUT)
	defer cancel()
	fmt.Println("Restarting connection")
	ws.socketMutex.Lock()
	if ws.socket != nil {
		ws.socketMutex.Unlock()
		err := ws.socket.Close()
		if err != nil {
			log.Printf("Error closing connection during restart: %v", err)
		}
		ws.socket = nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(RECONNECTED_DELAY):
	}

	return ws.connect(ctx)
}

func (ws *WSClient) sendPump(ctx context.Context) {
	defer ws.wg.Done()
	defer fmt.Println("sendsendPump Done")
	fmt.Println("sendPump started")
	for {
		if ws.closed.Load() {
			log.Println("sendPump stopping due to closed client")
			return
		}
		select {
		case <-ctx.Done():
			return
		case req, ok := <-ws.SendMsgChan:
			if !ok {
				return
			}

			formattedReq := []ReqMessage{req}
			jsonMsg, err := json.Marshal(formattedReq)
			if err != nil {
				ws.ErrChan <- err
			}

			fmt.Println(string(jsonMsg))
			ws.socketMutex.Lock()
			if ws.socket == nil {
				ws.socketMutex.Unlock()
				ws.ErrChan <- errors.New("Socket is nil")
				time.Sleep(time.Second)
				continue
			}

			// ws.socket.SetWriteDeadline(time.Now().Add(WRITE_TIMEOUT))
			ws.socketMutex.Unlock()
			err = ws.socket.WriteMessage(websocket.TextMessage, jsonMsg)

			if err != nil {
				fmt.Println("Error sending message")
				ws.ErrChan <- err
				time.Sleep(time.Second)
				continue
			}
		}
	}
}

func (ws *WSClient) receivePump(ctx context.Context) {
	defer ws.wg.Done()
	defer fmt.Println("receivePump Done")
	fmt.Println("receivePump started")

	for {
		if ws.closed.Load() {
			log.Println("receivePump stopping due to closed client")
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
			ws.socketMutex.Lock()
			if ws.socket == nil {
				ws.socketMutex.Unlock()
				ws.ErrChan <- errors.New("socket is nil in receivePump")
				time.Sleep(time.Second)
				continue
			}
			// ws.socket.SetReadDeadline(time.Now().Add(READ_TIMEOUT))

			ws.socketMutex.Unlock()
			_, msg, err := ws.socket.ReadMessage()

			if err != nil {
				log.Printf("Error reading message: %v", err)
				ws.ErrChan <- err
				time.Sleep(time.Second)
				continue
			}

			response := new(RespMessage)
			if err := json.Unmarshal(msg, response); err != nil {
				ws.ErrChan <- err
				continue
			}

			if len(response.Err) > 0 {
				ws.ErrChan <- fmt.Errorf("error response: %s", response.Err[0].Message)
				continue
			}

			fmt.Println("response", response)
			for _, data := range response.Data {
				if data.ImageURL == "" {
					ws.ErrChan <- errors.New("empty image URL in response")
					continue
				}

				select {
				case ws.ReceiveMsgChan <- []byte(data.ImageURL):
				default:
					ws.ErrChan <- errors.New("receive buffer is full")
				}
			}
		}
	}
}

func (ws *WSClient) errorHandler(ctx context.Context) {
	defer ws.wg.Done()
	defer fmt.Println("errorHandler Done")
	fmt.Println("errorHandler started")

	retryCount := 0
	backoffDuration := RECONNECTED_DELAY

	for {
		if ws.closed.Load() {
			log.Println("errorHandler stopping due to closed client")
			return
		}
		select {
		case <-ctx.Done():
			fmt.Println("errorHandler Done")
			return
		case err, ok := <-ws.ErrChan:
			if !ok {
				return
			}
			log.Printf("WebSocket error: %v", err)
			var netErr net.Error

			switch {
			case errors.As(err, &netErr) && netErr.Timeout():
				log.Println("Connection timeout detected")
				ws.Close(ctx)
				return
			case websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway):
				log.Println("WebSocket closed by server")
				ws.Close(ctx)
				return
			case websocket.IsUnexpectedCloseError(err, websocket.CloseAbnormalClosure, websocket.CloseServiceRestart, websocket.CloseTryAgainLater):
				log.Println("WebSocket closed unexpectedly")
				if !ws.handleReconnect(ctx, &retryCount, &backoffDuration) {
					log.Println("Error handling reconnection")
					return
				}
			default:
				log.Printf("Unhandled error: %v", err)
			}
		}
	}
}

func (ws *WSClient) Close(ctx context.Context) error {
	var closeErr error
	ws.closeOnce.Do(func() {
		log.Println("Starting WebSocket client shutdown")
		ws.closed.Store(true)

		ws.socketMutex.Lock()
		defer ws.socketMutex.Unlock()

		if ws.socket != nil {
			ws.socket.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
				time.Now().Add(time.Second),
			)

			if err := ws.socket.Close(); err != nil {
				log.Printf("Error closing WebSocket connection: %v", err)
			}
		}

		if ws.Done != nil {
			close(ws.Done)
		}

		ctx.Done()

		log.Println("WebSocket client shutdown completed")
	})
	return closeErr
}

func (ws *WSClient) handleReconnect(ctx context.Context, retryCount *int, backoffDuration *time.Duration) bool {
	if *retryCount >= MAX_RETRIES {
		log.Printf("Failed to reconnect after %d attempts", MAX_RETRIES)
		ws.Close(ctx)
		return false
	}

	retryCtx, cancel := context.WithTimeout(ctx, RECONNECT_TIMEOUT)
	defer cancel()

	if err := ws.restart(retryCtx); err != nil {
		*retryCount++
		*backoffDuration *= 2
		if *backoffDuration > 1*time.Minute {
			*backoffDuration = 1 * time.Minute
		}
		log.Printf("Reconnection attempt %d failed: %v. Next attempt in %v", *retryCount, err, *backoffDuration)

		select {
		case <-ctx.Done():
			return false
		case <-time.After(*backoffDuration):
			return true
		}
	}

	log.Println("Successfully reconnected")
	*retryCount = 0
	*backoffDuration = RECONNECTED_DELAY
	return true
}

func safeClose[T any](ch chan T) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic while closing channel: %v", r)
		}
	}()

	select {
	case _, ok := <-ch:
		if ok {
			close(ch)
		}
	default:
		close(ch)
	}
}

func (ws *WSClient) SendMsg(msg ReqMessage, ctx context.Context) error {
	if ws.closed.Load() || ws.socket == nil || ws.SendMsgChan == nil {
		ws.Start(ctx)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case ws.SendMsgChan <- msg:
		return nil
	default:
		return errors.New("send channel is full")
	}
}

func GenerateUUID() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}

// func (ws *WSClient) resizeReceiveChannel() {
// 	oldChan := ws.ReceiveMsgChan
// 	newSize := cap(oldChan) * 2
// 	if newSize > RECEIVE_CHAN_MAX_SIZE {
// 		newSize = RECEIVE_CHAN_MAX_SIZE
// 	}
// 	newChan := make(chan []byte, newSize)

// 	close(oldChan)
// 	for msg := range oldChan {
// 		newChan <- msg
// 	}

// 	ws.ReceiveMsgChan = newChan
// }

// func (ws *WSClient) clearChan() {
// 	ws.mutex.Lock()
// 	defer ws.mutex.Unlock()
// 	for {
// 		select {
// 		case <-ws.ReceiveMsgChan:
// 		default:
// 			return
// 		}
// 	}
// }
