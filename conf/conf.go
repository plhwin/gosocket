package conf

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

const (
	TransportProtocolText   = "text"
	TransportProtocolBinary = "binary"
)

var (
	Acceptor  acceptor
	Initiator initiator
)

type acceptor struct {
	TransportProtocol transportProtocol
	Websocket         websocket
	Heartbeat         heartbeat
	Logs              logs
}

type transportProtocol struct {
	Send    string
	Receive string
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
	TransportProtocol transportProtocol
	Logs              logs
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

	// set default value for acceptor heartbeat
	if Acceptor.Heartbeat.PingInterval <= 0 {
		Acceptor.Heartbeat.PingInterval = 5
	}
	if Acceptor.Heartbeat.PingMaxTimes <= 0 {
		Acceptor.Heartbeat.PingMaxTimes = 2
	}

	// set default value for transport protocol
	switch viper.GetString("acceptor.transportProtocol.send") {
	case TransportProtocolBinary:
		Acceptor.TransportProtocol.Send = TransportProtocolBinary
	default:
		Acceptor.TransportProtocol.Send = TransportProtocolText
	}
	switch viper.GetString("acceptor.transportProtocol.receive") {
	case TransportProtocolBinary:
		Acceptor.TransportProtocol.Receive = TransportProtocolBinary
	default:
		Acceptor.TransportProtocol.Receive = TransportProtocolText
	}
	switch viper.GetString("initiator.transportProtocol.send") {
	case TransportProtocolBinary:
		Initiator.TransportProtocol.Send = TransportProtocolBinary
	default:
		Initiator.TransportProtocol.Send = TransportProtocolText
	}
	switch viper.GetString("initiator.transportProtocol.receive") {
	case TransportProtocolBinary:
		Initiator.TransportProtocol.Receive = TransportProtocolBinary
	default:
		Initiator.TransportProtocol.Receive = TransportProtocolText
	}

	log.Printf("[gosocket][config]:\nAcceptor: %+v \nInitiator: %+v \n\n", Acceptor, Initiator)
}
