package business

import (
	"bytes"
	"container/list"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/sonar/client/logs"
	"github.com/dumacp/sonar/client/messages"
	"github.com/dumacp/sonar/contador"
)

type MsgToTest struct {
	Data []byte
}

type MsgLogRequest struct{}
type MsgLogResponse struct {
	Value []byte
}

//ListenActor actor to listen events
type ListenActor struct {
	// *logs.Logger
	context actor.Context
	test    bool
	// countingActor *actor.PID
	enters0Before int64
	exits0Before  int64
	locks0Before  int64
	enters1Before int64
	exits1Before  int64
	locks1Before  int64

	quit chan int

	socket   string
	baudRate int
	dev      *contador.Device
	// timeFailure int
	// counterType int
	sendConsole bool
	countersMem []int64
	queue       *list.List
}

//NewListen create listen actor
func NewListen(socket string, baudRate int) *ListenActor {
	act := &ListenActor{}
	// act.countingActor = countingActor
	act.socket = socket
	act.baudRate = baudRate
	// act.logs.Logger = &logs.Logger{}
	// act.quit = make(chan int, 0)
	// act.timeFailure = 3
	act.countersMem = make([]int64, 0)
	act.queue = list.New()
	return act
}

func (act *ListenActor) SendToConsole(send bool) {
	act.sendConsole = send
}

func (act *ListenActor) Test(test bool) {
	act.test = test
}

//Receive func Receive in actor
func (act *ListenActor) Receive(ctx actor.Context) {
	act.context = ctx
	switch msg := ctx.Message().(type) {
	case *actor.Stopped:
		logs.LogInfo.Printf("actor stopped, reason: %s", msg)
	case *actor.Started:
		// act.initlogs.Logs()
		logs.LogInfo.Printf("actor started \"%s\"", ctx.Self().Id)
		if !act.test {
			dev, err := contador.NewSerial(act.socket, act.baudRate, 30*time.Second)
			if err != nil {
				time.Sleep(3 * time.Second)
				logs.LogError.Panicln(err)
			}
			logs.LogInfo.Printf("connected with serial port: %s", act.socket)
			act.dev = dev
			select {
			case <-act.quit:
			default:
				if act.quit != nil {
					close(act.quit)
				}
			}
			act.quit = make(chan int)
			go act.runNewListen(act.quit)
		}
	case *actor.Stopping:
		logs.LogWarn.Println("stopped actor")
		select {
		case _, ok := <-act.quit:
			if ok {
				close(act.quit)
			}
		case <-time.After(1 * time.Second):
			close(act.quit)
		}
	// case *messages.CountingActor:
	// 	act.countingActor = actor.NewPID(msg.Address, msg.ID)
	case *msgListenError:
		ctx.Send(ctx.Parent(), &msgPingError{})
		time.Sleep(3 * time.Second)
		logs.LogError.Panicln("listen error")
	case *MsgToTest:
		logs.LogBuild.Printf("test frame: %s", msg.Data)

		act.processData(msg.Data)
		if act.sendConsole {
			logs.LogBuild.Printf("send to console: %q", msg.Data)
			contador.SendData(act.dev, msg.Data)
		}
	case *MsgLogRequest:
		if act.queue.Len() > 0 {
			e := act.queue.Front()
			ev := e
			for ev != nil {
				if v, ok := ev.Value.([]byte); ok {
					ctx.Send(ctx.Sender(), &MsgLogResponse{Value: v})
				}
				ev = e.Next()
			}
		}
	case *msgGPS:
		// if !act.sendGPS {
		// 	break
		// }
		data := make([]byte, 0)
		data = append(data, []byte(">S;0")...)
		data = append(data, msg.data[0:len(msg.data)-3]...)
		data = append(data, []byte(";*")...)
		csum := byte(0)
		for _, v := range data {
			csum ^= v
		}
		data = append(data, []byte(fmt.Sprintf("%02X<", csum))...)
		data = append(data, []byte("\r\n")...)
		logs.LogBuild.Printf("send to console ->, %q", data)
		contador.SendData(act.dev, data)
	}
}

type msgListenError struct{}

