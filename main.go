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
	}


	fmt.Println(utils.SelfPath())
	fmt.Println(colors.BlueBold("this is you name!"))
	// fmt.Println(MY_TPL)

	// ioutil.WriteFile("/Users/apple/Desktop/aa.m",[]byte(MY_TPL),0644)
}

