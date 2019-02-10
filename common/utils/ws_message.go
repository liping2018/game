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
	Roomid string      `json:"roomid"` //房间编号
	Userid uint64      `json:"userid"` //发送者编号
	Type   string      `json:"type"`   //消息类型
	Msg    interface{} `json:"msg"`    //消息主体
	Range  int         `json:"range"`  //接收范围
}

//接收者
const (
	All_PLAYER  = 0 //给所有玩家发送消息
	TEAM_PLAYER = 1 //给队友发送消息
	ONE_PLAYER  = 2 //给单独一个玩家发送消息
)

const (
	PALYER_ENTER = "enter"   //玩家进入
	PLAYER_EXIT  = "exit"    //玩家退出
	GAME_START   = "start"   //游戏开始
	GAME_OVER    = "over"    //游戏结束
	MESSAGE      = "message" //游戏消息
)

//玩家结构体
type Player struct {
	Userid  uint64          `json:"userid"`  //用户编号
	Roomid  string          `json:"roomid"`  //房间编号
	Conn    *websocket.Conn `json:"conn"`    //websocket连接
	Status  int             `json:"status"`  //玩家状态
	IsAdmin int             `json:"isadmin"` //玩家身份
}

//房间结构体
type Room struct {
	Roomid string             `json:"roomid"` //房间编号
	Player map[uint64]*Player `json:"player"` //用户集合
	Status int                `json:status`   //房间状态
	Size   int                `json:"size"`   //房间大小
}

//房间
type SafeRoom struct {
	RMutex sync.RWMutex
	SRoom  map[string]*Room
}

//用户池子
type SafePool struct {
	SPool  map[uint64]*Player
	RMutex sync.RWMutex
}

//读取所有房间信息
func (sf *SafeRoom) ReadAllSafeRoom() map[string]*Room {
	sf.RMutex.RLock()
	defer sf.RMutex.RUnlock()
	return sf.SRoom
}

//读取一个房间信息
func (sf *SafeRoom) ReadOneSafeRoom(roomid string) *Room {
	sf.RMutex.RLock()
	defer sf.RMutex.RUnlock()
	room, ok := sf.SRoom[roomid]
	if ok {
		return room
	}
	return nil
}

//删除房间信息
func (sf *SafeRoom) DelSafeRoom(roomid string) {
	sf.RMutex.Lock()
	defer sf.RMutex.Unlock()
	if _, ok := sf.SRoom[roomid]; ok {
		delete(sf.SRoom, roomid)
	}
}

//写入房间信息
func (sf *SafeRoom) WriteSafeRoom(roomid string) {
	sf.RMutex.Lock()
	defer sf.RMutex.Unlock()
	room := new(Room)
	room.Roomid = roomid
	room.Player = make(map[uint64]*Player)
	room.Status = 1
	sf.SRoom[roomid] = room
}

//获取用户池数据
func (sp *SafePool) ReadAllSafePool() map[uint64]*Player {
	sp.RMutex.RLock()
	defer sp.RMutex.RUnlock()
	return sp.SPool
}

func (sp *SafePool) ReadOneSafePool(userid uint64) *Player {
	sp.RMutex.RLock()
	defer sp.RMutex.RUnlock()
	player, ok := sp.SPool[userid]
	if ok {
		return player
	}
	return nil
}

//删除用户池数据
func (sp *SafePool) DelSafePool(userid uint64) {
	sp.RMutex.Lock()
	defer sp.RMutex.Unlock()
	if _, ok := sp.SPool[userid]; ok {
		delete(sp.SPool, userid)
	}
}

//增加用户池数据
func (sp *SafePool) WriteSafePool(player *Player) {
	sp.RMutex.Lock()
	defer sp.RMutex.Unlock()
	sp.SPool[player.Userid] = player
}
