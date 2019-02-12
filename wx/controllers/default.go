package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

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
	data := c.GetString("data")
	err = c.ParseHttpReq(data, player)
	if err != nil {
		player.Userid = userid
		player.Conn = ws
		player.WaitTime = time.Now().Unix()
		utils.WSAdd(&player)
		for {
			_, p, err := ws.ReadMessage()
			if err != nil {
				log.Printf("页面可能断开啦 ws.ReadJSON error: %v", err)
				break
			} else {
				fmt.Println("接受到从页面上反馈回来的信息 ", string(p))

			}
		}
	}
	c.StopRun()
}

func (c *MainController) ParseHttpReq(reqdata string, httpReq interface{}) error {
	err := json.Unmarshal([]byte(reqdata), httpReq)
	if err != nil {
		logs.Error(err)
		return err
	}
	return nil
}
