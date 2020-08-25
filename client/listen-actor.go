package client

import (
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/sonar/client/messages"
	"github.com/dumacp/sonar/contador"
)

//ListenActor actor to listen events
type ListenActor struct {
	*Logger
	context       actor.Context
	countingActor *actor.PID
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
}

//NewListen create listen actor
func NewListen(socket string, baudRate int, countingActor *actor.PID) *ListenActor {
	act := &ListenActor{}
	act.countingActor = countingActor
	act.socket = socket
	act.baudRate = baudRate
	act.Logger = &Logger{}
	act.quit = make(chan int, 0)
	return act
}

//Receive func Receive in actor
func (act *ListenActor) Receive(ctx actor.Context) {
	act.context = ctx
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		act.initLogs()
		act.infoLog.Printf("actor started \"%s\"", ctx.Self().Id)
		dev, err := NewSerial(act.socket, act.baudRate)
		if err != nil {
			time.Sleep(3 * time.Second)
			act.errLog.Panicln(err)
		}
		act.infoLog.Printf("connected with serial port: %s", act.socket)
		act.dev = dev
		go act.runListen(act.quit)
	case *actor.Stopping:
		act.warnLog.Println("stopped actor")
		select {
		case act.quit <- 1:
		case <-time.After(3 * time.Second):
		}
	case *messages.CountingActor:
		act.countingActor = actor.NewPID(msg.Address, msg.ID)
	case *msgListenError:
		act.errLog.Panicln("listen error")
	}
}

type msgListenError struct{}

func (act *ListenActor) runListen(quit chan int) {
	events := Listen(quit, act.dev, act.errLog)
	for v := range events {
		act.buildLog.Printf("listen event: %#v\n", v)
		switch event := v.(type) {
		case *Input:
			if event.id == 0 {
				enters := event.value
				if diff := enters - act.enters0Before; diff > 0 {
					act.context.Send(act.countingActor, &messages.Event{Id: 0, Type: messages.INPUT, Value: enters})
				}
				act.enters0Before = enters
			} else {
				enters := event.value
				if diff := enters - act.enters1Before; diff > 0 {
					act.context.Send(act.countingActor, &messages.Event{Id: 1, Type: messages.INPUT, Value: enters})
				}
				act.enters1Before = enters
			}
		case *Output:
			if event.id == 0 {
				enters := event.value
				if diff := enters - act.exits0Before; diff > 0 {
					act.context.Send(act.countingActor, &messages.Event{Id: 0, Type: messages.OUTPUT, Value: enters})
				}
				act.exits0Before = enters
			} else {
				enters := event.value
				if diff := enters - act.exits1Before; diff > 0 {
					act.context.Send(act.countingActor, &messages.Event{Id: 1, Type: messages.OUTPUT, Value: enters})
				}
				act.exits1Before = enters
			}
		case *Lock:
			if event.id == 0 {
				enters := event.value
				if diff := enters - act.locks0Before; diff > 0 {
					act.context.Send(act.countingActor, &messages.Event{Id: 0, Type: messages.TAMPERING, Value: enters})
				}
				act.locks0Before = enters
			} else {
				enters := event.value
				if diff := enters - act.locks1Before; diff > 0 {
					act.context.Send(act.countingActor, &messages.Event{Id: 1, Type: messages.TAMPERING, Value: enters})
				}
				act.locks1Before = enters
			}
		}
	}
	act.context.Send(act.context.Self(), &msgListenError{})
}
