package controllers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/astaxie/beego/logs"

	"game/common/utils"

	"github.com/astaxie/beego"
	"github.com/gorilla/websocket"
)

type MainController struct {
	beego.Controller
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (c *MainController) Get() {
	logs.Debug("第一步")
	var player utils.Player
	ws, err := upgrader.Upgrade(c.Ctx.ResponseWriter, c.Ctx.Request, nil)
	if err != nil {
		log.Fatal(err)
	}
	userid, err := c.GetUint64("userid")
	player.Userid = userid
	player.Conn = ws
	utils.WSAdd(&player)

	for {
		_, p, err := ws.ReadMessage()

		if err != nil {
			log.Printf("页面可能断开啦 ws.ReadJSON error: %v", err)
			utils.RemovePalyer(player.Roomid, userid)
			break
		} else {
			fmt.Println("接受到从页面上反馈回来的信息 ", string(p))

		}
	}
	c.StopRun()
}
