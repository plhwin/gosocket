package conf

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

var (
	RemoteAddrHeaderName string
	Logs                 logs
)

type logs struct {
	Heartbeat bool
	Room      bool
}

func Init(configFile string) {
	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("gosocket config read error: %s \n", err))
	}
	initConf()
}

func initConf() {
	RemoteAddrHeaderName = viper.GetString("remoteAddrHeaderName")
	Logs = logs{
		Heartbeat: viper.GetBool("logs.heartbeat"),
		Room:      viper.GetBool("logs.room"),
	}

	log.Printf("[gosocket][config]: RemoteAddrHeaderName: %+v Logs: %+v \n", RemoteAddrHeaderName, Logs)
}
