//
//File: ws_sendmsg.go
//auth: lip
//desc:
//date:  2019-01-26
//

package utils

import (
	"game/common/property"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"strconv"
	"sync"
	"time"

	"github.com/robfig/cron"

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
	// go handlePlayer()
	go initMatch()

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

//定时匹配
func initMatch() {
	c := cron.New()
	//一秒匹配一次
	c.AddFunc("0/1 * * * * ?", func() {
		logs.Debug("开始一次匹配")
		go matchPlayer()
	})
	c.Start()
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

		switch wsm.Type {

		case PLAYER_EXIT:
			break
		case GAME_START:
			break
		case GAME_OVER:
			gameSettle(wsm) //游戏结算
			break
		case MESSAGE:

			switch wsm.Range {

			case All_PLAYER: //广播全部玩家
				room := sfroom.ReadOneSafeRoom(wsm.Roomid)
				for _, v := range room.Player {
					if wsm.Userid != v.Userid && v.Status == 1 {
						err := v.Conn.WriteJSON(wsm)
						if err != nil { //玩家掉线
							logs.Debug("玩家掉线", v.Userid)
							v.Status = 0
						}
					}
				}
				break
			case TEAM_PLAYER: //广播队友玩家
				room := sfroom.ReadOneSafeRoom(wsm.Roomid)
				player := room.Player[wsm.Userid]
				for _, v := range room.Player {
					if v.Status == 1 && wsm.Userid != v.Userid {
						if player.Team == v.Team {
							err := v.Conn.WriteJSON(wsm)
							if err != nil { //玩家掉线
								logs.Debug("玩家掉线", v.Userid)
								v.Status = 0
							}
						}
					}
				}
				break
			case ONE_PLAYER: //发送给其中一个玩家
				break
			default:
				logs.Debug("消息范围错误")
			}
			break
		default:
			logs.Debug("消息类型错误")

		}
	}
}

//游戏结束结算
func gameSettle(wsm WSMessage) {

	room := sfroom.ReadOneSafeRoom(wsm.Roomid)
	room.Point = append(room.Point, wsm.Msg)
	var count = 0
	for _, v := range room.Player {
		if v.Status == 1 {
			count++
		}
	}
	if len(room.Point) == count {
		//发送结算消息
		for _, v := range room.Player {
			if v.Status == 1 {
				err := v.Conn.WriteJSON(room.Point)
				if err != nil {
					logs.Debug("发送消息失败")
				}
			}
		}
	}
	removeRoom(room.Roomid)
}

//玩家匹配
func matchPlayer() {
	players := sfpool.ReadAllSafePool()
	logs.Debug("进入匹配", players)
	gplayers := make(map[int64][]*Player)

	//按rank值分组
	for _, v := range players {
		logs.Debug("上一步", v)
		// logs.Debug("这里报错吗", gplayers[v.Rank].MPool)
		logs.Debug("这儿呢", gplayers[v.Rank])
		gplayers[v.Rank] = append(gplayers[v.Rank], v)
	}
	logs.Debug("第一步")
	//找出每个组中匹配时间最长的玩家进行匹配
	for _, gv := range gplayers {
		var continueMatch = true
		for {
			if !continueMatch {
				break
			}
			var oldest *Player = nil
			for _, mv := range gv {
				logs.Debug("玩家信息", mv)

				if oldest == nil {
					oldest = mv
				} else if mv.WaitTime < oldest.WaitTime {
					oldest = mv
				}
			}
			logs.Debug("等待时间最久的", oldest)
			if oldest == nil {
				break
			}
			now := time.Now().Unix()
			//按照等待时间扩大匹配范围
			var waittime int64 = ((now - oldest.WaitTime) / 1000)
			var min = (oldest.Rank - waittime)
			if min < 0 {
				min = 0
			}

			var max = oldest.Rank + waittime
			var gplayers2 []*Player
			logs.Debug("这玩意是不是出错了", gplayers2)
			for _, gmv := range players {

				if gmv.Rank <= max && gmv.Rank >= min {
					gplayers2 = append(gplayers2, gmv)
					//将玩家从匹配池删除
					sfpool.DelSafePool(gmv.Userid)
				}
				//
				if len(gplayers2) >= roomSize {
					break
				}
			}
			logs.Debug("房间和匹配认数的大小", len(gplayers2), roomSize)
			//匹配完成
			if len(gplayers2) == roomSize {
				//开房间
				logs.Debug("匹配成功，创建房间开始游戏")
				room := createNewRoom()
				for _, v := range gplayers2 {
					if room != nil {
						room.Player[v.Userid] = v
					}
				}
			} else { //匹配失败
				//把玩家放回匹配池
				continueMatch = false
				logs.Debug("玩家匹配失败，重新放回匹配池")
				for _, v := range gplayers2 {
					sfpool.WriteSafePool(v)
				}
			}

		}
	}
}

//创建新房间
func createNewRoom() *Room {
	var roomid string
	for {
		roomid = RandomString(8)
		logs.Debug("随机生成房间号", roomid)
		if sfroom.ReadOneSafeRoom(roomid) == nil {
			logs.Debug("哪里报错", roomid)
			break
		}
	}
	room := sfroom.ReadOneSafeRoom(roomid)
	return room
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
		if player.Conn != nil && player.Status == 1 { //正常玩家
			sfpool.WriteSafePool(player)
		} else if player.Conn != nil && player.Status == 0 { //断线重连玩家
			room := sfroom.ReadOneSafeRoom(player.Roomid)
			if room != nil && room.Player[player.Userid] != nil {
				room.Player[player.Userid].Status = 1
				room.Player[player.Userid].Conn = player.Conn
			} else { //房间已经解散，重新加入玩家池
				sfpool.WriteSafePool(player)
			}
		}
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
