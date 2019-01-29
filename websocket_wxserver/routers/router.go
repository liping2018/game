package routers

import (
	"game/websocket_wxserver/controllers"

	"github.com/astaxie/beego"
)

func init() {
	beego.Router("/", &controllers.MainController{})
	beego.Router("/send", &controllers.MainController{}, "*:Message")
	beego.Router("/start", &controllers.MainController{}, "*:Start")
	beego.Router("/exit", &controllers.MainController{}, "*:PlayerExit")
}
