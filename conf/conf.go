package conf

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

const (
	// Websocket Message Type
	WebsocketMessageTypeText   = "Text"
	WebsocketMessageTypeBinary = "Binary"

	// Serialize
	TransportSerializeText     = "Text"
	TransportSerializeProtobuf = "Protobuf"

	// Compress
	TransportCompressNone   = "None"
	TransportCompressSnappy = "Snappy"
	TransportCompressFLate  = "FLate"
	TransportCompressGzip   = "Gzip"
)

var (
	Acceptor  acceptor
	Initiator initiator
)

type acceptor struct {
	Transport transport
	Websocket websocket
	Heartbeat heartbeat
	Logs      logs
}

type transport struct {
	Send    transportConfig
	Receive transportConfig
}

type transportConfig struct {
	Serialize string
	Compress  string
}

type websocket struct {
	MessageType          string
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
	Transport transport
	Logs      logs
}

func Init(configFile string) {
	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("gosocket config read error: %s \n", err))
	}
	initConf()
}

func initConf() {
	websocketMessageTypes := []string{WebsocketMessageTypeText, WebsocketMessageTypeBinary}
	serializations := []string{TransportSerializeText, TransportSerializeProtobuf}
	compresses := []string{TransportCompressNone, TransportCompressSnappy, TransportCompressFLate, TransportCompressGzip}

	Acceptor = acceptor{
		Transport: transport{
			Send: transportConfig{
				Serialize: getVal(viper.GetString("acceptor.transport.send.serialize"), serializations, TransportSerializeText),
				Compress:  getVal(viper.GetString("acceptor.transport.send.compress"), compresses, TransportCompressNone),
			},
			Receive: transportConfig{
				Serialize: getVal(viper.GetString("acceptor.transport.receive.serialize"), serializations, TransportSerializeText),
				Compress:  getVal(viper.GetString("acceptor.transport.receive.compress"), compresses, TransportCompressNone),
			},
		},
		Websocket: websocket{
			MessageType:          getVal(viper.GetString("acceptor.websocket.messageType"), websocketMessageTypes, WebsocketMessageTypeText),
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
		Transport: transport{
			Send: transportConfig{
				Serialize: getVal(viper.GetString("initiator.transport.send.serialize"), serializations, TransportSerializeText),
				Compress:  getVal(viper.GetString("initiator.transport.send.compress"), compresses, TransportCompressNone),
			},
			Receive: transportConfig{
				Serialize: getVal(viper.GetString("initiator.transport.receive.serialize"), serializations, TransportSerializeText),
				Compress:  getVal(viper.GetString("initiator.transport.receive.compress"), compresses, TransportCompressNone),
			},
		},
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

	log.Printf("[gosocket][config]:\nAcceptor: %+v \nInitiator: %+v \n\n", Acceptor, Initiator)
}

func getVal(s string, ss []string, def string) (v string) {
	exist := false
	for _, val := range ss {
		if val == s {
			exist = true
			break
		}
	}
	v = def
	if exist {
		v = s
	}
	return
}
