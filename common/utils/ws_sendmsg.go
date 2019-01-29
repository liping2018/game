//
//File: ws_sendmsg.go
//auth: lip
//desc:
//date:  2019-01-26
//

package utils

import (
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"sync"

	"github.com/astaxie/beego/logs"
)

var (
	broadcast = make(chan WSMessage)
	room      = make(map[string]*Room)
)

type WechatWS struct{}

var rwLock sync.RWMutex

func InitWS() {
	go handleMessages()
	go initWSRpc()
}

var ws *WechatWS

func initWSRpc() {
	ws = new(WechatWS)
	//注册rpc服务
	rpc.Register(ws)
	//获取tcpaddr
	var host = "localhost:8090"
	// host := property.GetProperty("ws::host")
	tcpaddr, err := net.ResolveTCPAddr("tcp4", host)
	logs.Debug("服务端初始化", tcpaddr)
	if err != nil {
		logs.Error(err)
	}
	//监听端口
	tcplisten, err2 := net.ListenTCP("tcp", tcpaddr)
	if err2 != nil {
		logs.Error(err2)
	}
	for {
		conn, err3 := tcplisten.Accept()
		if err3 != nil {
			continue
		}
		//使用goroutine单独处理rpc连接请求
		//这里使用jsonrpc进行处理
		go jsonrpc.ServeConn(conn)
	}

}

//发送数据至页面
func handleMessages() {
	for {
		wsm := <-broadcast
		ws := room[wsm.Room.Roomid]
		if ws != nil {
			for _, p := range ws.Player {
				if p.Conn != nil {
					err := p.Conn.WriteJSON(wsm)
					if err != nil {
						logs.Error("client.WriteJSON error: ", err)
						p.Conn.Close()
						delete(ws.Player, p.Userid)
					}

				} else {
					logs.Error("连接断开了")
				}
			}
		}
	}
}

//寻找可加入的房间
func FindAvailableRoom() string {
	for roomid, avRoom := range room {
		if avRoom.Status == 1 {
			return roomid
		}
	}
	return ""
}

//创建新房间
func CreateNewRoom() string {
	var roomid string
	for {
		roomid = RandomString(8)
		logs.Debug("随机生成房间号", roomid)
		if room[roomid] == nil {
			logs.Debug("哪里报错", room[roomid])
			break
		}
	}
	return roomid
}

//移除房间
func RemoveRoom(roomid string) {
	delete(room, roomid)
}

//添加玩家
func WSAdd(player *Player) {

	rwLock.Lock()

	logs.Debug("寻找可加入房间")
	var roomid string
	//查找可加入房间
	if roomid = FindAvailableRoom(); roomid != "" {
		logs.Debug("找到了可加入的", roomid)
		player.Roomid = roomid
		room[roomid].Player[player.Userid] = player
		ws.WSSendMsg(WSMessage{Room: room[roomid], Type: PALYER_ENTER, Msg: Player{IsAdmin: 0}, ReciverId: player.Userid}, nil)
	} else { //创建新房间
		logs.Debug("创建新房间")
		roomid = CreateNewRoom()
		p := make(map[uint64]*Player)
		var r Room
		player.IsAdmin = 1
		r.Player = p
		player.Roomid = roomid
		r.Player[player.Userid] = player
		r.Roomid = roomid
		r.Status = 1
		room[roomid] = &r
		ws.WSSendMsg(WSMessage{Room: room[roomid], ReciverId: player.Userid, Type: PALYER_ENTER, Msg: Player{IsAdmin: 1}}, nil)
	}
	rwLock.Unlock()
	logs.Debug("socket created ")
}

//移除玩家
func RemovePalyer(roomid string, userid uint64) {
	rwLock.Lock()
	ro := room[roomid]
	logs.Debug("这个房间有人退出", ro, roomid)
	if ro != nil {
		player := ro.Player[userid]
		if player != nil {
			if player.Conn != nil {
				player.Conn.Close()
				logs.Debug("断开连接")
			}
			logs.Debug("这个人退出房间", userid)
			delete(ro.Player, userid)
			ws.WSSendMsg(WSMessage{Room: room[roomid], Senderid: player.Userid, Type: PLAYER_EXIT, Msg: Player{}, ReciverId: 0}, nil)
			//房主退出，移交房主权限
			if player.IsAdmin == 1 {
				logs.Debug("房主退出")
				//房间为空
				if len(room[roomid].Player) == 0 {
					logs.Debug("房间已经空了，移除")
					//移除房间
					RemoveRoom(roomid)
				} else {
					for _, p := range room[roomid].Player {
						p.IsAdmin = 1
						logs.Debug("这个人成了房主", p.Userid)
						ws.WSSendMsg(WSMessage{Room: room[roomid], Senderid: player.Userid, Type: PLAYER_EXIT, Msg: p, ReciverId: p.Userid}, nil)
						break
					}
				}
			}
		}
	}
	rwLock.Unlock()
	logs.Debug(userid, "remove ")
}

//发送消息
func (w *WechatWS) WSSendMsg(wsm WSMessage, ret *int) error {
	logs.Debug("进来发送消息")
	r := room[wsm.Room.Roomid]
	if r != nil {
		logs.Debug("房间号", wsm.Room.Roomid)
		// *ret = 0
		if r.Player == nil {
			logs.Debug("房间用户为空")
			logs.Error(wsm.Room.Roomid, "room has no one")
			// *ret = 1
			RemoveRoom(wsm.Room.Roomid)
			return nil
		}
		broadcast <- wsm
	} else {
		logs.Debug("这个房间不存在", wsm.Room.Roomid)
	}
	return nil
}

//游戏开始修改房间状态
func (w *WechatWS) StartGame(roomid string, ret *int) error {
	r := room[roomid]
	logs.Debug("这个房间即将关闭", roomid)
	if r != nil {
		logs.Debug("游戏已经开始，无法加入")
		r.Status = 0
		ws.WSSendMsg(WSMessage{Room: room[roomid], Type: GAME_START}, nil)
		return nil
	}
	return nil
}

//玩家退出游戏
func (w *WechatWS) ExitGame(player Player, ret *int) error {
	logs.Debug("有人退出", player.Roomid, player.Userid)
	RemovePalyer(player.Roomid, player.Userid)
	return nil
}

//游戏结束释放房间
func (w *WechatWS) GameOver(roomid string) {
	r := room[roomid]
	for u, p := range r.Player {
		if p.Conn != nil {
			p.Conn.Close()
		}
		delete(r.Player, u)
	}
	RemoveRoom(roomid)
}
