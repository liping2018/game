package controllers

import (
	"encoding/json"
	"game/common/utils"

	"github.com/astaxie/beego/logs"

	"github.com/astaxie/beego"
)

type MainController struct {
	beego.Controller
}

const (
	RSP_SUCCESS      = 1 //请求的命令成功处理
	RSP_FAIL         = 0
	RSP_FAIL_PARAMS  = RSP_FAIL - 1 //参数不完整或格式错误
	RSP_KEY_NOTEXIST = RSP_FAIL - 2 //key不存在
)

type RspData struct {
	Cmd     string `json:"cmd"`
	Retcode int    `json:"retcode"`
}

func (c *MainController) Get() {
	c.Data["Website"] = "beego.me"
	c.Data["Email"] = "astaxie@gmail.com"
	c.TplName = "index.html"
}
func (this *MainController) JsonOutput(rspdata interface{}) {
	this.Data["json"] = rspdata
	out, err := json.Marshal(rspdata)
	if err != nil {
		logs.Error(err)
	} else {
		logs.Debug("respose = " + string(out))
	}
	this.ServeJSON()
}

//玩家发送消息
func (c *MainController) Message() {

	var rspdata RspData
	// rspdata.Retcode = RSP_FAIL
	// // var reciverid uint64
	// // roomid := c.GetString("roomid")
	// // reciverid, err := c.GetUint64("reciverid")
	// if err != nil {
	// 	reciverid = 0
	// }
	// cmd := c.GetString("cmd")
	// rspdata.Cmd = cmd
	// // senderid, err := c.GetUint64("senderid")
	// if err != nil {
	// 	logs.Error(err)
	// } else {
	// 	// msg := c.GetString("msg")
	// 	var room utils.Room
	// 	room.Roomid = roomid
	// 	// ret := utils.WSSendMsg(utils.WSMessage{Room: &room, Senderid: senderid, ReciverId: reciverid, Type: utils.MESSAGE, Msg: msg})
	// 	// if ret == 1 {
	// 	// 	rspdata.Retcode = RSP_SUCCESS
	// 	// }
	// }
	c.JsonOutput(rspdata)
}

//玩家退出
func (c *MainController) PlayerExit() {
	var rspdata RspData
	var player utils.Player
	rspdata.Retcode = RSP_FAIL
	roomid := c.GetString("roomid")
	userid, err := c.GetUint64("userid")
	player.Roomid = roomid
	player.Userid = userid
	cmd := c.GetString("cmd")
	rspdata.Cmd = cmd
	if err != nil {
		logs.Error(err)
	} else {
		utils.ExitGame(player)
		rspdata.Retcode = RSP_SUCCESS
	}
	c.JsonOutput(rspdata)
}

func (c *MainController) Start() {

	var rspdata RspData

	roomid := c.GetString("roomid")
	cmd := c.GetString("cmd")
	rspdata.Cmd = cmd
	rspdata.Retcode = RSP_FAIL
	ret := utils.StartGame(roomid)
	if ret == 1 {
		rspdata.Retcode = RSP_SUCCESS
	}
	c.JsonOutput(rspdata)
}
