package main

import (
	"game/common/utils"
	_ "game/wx/routers"

	"github.com/astaxie/beego"
)

func main() {
	utils.InitWS()
	beego.Run()
}
