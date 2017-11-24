package main

import (
	// "io/ioutil"
	"fmt"
	_ "github.com/cheneylew/beego_api/routers"
	"github.com/astaxie/beego"
	"github.com/cheneylew/goutil/utils"
	"github.com/beego/bee/logger/colors"
)

func main() {
	if false {
		beego.Run()

		fmt.Println(utils.SelfPath())
		fmt.Println(colors.BlueBold("this is you name!"))
	}


	str := utils.FileReadAllString("/Users/apple/Desktop/a.txt")
	utils.JJKPrintln(str)

}

