//File: WSMessage.go
//auth: lip
//desc:
//date:  2019-01-26
//

package utils

import (
	"sync"

	"github.com/gorilla/websocket"
)

//传递信息结构体
type WSMessage struct {
	Room      *Room       `json:"room"`      //房间号
	Senderid  uint64      `json:"senderid"`  //发送者编号
	Type      string      `json:"type"`      //消息类型
	Msg       interface{} `json:"msg"`       //消息主体
	ReciverId uint64      `json:"reciverid"` //接收者编号
}

const (
	PALYER_ENTER = "enter"
	PLAYER_EXIT  = "exit"
	GAME_START   = "start"
	MESSAGE      = "message"
)

//玩家结构体
type Player struct {
	Userid  uint64          `json:"userid"`
	Roomid  string          `json:"roomid"`
	Conn    *websocket.Conn `json:"conn"`
	IsAdmin int             `json:"isadmin"`
}

//房间结构体
type Room struct {
	Roomid string             `json:"roomid"`
	Player map[uint64]*Player `json:"player"`
	Status int                `json:status`
}

type SafeRoom struct {
	sync.RWMutex
	Room
}
