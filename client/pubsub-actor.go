package client

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/pubsub"
	"github.com/dumacp/sonar/client/messages"
	MQTT "github.com/eclipse/paho.mqtt.golang"
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

//ActorPubsub actor to send mesages to MQTTT broket
type ActorPubsub struct {
	ctx actor.Context
	*Logger
	clientMqtt MQTT.Client
	debug      bool
	chGPS      chan []byte
}

//NewPubSubActor create PubSubActor
func NewPubSubActor() *ActorPubsub {
	act := &ActorPubsub{}
	act.Logger = &Logger{}
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
		act.initLogs()

		clientMqtt, err := connectMqtt(act.ctx)
		if err != nil {
			time.Sleep(3 * time.Second)
			panic(err)
		}
		act.clientMqtt = clientMqtt
		act.infoLog.Printf("actor started \"%s\"", ctx.Self().Id)
	case *actor.Stopping:
		act.clientMqtt.Disconnect(100)
	case *messages.Snapshot:
		reg := &register{}
		regv := make([]int64, 0)
		for _, v := range msg.RawInputs {
			regv = append(regv, v)
		}
		for _, v := range msg.RawOutputs {
			regv = append(regv, v)
		}
		reg.Registers = regv
		data, err := json.Marshal(reg)
		if err != nil {
			act.errLog.Println(err)
			break
		}
		act.buildLog.Printf("data: %q", data)
		token := act.clientMqtt.Publish(topicCounter, 0, false, data)
		if ok := token.WaitTimeout(3 * time.Second); !ok {
			act.clientMqtt.Disconnect(100)
			act.errLog.Panic("MQTT connection failed")
		}
	case *msgEvent:
		// fmt.Printf("event: %s\n", msg.event)
		act.buildLog.Printf("data: %q", msg)
		token := act.clientMqtt.Publish(topicEvents, 0, false, msg.data)
		if ok := token.WaitTimeout(3 * time.Second); !ok {
			act.clientMqtt.Disconnect(100)
			act.errLog.Panic("MQTT connection failed")
		}
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
		act.buildLog.Printf("data: %q", data)
		token := act.clientMqtt.Publish(topicEvents, 0, false, data)
		if ok := token.WaitTimeout(3 * time.Second); !ok {
			act.clientMqtt.Disconnect(100)
			act.errLog.Panic("MQTT connection failed")
		}
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
