package generatingPic

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	URL                   = "wss://ws-api.runware.ai/v1"
	SEND_CHAN_MAX_SIZE    = 128
	RECEIVE_CHAN_MAX_SIZE = 128
	TIMEOUT               = 60 * time.Second
	RECONNECTED_DELAY     = 3 * time.Second
	MAX_RETRIES           = 3
	WRITE_TIMEOUT         = 10 * time.Second
	READ_TIMEOUT          = 30 * time.Second
)

type AuthCredentials struct {
	TaskType string `json:"taskType"`
	ApiKey   string `json:"apiKey"`
}

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
	Massage   string `json:"message"`
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
	auth           AuthCredentials
	url            string
	socket         *websocket.Conn
	SendMsgChan    chan ReqMessage
	ReceiveMsgChan chan []byte
	ErrChan        chan error
	Done           chan struct{}
	mutex          sync.Mutex
}

func NewWSClient(apiKey string, userID uint) *WSClient {
	return &WSClient{
		User: struct {
			ID   uint
			UUID string
		}{ID: userID, UUID: ""},
		auth:           AuthCredentials{"authentication", apiKey},
		url:            URL,
		SendMsgChan:    make(chan ReqMessage, SEND_CHAN_MAX_SIZE),
		ReceiveMsgChan: make(chan []byte, 16),
		ErrChan:        make(chan error),
		Done:           make(chan struct{}),
	}
}

func (ws *WSClient) connect() error {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	socket, _, err := websocket.DefaultDialer.Dial(ws.url, nil)
	if err != nil {
		return err
	}
	ws.socket = socket
	authCredentials := []AuthCredentials{ws.auth}
	jsonAuth, err := json.Marshal(authCredentials)
	if err != nil {
		return err
	}

	err = ws.socket.WriteMessage(websocket.TextMessage, jsonAuth)
	if err != nil {
		return err
	}

	respMsg := new(RespMessage)
	err = ws.socket.ReadJSON(respMsg)
	if err != nil {
		return err
	}

	if respMsg.Err != nil {
		return errors.New("Authentication failed: " + respMsg.Err[0].Massage)
	}

	sessionUUID := respMsg.Data[0].ConnectionSessionUUID
	if sessionUUID == "" {
		return errors.New("Authentication failed: connectionSessionUUID not found")
	}

	ws.User.UUID = sessionUUID
	log.Printf("UserID:%d was authenticated with session UUID: %s", ws.User.ID, sessionUUID)
	return nil
}

func (ws *WSClient) Start() {
	err := ws.connect()
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	go ws.sendPump()
	go ws.receivePump()
	go ws.errorHandler()
}

func (ws *WSClient) sendPump() {
	for {
		select {
		case <-ws.Done:
			return
		default:
			ws.socket.SetWriteDeadline(time.Now().Add(WRITE_TIMEOUT))

			req := <-ws.SendMsgChan
			formattedReq := []ReqMessage{req}
			jsonMsg, err := json.Marshal(formattedReq)
			if err != nil {
				ws.ErrChan <- err
			}
			fmt.Println(string(jsonMsg))
			err = ws.socket.WriteMessage(websocket.TextMessage, jsonMsg)
			if err != nil {
				ws.ErrChan <- err
			}
		}
	}
}

func (ws *WSClient) receivePump() {
	for {
		select {
		case <-ws.Done:
			return
		default:
			ws.socket.SetReadDeadline(time.Now().Add(READ_TIMEOUT))

			response := new(RespMessage)
			err := ws.socket.ReadJSON(response)
			if err != nil {
				ws.ErrChan <- err
			}

			if response.Err != nil {
				ws.ErrChan <- errors.New("Error response: " + response.Err[0].Massage)
			}
			if len(response.Data) == 0 {
				ws.ErrChan <- errors.New("Empty response")
			}
			for i := range len(response.Data) {
				ImgUrl := response.Data[i].ImageURL
				if ImgUrl == "" {
					ws.ErrChan <- errors.New("Empty image URL")
				}
				select {
				case ws.ReceiveMsgChan <- []byte(ImgUrl):
				default:
					log.Println("Warning: receive buffer full, resizing...")
					if len(ws.ReceiveMsgChan) < RECEIVE_CHAN_MAX_SIZE {
						ws.resizeReceiveChannel()
					} else {
						ws.ErrChan <- errors.New("Receive buffer is full")
						ws.clearChan()
					}
				}
			}
		}
	}
}

func (ws *WSClient) errorHandler() {
	retryCount := 0

	for {
		select {
		case <-ws.Done:
			return
		default:
			err := <-ws.ErrChan
			log.Printf("WebSocket error: %v", err)
			if websocket.IsCloseError(err, websocket.CloseAbnormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseNoStatusReceived) {
				log.Println("Connection closed, attempting to reconnect...")
				for retryCount < MAX_RETRIES {
					time.Sleep(WRITE_TIMEOUT)
					err := ws.connect()
					if err == nil {
						log.Println("Successfully reconnected")
						retryCount = 0
						break
					}
					retryCount++
					log.Printf("Reconnection attempt %d failed: %v", retryCount, err)
				}
				if retryCount >= MAX_RETRIES {
					log.Printf("Failed to reconnect after %d attempts", MAX_RETRIES)
					ws.Close()
					return
				}
			} else if err == io.EOF {
				log.Println("Connection closed by server")
				ws.Close()
				return
			} else {
				log.Printf("Unhandled WebSocket error: %v", err)
				return
			}
		}
	}
}

func (ws *WSClient) Close() {
	select {
	case <-ws.Done:
		return
	default:
		close(ws.Done)
	}

	if ws.socket != nil {
		closeChan := make(chan error, 1)
		go func() {
			closeChan <- ws.socket.WriteMessage(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			)
		}()

		select {
		case err := <-closeChan:
			if err != nil {
				log.Printf("Error while closing WebSocket: %v", err)
			}
		case <-time.After(TIMEOUT):
			log.Println("Timeout while closing WebSocket")
		}

		err := ws.socket.Close()
		if err != nil {
			log.Printf("Error while closing WebSocket: %v", err)
			return
		}

		ws.socket = nil
	}
}

func (ws *WSClient) resizeReceiveChannel() {
	oldChan := ws.ReceiveMsgChan
	newSize := cap(oldChan) * 2
	if newSize > RECEIVE_CHAN_MAX_SIZE {
		newSize = RECEIVE_CHAN_MAX_SIZE
	}
	newChan := make(chan []byte, newSize)

	close(oldChan)
	for msg := range oldChan {
		newChan <- msg
	}

	ws.ReceiveMsgChan = newChan
}

func (ws *WSClient) clearChan() {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	for {
		select {
		case <-ws.ReceiveMsgChan:
		default:
			return
		}
	}
}

func GenerateUUID() string {
	hexUUID := uuid.New().String()
	hexUUID = strings.ReplaceAll(hexUUID, "-", "")
	return hexUUID
}
