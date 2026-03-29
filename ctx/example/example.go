package main

import (
	"fmt"

	"github.com/hsuanshao/golibs/ctx"
	"github.com/sirupsen/logrus"
)

func init() {
}

type User struct {
	Name string
	Age  uint
}

func main() {
	c1 := ctx.Background()
	fmt.Println(int(logrus.DebugLevel))
	// print differ level to see do we send currect level message to rollbar
	c1.Debug("Now, I send debug should send log to rollbar, this is deug level log")
	c1.WithField("logtest", "info").Info("This is info level log")
	c1.Warn("this is warn level log")
	c1.Error("this is error level log")
	c1.Fatal("this is fatal error level log")

}