func (act *ListenActor) processData(v []byte) {

	switch {
	case bytes.Contains(v, []byte("RPT")):
		act.queue.PushBack(v)
		if act.queue.Len() > 5 {
			e := act.queue.Front()
			act.queue.Remove(e)
		}

		split := strings.Split(string(v), ";")
		// logs.LogBuild.Printf("split, len = %d -> %v", len(split), split)
		if len(split) < 17 {
			return
		}
		data := make([]int64, 15)
		ok := true
		for i := range data {
			xint, err := strconv.ParseInt((split[i+2]), 10, 64)
			if err != nil {
				ok = false
				break
			}
			data[i] = xint
		}
		if !ok {
			return
		}

		logs.LogBuild.Printf("data: %v", data)
		logs.LogBuild.Printf("countersMem: %v", act.countersMem)

		if len(act.countersMem) <= 0 || data[0] > act.countersMem[0] {
			enters := data[0]
			if diff := enters - act.enters0Before; diff > 0 || len(act.countersMem) <= 0 {
				act.context.Send(act.context.Parent(), &messages.Event{Id: 0, Type: messages.INPUT, Value: enters})
			}
			act.enters0Before = enters
		}
		if len(act.countersMem) <= 0 || data[1] > act.countersMem[1] {
			enters := data[1]
			if diff := enters - act.exits0Before; diff > 0 || len(act.countersMem) <= 0 {
				act.context.Send(act.context.Parent(), &messages.Event{Id: 0, Type: messages.OUTPUT, Value: enters})
			}
			act.exits0Before = enters
		}
		if len(act.countersMem) <= 0 || data[2] > act.countersMem[2] {
			enters := data[2]
			if diff := enters - act.enters1Before; diff > 0 || len(act.countersMem) <= 0 {
				act.context.Send(act.context.Parent(), &messages.Event{Id: 1, Type: messages.INPUT, Value: enters})
			}
			act.enters1Before = enters
		}
		if len(act.countersMem) <= 0 || data[3] > act.countersMem[3] {
			enters := data[3]
			if diff := enters - act.exits1Before; diff > 0 || len(act.countersMem) <= 0 {
				act.context.Send(act.context.Parent(), &messages.Event{Id: 1, Type: messages.OUTPUT, Value: enters})
			}
			act.exits1Before = enters
		}

		switch {
		case len(split) == 17:

			if len(act.countersMem) >= 10 {
				id := int32(0)
				if data[10] > act.countersMem[10] {
					enters := data[10]
					if diff := enters - act.locks0Before; diff > 0 {
						act.context.Send(act.context.Parent(), &messages.Event{
							Id: id, Type: messages.TAMPERING, Value: enters})
					}
					act.locks0Before = enters
				}
			}
			if len(act.countersMem) >= 13 {
				id := int32(1)
				if data[13] > act.countersMem[13] {
					enters := data[13]
					if diff := enters - act.locks1Before; diff > 0 {
						act.context.Send(act.context.Parent(), &messages.Event{
							Id: id, Type: messages.TAMPERING, Value: enters})
					}
					act.locks1Before = enters
				}
			}
		case len(split) == 18:
			if len(act.countersMem) >= 14 && data[14] > act.countersMem[14] {
				id := int32(0)
				if data[13] > act.countersMem[13] {
					id = 1
					enters := data[13]
					if diff := enters - act.locks0Before; diff > 0 {
						act.context.Send(act.context.Parent(), &messages.Event{
							Id: id, Type: messages.TAMPERING, Value: enters})
					}
					act.locks0Before = enters
				} else {
					enters := data[14]
					if diff := enters - act.locks1Before; diff > 0 {
						act.context.Send(act.context.Parent(), &messages.Event{
							Id: id, Type: messages.TAMPERING, Value: enters})
					}
					act.locks1Before = enters
				}
			}
		}
		act.countersMem = data
	}

}

func (act *ListenActor) runNewListen(quit chan int) {

	chlisten := act.dev.ListenRawChannel(quit)
	first := true

	for v := range chlisten {

		logs.LogBuild.Printf("trama: %s\n", v)
		act.processData(v)
		//if act.sendConsole && bytes.Contains(v, []byte("RPTC")) {
		if bytes.Contains(v, []byte("RPTC")) {
			if first {
				first = false
				logs.LogInfo.Printf("actual door register: %s\n", v)
			}
			logs.LogBuild.Printf("send to console: %q", v)

			if act.sendConsole {
				contador.SendData(act.dev, []byte(v))
			}
		}
	}
	act.context.Send(act.context.Self(), &msgListenError{})
}

func (act *ListenActor) runListen(quit chan int) {
	events := contador.Listen(quit, act.dev)

	if err := func() (err error) {
		defer func() {
			r := recover()
			if r != nil {
				err = fmt.Errorf(fmt.Sprintf("recover Listen -> %v", r))
			}
		}()
		for v := range events {
			logs.LogBuild.Printf("listen event: %#v\n", v)
			switch event := v.(type) {
			case *contador.Input:
				id := event.Id
				if id == 0 {
					enters := event.Value
					if diff := enters - act.enters0Before; diff > 0 {
						act.context.Send(act.context.Parent(), &messages.Event{Id: 0, Type: messages.INPUT, Value: enters})
					}
					act.enters0Before = enters
				} else {
					enters := event.Value
					if diff := enters - act.enters1Before; diff > 0 {
						act.context.Send(act.context.Parent(), &messages.Event{Id: 1, Type: messages.INPUT, Value: enters})
					}
					act.enters1Before = enters
				}
			case *contador.Output:
				id := event.Id
				if id == 0 {
					enters := event.Value
					if diff := enters - act.exits0Before; diff > 0 {
						act.context.Send(act.context.Parent(), &messages.Event{Id: 0, Type: messages.OUTPUT, Value: enters})
					}
					act.exits0Before = enters
				} else {
					enters := event.Value
					if diff := enters - act.exits1Before; diff > 0 {
						act.context.Send(act.context.Parent(), &messages.Event{
							Id: 1, Type: messages.OUTPUT, Value: enters})
					}
					act.exits1Before = enters
				}
			case *contador.Lock:
				id := event.Id
				if id == 0 {
					enters := event.Value
					if diff := enters - act.locks0Before; diff > 0 {
						act.context.Send(act.context.Parent(), &messages.Event{
							Id: 0, Type: messages.TAMPERING, Value: enters})
					}
					act.locks0Before = enters
				} else {
					enters := event.Value
					if diff := enters - act.locks1Before; diff > 0 {
						act.context.Send(act.context.Parent(), &messages.Event{
							Id: 1, Type: messages.TAMPERING, Value: enters})
					}
					act.locks1Before = enters
				}
			}

		}
		return nil
	}(); err != nil {
		logs.LogError.Println(err)
	}

	act.context.Send(act.context.Self(), &msgListenError{})
}
