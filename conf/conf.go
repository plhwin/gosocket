package conf

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

var (
	Acceptor = acceptor{
		Heartbeat: heartbeat{
			PingInterval: 5,
			PingMaxTimes: 2,
		},
	}
	Initiator initiator
)

type acceptor struct {
	Websocket websocket
	Heartbeat heartbeat
	Logs      logs
}

type websocket struct {
	RemoteAddrHeaderName string
}

type heartbeat struct {
	PingInterval int
	PingMaxTimes int
}

type logs struct {
	Heartbeat heartbeatLogs
	Room      room
	LeaveAll  bool
}

type heartbeatLogs struct {
	PingSend           bool
	PingSendPrintDelay int64
	PingReceive        bool
	PongReceive        bool
}

type room struct {
	Join  bool
	Leave bool
}

type initiator struct {
	Logs logs
}

func Init(configFile string) {
	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("gosocket config read error: %s \n", err))
	}
	initConf()
}

func initConf() {
	Acceptor = acceptor{
		Websocket: websocket{
			RemoteAddrHeaderName: viper.GetString("acceptor.websocket.remoteAddrHeaderName"),
		},
		Heartbeat: heartbeat{
			PingInterval: viper.GetInt("acceptor.heartbeat.pingInterval"),
			PingMaxTimes: viper.GetInt("acceptor.heartbeat.pingMaxTimes"),
		},
		Logs: logs{
			Heartbeat: heartbeatLogs{
				PingSend:           viper.GetBool("acceptor.logs.heartbeat.pingSend"),
				PingSendPrintDelay: viper.GetInt64("acceptor.logs.heartbeat.pingSendPrintDelay"),
				PingReceive:        viper.GetBool("acceptor.logs.heartbeat.pingReceive"),
				PongReceive:        viper.GetBool("acceptor.logs.heartbeat.pongReceive"),
			},
			Room: room{
				Join:  viper.GetBool("acceptor.logs.room.join"),
				Leave: viper.GetBool("acceptor.logs.room.leave"),
			},
			LeaveAll: viper.GetBool("acceptor.logs.leaveAll"),
		},
	}
	Initiator = initiator{
		Logs: logs{
			Heartbeat: heartbeatLogs{
				PingReceive: viper.GetBool("initiator.logs.heartbeat.pingReceive"),
				PongReceive: viper.GetBool("initiator.logs.heartbeat.pongReceive"),
			},
		},
	}
	log.Printf("[gosocket][config]:\nAcceptor: %+v \nInitiator: %+v \n\n", Acceptor, Initiator)
}
