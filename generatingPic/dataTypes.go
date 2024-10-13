package generatingPic

import (
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
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
	Scheduler      string   `json:"scheduler,omitempty"`
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