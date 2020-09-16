package business

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/pubsub"
	"github.com/dumacp/sonar/client/logs"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	clietnName  = "go-camera-actor"
	topicEvents = "EVENTS/backcounter"
	// topicScene   = "EVENTS/scene"
	topicCounter = "COUNTERSDOOR"
)

type msgGPS struct {
	data []byte
}

type MsgSendEvents struct {
	Data bool
}

//ActorPubsub actor to send mesages to MQTTT broket
type ActorPubsub struct {
	ctx actor.Context
	// *logs.Logger
	clientMqtt MQTT.Client
	debug      bool
	chGPS      chan []byte
	sendEvents bool
}

//NewPubSubActor create PubSubActor
func NewPubSubActor() *ActorPubsub {
	act := &ActorPubsub{}
	act.sendEvents = true
	// act.logs.Logger = &logs.Logger{}
	return act
}

type register struct {
	Registers []int64 `json:"registers"`
}

//Receive func Receive to actor
func (act *ActorPubsub) Receive(ctx actor.Context) {
	act.ctx = ctx
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		// act.initlogs.Logs()

		clientMqtt, err := connectMqtt(act.ctx)
		if err != nil {
			time.Sleep(3 * time.Second)
			panic(err)
		}
		act.clientMqtt = clientMqtt
		logs.LogInfo.Printf("actor started \"%s\"", ctx.Self().Id)
	case *actor.Stopping:
		act.clientMqtt.Disconnect(100)
	case *register:
		data, err := json.Marshal(msg)
		if err != nil {
			logs.LogError.Println(err)
			break
		}
		logs.LogBuild.Printf("data: %q", data)

		publish(act.clientMqtt, topicCounter, data, act.sendEvents)

	// case *messages.Snapshot:
	// 	reg := &register{}
	// 	regv := make([]int64, 0)
	// 	sortInputs := SortMap(msg.Inputs)
	// 	for _, v := range sortInputs {
	// 		regv = append(regv, v)
	// 	}
	// 	sortOutputs := SortMap(msg.Outputs)
	// 	for _, v := range sortOutputs {
	// 		regv = append(regv, v)
	// 	}
	// 	reg.Registers = regv
	// 	data, err := json.Marshal(reg)
	// 	if err != nil {
	// 		logs.LogError.Println(err)
	// 		break
	// 	}
	// 	logs.LogBuild.Printf("data: %q", data)
	// 	token := act.clientMqtt.Publish(topicCounter, 0, false, data)
	// 	if ok := token.WaitTimeout(3 * time.Second); !ok {
	// 		act.clientMqtt.Disconnect(100)
	// 		logs.LogError.Panic("MQTT connection failed")
	// 	}
	case *MsgSendEvents:
		act.sendEvents = msg.Data
	case *msgEvent:
		// fmt.Printf("event: %s\n", msg.event)
		logs.LogBuild.Printf("data: %q", msg)
		publish(act.clientMqtt, topicEvents, msg.data, act.sendEvents)

	case *msgPingError:
		// fmt.Printf("event: %s\n", msg.event)
		message := &pubsub.Message{
			Timestamp: float64(time.Now().UnixNano()) / 1000000000,
			Type:      "CounterDisconnect",
			Value:     1,
		}
		data, err := json.Marshal(message)
		if err != nil {
			break
		}
		logs.LogBuild.Printf("data: %q", data)
		publish(act.clientMqtt, topicEvents, data, act.sendEvents)

	}
}
func publish(c mqtt.Client, topic string, payload interface{}, send bool) {
	if !send {
		logs.LogBuild.Println("data send disable")
	}
	token := c.Publish(topicEvents, 0, false, payload)
	if ok := token.WaitTimeout(3 * time.Second); !ok {
		c.Disconnect(100)
		logs.LogError.Panic("MQTT connection failed")
	}
}

func onMessageWithChannel(ctx actor.Context) func(c MQTT.Client, msg MQTT.Message) {
	onMessage := func(c MQTT.Client, msg MQTT.Message) {
		ctx.Send(ctx.Parent(), &msgGPS{data: msg.Payload()})
	}
	return onMessage
}

func connectMqtt(ctx actor.Context) (MQTT.Client, error) {
	opts := MQTT.NewClientOptions().AddBroker("tcp://127.0.0.1:1883")
	opts.SetClientID(clietnName)
	opts.SetAutoReconnect(false)
	conn := MQTT.NewClient(opts)
	token := conn.Connect()
	if ok := token.WaitTimeout(30 * time.Second); !ok {
		return nil, fmt.Errorf("MQTT connection failed")
	}
	tokenSubs := conn.Subscribe("GPS", 0, onMessageWithChannel(ctx))
	if ok := tokenSubs.WaitTimeout(30 * time.Second); !ok {
		return nil, fmt.Errorf("MQTT connection failed")
	}
	return conn, nil
}

func (act *ActorPubsub) funcSendGps() {
	go func() {
		for v := range act.chGPS {
			act.ctx.Send(act.ctx.Parent(), &msgGPS{data: v})
		}
	}()
}
