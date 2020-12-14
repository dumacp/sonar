package business

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/pubsub"
	"github.com/dumacp/sonar/client/logs"
	"github.com/dumacp/sonar/client/messages"
)

const (
	gprmctype = 0
	gngnsType = 1
)

//EventActor type
type EventActor struct {
	// *logs.Logger
	mem1    *memoryGPS
	mem2    *memoryGPS
	puertas map[uint]uint
	// sendEvents bool
}

//NewEventActor create EventActor
func NewEventActor() *EventActor {
	event := &EventActor{}
	// event.logs.Logger = &logs.Logger{}
	event.mem1 = &memoryGPS{}
	event.mem2 = &memoryGPS{}
	event.puertas = make(map[uint]uint)
	// event.sendEvents = true
	return event
}

type msgEventCounter struct {
	data []byte
}

type msgEvent struct {
	data []byte
}

//Receive function to Receive actor messages
func (act *EventActor) Receive(ctx actor.Context) {

	switch msg := ctx.Message().(type) {
	case *actor.Started:
		// act.initlogs.Logs()
		logs.LogInfo.Printf("actor started \"%s\"", ctx.Self().Id)
	case *messages.Event:
		logs.LogBuild.Printf("\"%s\" - event -> '%v'\n", ctx.Self().GetId(), msg)
		// if !act.sendEvents {
		// 	logs.LogBuild.Println("send events disable")
		// 	break
		// }
		var event []byte
		switch msg.Type {
		case messages.INPUT:
			event = buildEventPass(ctx, msg, act.mem1, act.mem2, act.puertas)
			ctx.Send(ctx.Parent(), &msgEventCounter{data: event})
		case messages.OUTPUT:
			event = buildEventPass(ctx, msg, act.mem1, act.mem2, act.puertas)
			ctx.Send(ctx.Parent(), &msgEventCounter{data: event})
		case messages.TAMPERING:
			event = buildEventTampering(ctx, msg, act.mem1, act.mem2, act.puertas)
			ctx.Send(ctx.Parent(), &msgEvent{data: event})
		}

	case *msgGPS:
		mem := captureGPS(msg.data)
		switch mem.typeM {
		case gprmctype:
			act.mem1 = &mem
		case gngnsType:
			act.mem2 = &mem
		}
	case *msgDoor:
		act.puertas[msg.id] = msg.value
	case *actor.Stopped:
		logs.LogWarn.Println("stoped actor")
	}
}

type memoryGPS struct {
	typeM     int
	frame     string
	timestamp int64
}

func captureGPS(gps []byte) memoryGPS {
	// memoryGPRMC := new(memoryGPS)
	// memoryGNSNS := new(memoryGPS)
	// ch1 := make(chan memoryGPS, 2)
	// ch2 := make(chan memoryGPS, 1)
	// go func() {
	// for v := range chGPS {
	memory := memoryGPS{}
	if bytes.Contains(gps, []byte("GPRMC")) {
		memory.typeM = gprmctype
		memory.frame = string(gps)
		memory.timestamp = time.Now().Unix()

	} else if bytes.Contains(gps, []byte("GNGNS")) {
		memory.typeM = gngnsType
		memory.frame = string(gps)
		memory.timestamp = time.Now().Unix()
	}
	return memory
}

func buildEventPass(ctx actor.Context, v *messages.Event, mem1, mem2 *memoryGPS, puerta map[uint]uint) []byte {
	tn := time.Now()

	contadores := []int64{0, 0}
	if v.Type == messages.INPUT {
		contadores[0] = v.Value
	} else if v.Type == messages.OUTPUT {
		contadores[1] = v.Value
	}
	frame := ""

	if mem1.timestamp > mem2.timestamp {
		if mem1.timestamp+30 > tn.Unix() {
			frame = mem1.frame
		}
	} else {
		if mem2.timestamp+30 > tn.Unix() {
			frame = mem2.frame
		}
	}

	doorState := uint(0)
	switch v.Id {
	case 0:
		if vm, ok := puerta[gpioPuerta1]; ok {
			doorState = vm
		}
	case 1:
		if vm, ok := puerta[gpioPuerta2]; ok {
			doorState = vm
		}
	}

	message := &pubsub.Message{
		Timestamp: float64(time.Now().UnixNano()) / 1000000000,
		Type:      "COUNTERSDOOR",
	}

	val := struct {
		Coord    string  `json:"coord"`
		ID       int32   `json:"id"`
		State    uint    `json:"state"`
		Counters []int64 `json:"counters"`
	}{
		frame,
		v.Id,
		doorState,
		contadores[0:2],
	}
	message.Value = val

	msg, err := json.Marshal(message)
	if err != nil {
		logs.LogError.Println(err)
	}
	logs.LogBuild.Printf("%s\n", msg)

	return msg
}

func buildEventTampering(ctx actor.Context, v *messages.Event, mem1, mem2 *memoryGPS, puerta map[uint]uint) []byte {
	tn := time.Now()

	if v.Type != messages.TAMPERING {
		return nil
	}
	frame := ""

	// if mem1.timestamp > mem2.timestamp {
	if mem1.timestamp+30 > tn.Unix() {
		frame = mem1.frame
		// }
	} else if mem2.timestamp+30 > tn.Unix() {
		frame = mem2.frame
		// }
	}

	doorState := uint(0)
	switch v.Id {
	case 0:
		if vm, ok := puerta[gpioPuerta1]; ok {
			doorState = vm
		}
	case 1:
		if vm, ok := puerta[gpioPuerta2]; ok {
			doorState = vm
		}
	}

	message := &pubsub.Message{
		Timestamp: float64(time.Now().UnixNano()) / 1000000000,
		Type:      "TAMPERING",
	}

	val := struct {
		Coord    string  `json:"coord"`
		ID       int32   `json:"id"`
		State    uint    `json:"state"`
		Counters []int64 `json:"counters"`
	}{
		frame,
		v.Id,
		doorState,
		[]int64{0, 0},
	}
	message.Value = val

	msg, err := json.Marshal(message)
	if err != nil {
		logs.LogError.Println(err)
	}
	logs.LogBuild.Printf("%s\n", msg)

	return msg
}
