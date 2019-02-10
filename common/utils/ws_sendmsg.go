//
//File: ws_sendmsg.go
//auth: lip
//desc:
//date:  2019-01-26
//

package utils

import (
	"errors"
	"game/common/property"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"strconv"
	"sync"

	"github.com/astaxie/beego/logs"
)

var (
	broadcast = make(chan WSMessage)
	cpool     = make(chan *Player)
)

type WechatWS struct{}

var roomSize int

var sfroom *SafeRoom
var sfpool *SafePool
var ws *WechatWS
var rwLock sync.RWMutex

func InitWS() {

	go handleMessages()
	go initWSRpc()
	go handlePlayer()

	var err error
	sfroom = newSafeRoom()
	sfpool = newSafePool()

	roomSize, err = strconv.Atoi(property.GetProperty("room::roomsize"))
	logs.Debug("房间大小", roomSize)
	if err != nil {
		logs.Debug(err)
	}
}

func initWSRpc() {
	ws = new(WechatWS)
	//注册rpc服务
	rpc.Register(ws)
	//获取tcpaddr
	// host := property.GetProperty("ws:host")
	host := "127.0.0.1:8090"
	logs.Debug("服务端地址", host)
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

func newSafeRoom() *SafeRoom {
	sfroom := new(SafeRoom)
	sfroom.SRoom = make(map[string]*Room)
	return sfroom
}

func newSafePool() *SafePool {
	sfpool := new(SafePool)
	sfpool.SPool = make(map[uint64]*Player)
	return sfpool
}

//发送数据至页面
func handleMessages() {
	for {
		wsm := <-broadcast
		r := sfroom.ReadAllSafeRoom()
		room := r[wsm.Roomid]

		if ws != nil {
			if wsm.Range != 0 { //存在指定的接收者为  单独发送消息
				// player := r.Player[wsm.ReciverId]
				// if player != nil {
				// 	if player.Conn != nil {
				// 		err := player.Conn.WriteJSON(wsm)
				// 		if err != nil {
				// 			logs.Error("client.WriteJSON error: ", err)
				// 			player.Conn.Close()
				// 			delete(r.Player, wsm.ReciverId)
				// 		}
				// 	} else {
				// 		delete(r.Player, player.Userid)
				// 		logs.Error("连接断开了,清除该用户", player.Userid)
				// 	}
				// } else {
				// 	logs.Debug("该用户已经不在房间", wsm.ReciverId)
				// }
			} else { //没有指定单独的接收者,则为广播
				for _, p := range room.Player {
					if wsm.Userid == p.Userid { //不广播给发送者
						continue
					}
					if p.Conn != nil {
						err := p.Conn.WriteJSON(wsm)
						if err != nil {
							logs.Error("client.WriteJSON error: ", err)
							p.Conn.Close()
							p.Status = 0
						}

					} else {
						p.Status = 0
						logs.Error("连接断开了", p.Userid)
					}
				}
			}
		}
	}
}

//接收玩家的加入
func handlePlayer() {
	for {
		logs.Debug("第三步")
		player := <-cpool
		if player.Conn != nil && player.Status == 1 { //正常玩家
			playerEnter(player)
		} else if player.Conn != nil && player.Status == 0 { //断线重连玩家
			room := sfroom.ReadOneSafeRoom(player.Roomid)
			if room != nil && room.Player[player.Userid] != nil {
				room.Player[player.Userid].Status = 1
				room.Player[player.Userid].Conn = player.Conn
			} else { //房间已经解散，重新加入玩家池
				playerEnter(player)
			}
		}
	}
}

//玩家加入处理
func playerEnter(player *Player) {

	// rwLock.Lock()
	// defer rwLock.Unlock()

	logs.Debug("第四步")
	logs.Debug("寻找可加入房间", player.Userid)
	var roomid string
	// var ret int

	//查找可加入房间
	if roomid = FindAvailableRoom(); roomid != "" {
		logs.Debug("找到了可加入的", roomid)
		err := enterRoom(player, roomid)
		if err != nil {
			logs.Debug("加入房间发生了错误", err)
			sfpool.WriteSafePool(player)
		}
		// ws.WSSendMsg(WSMessage{Room: room[roomid], Type: PALYER_ENTER, Msg: Player{IsAdmin: 0}, Senderid: player.Userid, ReciverId: 0}, &ret)
	} else { //创建新房间

		sfpool.WriteSafePool(player)
		players := sfpool.ReadAllSafePool()
		logs.Debug("满足条件的话会创建房间")
		if len(players) > 2 {
			logs.Debug("创建新房间")
			roomid = createNewRoom()
			// p := make(map[uint64]*Player)
			// var room Room
			// player.IsAdmin = 1
			// room.Player = p
			// player.Roomid = roomid
			// room.Player[player.Userid] = player
			// room.Roomid = roomid
			// room.Status = 1
			//创建房间
			sfroom.WriteSafeRoom(roomid)
			for k, v := range players {
				//第一个为房主
				if k == 0 {
					v.IsAdmin = 1

				}
				if err := enterRoom(v, roomid); err == nil {
					logs.Debug("加入房间", k, v)
					sfpool.DelSafePool(k)
				} else {
					logs.Error("加入房间发生错误", err)
				}
			}
			// rooms := sfroom.ReadAllSafeRoom()
			// rooms[roomid] = &room
			// ws.WSSendMsg(WSMessage{Room: room[roomid], ReciverId: player.Userid, Type: PALYER_ENTER, Msg: Player{IsAdmin: 1}}, &ret)
			// if ret != 1 {
			// 	logs.Debug("玩家加入广播消息失败")
			// }
		}
	}
	logs.Debug("玩家池中的玩家", sfpool.ReadAllSafePool())
}

//寻找可加入的房间
func FindAvailableRoom() string {
	logs.Debug("寻找可以加入的房间")
	room := sfroom.ReadAllSafeRoom()
	for roomid, avRoom := range room {
		if avRoom.Status == 1 {
			return roomid
		}
	}
	return ""
}

//创建新房间
func createNewRoom() string {
	var roomid string
	for {
		roomid = RandomString(8)
		logs.Debug("随机生成房间号", roomid)
		if sfroom.ReadOneSafeRoom(roomid) == nil {
			logs.Debug("哪里报错", roomid)
			break
		}
	}
	return roomid
}

//玩家加入房间
func enterRoom(player *Player, roomid string) error {
	logs.Debug("玩家加入房间", roomid)
	room := sfroom.ReadOneSafeRoom(roomid)
	logs.Debug("房间信息", room)
	if room != nil {
		logs.Debug("房间存在奥")
		if len(room.Player) <= roomSize {
			room.Player[player.Userid] = player
			if len(room.Player) >= roomSize {
				logs.Debug("房间已经满了", "已经不可以再加入了", room.Player)
				room.Status = 0
			}
			player.Roomid = roomid
			return nil
		}
	}
	return errors.New("该房间不可加入")
}

//移除房间
func removeRoom(roomid string) {
	sfroom.DelSafeRoom(roomid)
}

//添加玩家
func WSAdd(player *Player) {
	logs.Debug("第二步")
	if player.Conn != nil {
		logs.Debug("连接状态", "不为空")

		if player.Roomid != "" {
			player.Status = 0
		}
		player.Status = 1
		cpool <- player
	}
}

//移除玩家
func RemovePalyer(roomid string, userid uint64) {

	if roomid != "" { //房间移除
		room := sfroom.ReadOneSafeRoom(roomid)
		logs.Debug("这个房间有人退出", room, roomid)
		// var ret int
		if room != nil {
			if room.Status == 1 { //游戏未开始，玩家退出或掉线
				player := room.Player[userid]
				if player != nil {
					if player.Conn != nil {
						player.Conn.Close()
						logs.Debug("断开连接")
					}
					logs.Debug("这个人退出房间", userid)
					delete(room.Player, userid)
					//房主退出，移交房主权限
					if player.IsAdmin == 1 {
						logs.Debug("房主退出")
						//房间为空
						if len(room.Player) == 0 {
							logs.Debug("房间已经空了，移除")
							//移除房间
							removeRoom(roomid)
						} else {
							for _, p := range room.Player {
								p.IsAdmin = 1
								logs.Debug("这个人成了房主", p.Userid)
								break
							}
						}
					}
				}
			} else if room.Status == 0 { //游戏已开始，玩家掉线或退出
				player := room.Player[userid]
				if player.Conn != nil {
					player.Conn.Close()
				}
				player.Status = 0
			}
		}
	} else {

	}
	logs.Debug(userid, "remove ")
}

//发送消息
func (w *WechatWS) WSSendMsg(wsm WSMessage, ret *int) error {
	logs.Debug("进来发送消息")

	room := sfroom.ReadOneSafeRoom(wsm.Roomid)
	if room != nil {
		logs.Debug("房间号", wsm.Roomid)
		*ret = 1
		if room.Player == nil {
			logs.Debug("房间用户为空")
			logs.Error(wsm.Roomid, "room has no one")
			*ret = 0
			removeRoom(wsm.Roomid)
			return nil
		}
		broadcast <- wsm
	} else {
		logs.Debug("这个房间不存在", wsm.Roomid)
	}
	return nil
}

//游戏开始修改房间状态
func (w *WechatWS) StartGame(roomid string, ret *int) error {
	room := sfroom.ReadOneSafeRoom(roomid)
	logs.Debug("这个房间即将关闭", roomid)
	*ret = 0
	if room != nil {
		logs.Debug("游戏已经开始，无法通过房间匹配加入")
		room.Status = 0
		// ws.WSSendMsg(WSMessage{Room: room[roomid], Type: GAME_START}, ret)
		if *ret != 1 {
			logs.Debug("游戏开始消息发送失败")
		}
		return nil
	}
	return nil
}

//玩家退出游戏
func (w *WechatWS) ExitGame(player Player, ret *int) error {
	logs.Debug("有人退出", player.Roomid, player.Userid)
	*ret = 1
	RemovePalyer(player.Roomid, player.Userid)
	return nil
}

//游戏结束释放房间
func (w *WechatWS) GameOver(roomid string) {

	room := sfroom.ReadOneSafeRoom(roomid)
	for u, p := range room.Player {
		if p.Conn != nil {
			p.Conn.Close()
		}
		delete(room.Player, u)
	}
	removeRoom(roomid)
}

/****************************************************************************/
