package generatingPic

import (
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/net/context"
)

const (
	URL              = "wss://ws-api.runware.ai/v1"
	SEND_BUF_SIZE    = 10
	RECEIVE_BUF_SIZE = 10
	TIMEOUT          = 60 * time.Second
	reconnectDelay   = 5 * time.Second
	writeTimeout     = 10 * time.Second
	readTimeout      = 120 * time.Second
)

type AuthCredentials struct {
	TaskType string `json:"taskType"`
	ApiKey   string `json:"apiKey"`
}

type Message struct {
	PositivePrompt string   `json:"positivePrompt"`
	Model          string   `json:"model"`
	Steps          int      `json:"steps"`
	Width          int      `json:"width"`
	Height         int      `json:"height"`
	NumberResults  int      `json:"numberResults"`
	OutputType     []string `json:"outputType"`
	TaskType       string   `json:"taskType"`
	TaskUUID       string   `json:"taskUUID"`
}

type WSClient struct {
	auth           AuthCredentials
	url            string
	socket         *websocket.Conn
	SendMsgChan    chan Message
	ReceiveMsgChan chan []byte
	ErrChan        chan error
	Done           chan struct{}
}

func NewWSClient(apiKey string) *WSClient {
	return &WSClient{
		auth:           AuthCredentials{"authentication", apiKey},
		url:            URL,
		SendMsgChan:    make(chan Message),
		ReceiveMsgChan: make(chan []byte, RECEIVE_BUF_SIZE),
		ErrChan:        make(chan error),
		Done:           make(chan struct{}),
	}
}

func (ws *WSClient) connect() error {
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

	msg := new(interface{})
	err = ws.socket.ReadJSON(msg)
	if err != nil {
		return err
	}

	msgValue, ok := (*msg).(map[string]interface{})
	if !ok {
		return errors.New("invalid message format: expected map[string]interface{}")
	}
	msgData, ok := msgValue["data"].([]interface{})
	if !ok {
		return errors.New("invalid data format: expected array")
	}
	if len(msgData) == 0 {
		return errors.New("no data received")
	}

	dataMap, ok := msgData[0].(map[string]interface{})
	if !ok {
		return errors.New("invalid data format: expected map")
	}

	sessionUUID, ok := dataMap["connectionSessionUUID"]
	if !ok {
		return errors.New("Authentication failed: connectionSessionUUID not found")
	}

	log.Printf("Authenticated with session UUID: %s", sessionUUID)
	return nil
}

func (ws *WSClient) Start(ctx context.Context) {
	err := ws.connect()
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	go ws.sendPump(ctx)
	go ws.receivePump(ctx)
	go ws.errorHandler(ctx)
}

func (ws *WSClient) sendPump(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			ws.socket.SetWriteDeadline(time.Now().Add(writeTimeout))
			req := <-ws.SendMsgChan
			reqData := []Message{req}

			jsonMsg, err := json.Marshal(reqData)
			if err != nil {
				ws.ErrChan <- err
			}
			err = ws.socket.WriteMessage(websocket.TextMessage, jsonMsg)
			if err != nil {
				ws.ErrChan <- err
			}
		}
	}
}

func (ws *WSClient) receivePump(ctx context.Context) {
	defer func() {
		log.Println("Stopping receivePump due to WebSocket closure.")
		ws.socket.Close()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			ws.socket.SetReadDeadline(time.Now().Add(readTimeout))
			message := new(map[string]interface{})
			err := ws.socket.ReadJSON(message)
			if err != nil {
				ws.ErrChan <- err
				return
			}
			recievedData, ok := (*message)["data"].([]interface{})
			if !ok {
				ws.ErrChan <- errors.New("invalid data format: expected array")
				return
			}

			dataMap, ok := recievedData[0].(map[string]interface{})
			if !ok {
				ws.ErrChan <- errors.New("invalid data format: expected map")
				return
			}

			ImgUrl, ok := dataMap["imageURL"]
			if !ok {
				ws.ErrChan <- errors.New("imageURL not found")
				return
			}

			select {
			case ws.ReceiveMsgChan <- []byte(ImgUrl.(string)):
			default:
				log.Println("Warning: receive buffer full, dropping message")
				return
			}
		}
	}
}

func (ws *WSClient) errorHandler(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-ws.ErrChan:
			log.Printf("WebSocket error: %v", err)
			ws.socket.Close()
		}
	}
}

func (ws *WSClient) Close() {
	close(ws.Done)
	if ws.socket != nil {
		ws.socket.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
		ws.socket.Close()
	}
}
