//
//File: ws_sendmsg_rpc.go
//auth: lip
//desc:
//date:  2019-01-26
//

package utils

import (
	"fmt"
	"net/rpc"
	"net/rpc/jsonrpc"

	"github.com/astaxie/beego/logs"
)

var wsrpc *rpc.Client

func InitWSClient() {
	//连接远程rpc服务
	//这里使用jsonrpc.Dial
	var err error
	// host := property.GetProperty("ws::host")
	var host = "localhost:8090"
	wsrpc, err = jsonrpc.Dial("tcp", host)
	logs.Debug("客户端初始化", wsrpc)
	if err != nil {
		logs.Error("wsrpc init error", err)
	}
}

// WSSendMsg
// @Description: description
// @return: returns 返回值1表示不在线
// @Author: lip
// @Date: 2019-01-26
func WSSendMsg(message WSMessage) int {

	ret := 0
	//调用远程方法
	//注意第三个参数是指针类型
	fmt.Println("wswendmsg:", wsrpc, message.Room, message.Senderid, message.Msg, message.Type)
	// WSMessage{Roomid: message.Roomid, Senderid: message.Senderid, Type: message.Type, Msg: message.Msg, ReciverId: message.ReciverId}
	err2 := wsrpc.Call("WechatWS.WSSendMsg", message, &ret)
	if err2 != nil {
		logs.Error("发送发生错误", err2)
	}

	logs.Debug("wssend :", ret)
	return ret

}

//开始游戏
func StartGame(roomid string) int {
	ret := 0
	err3 := wsrpc.Call("WechatWS.StartGame", roomid, &ret)
	if err3 != nil {
		logs.Error("开始发生错误", err3)
	}
	logs.Debug("start  game", ret)
	return ret
}

//玩家退出
func ExitGame(player Player) int {
	ret := 0
	err4 := wsrpc.Call("WechatWS.ExitGame", player, &ret)
	if err4 != nil {
		logs.Error("开始发生错误", err4)
	}
	logs.Debug("exit  game", ret)
	return ret
}
