package property

import "github.com/astaxie/beego"

func GetProperty(path string) string {
	return beego.AppConfig.String(path)
}
