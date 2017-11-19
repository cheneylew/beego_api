package routers

import (
	"cheneylew.com/cheneylew.com/beego_api/controllers"
	"github.com/astaxie/beego"
)

func init() {
    beego.Router("/", &controllers.MainController{})
}
