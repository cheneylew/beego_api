package routers

import (
	"github.com/cheneylew/beego_api/controllers"
	"github.com/astaxie/beego"
)

func init() {
    beego.Router("/", &controllers.MainController{})
}
