package main

import (
	"game/common/utils"
	_ "game/websocket_wxserver/routers"

	"github.com/astaxie/beego"
)

func main() {
	utils.InitWSClient()
	beego.Run()
}
