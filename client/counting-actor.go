package client

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/persistence"
	"github.com/dumacp/sonar/client/messages"
)

//CountingActor struct
type CountingActor struct {
	persistence.Mixin
	*Logger
	flagRecovering bool
	inputs         map[int32]int64
	outputs        map[int32]int64
	rawInputs      map[int32]int64
	rawOutputs     map[int32]int64

	puertas   map[uint]uint
	openState map[int32]uint

	pubsub *actor.PID
	doors  *actor.PID
	events *actor.PID
	ping   *actor.PID
}

//NewCountingActor create CountingActor
func NewCountingActor() *CountingActor {
	count := &CountingActor{}
	count.Logger = &Logger{}
	count.puertas = make(map[uint]uint)
	count.openState = make(map[int32]uint)
	return count
}

//SetZeroOpenStateDoor0 set the open state in gpio door
func (a *CountingActor) SetZeroOpenStateDoor0(state bool) {
	if state {
		a.openState[0] = 0
	} else {
		a.openState[0] = 1
	}
}

//SetZeroOpenStateDoor1 set the open state in gpio door
func (a *CountingActor) SetZeroOpenStateDoor1(state bool) {
	if state {
		a.openState[1] = 0
	} else {
		a.openState[1] = 1
	}
}

// type Snapshot struct {
// 	Inputs  uint32
// 	Outputs uint32
// }

// func (snap *Snapshot) Reset() { *snap = Snapshot{} }
// func (snap *Snapshot) String() string {
// 	return fmt.Sprintf("{Inputs: %d, Outputs: %d}", snap.Inputs, snap.Outputs)
// }
// func (snap *Snapshot) ProtoMessage() {}

//MsgSendRegisters messages to send registers to pubsub
type MsgSendRegisters struct{}

//Receive function to receive message in actor
func (a *CountingActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		a.inputs = make(map[int32]int64)
		a.outputs = make(map[int32]int64)
		a.rawInputs = make(map[int32]int64)
		a.rawOutputs = make(map[int32]int64)
		a.initLogs()
		a.infoLog.Printf("actor started \"%s\"", ctx.Self().Id)

		pubsub := NewPubSubActor()
		pubsub.SetLogError(a.errLog).
			SetLogWarn(a.warnLog).
			SetLogInfo(a.infoLog).
			SetLogBuild(a.buildLog)
		if a.debug {
			pubsub.WithDebug()
		}
		props1 := actor.PropsFromFunc(pubsub.Receive)
		pid1, err := ctx.SpawnNamed(props1, "pubsub")
		if err != nil {
			a.errLog.Panicln(err)
		}
		a.pubsub = pid1

		events := NewEventActor()
		events.SetLogError(a.errLog).
			SetLogWarn(a.warnLog).
			SetLogInfo(a.infoLog).
			SetLogBuild(a.buildLog)
		if a.debug {
			events.WithDebug()
		}
		props2 := actor.PropsFromProducer(func() actor.Actor { return events })
		pid2, err := ctx.SpawnNamed(props2, "events")
		if err != nil {
			a.errLog.Panicln(err)
		}
		a.events = pid2

		props3 := actor.PropsFromProducer(func() actor.Actor { return &DoorsActor{} })
		pid3, err := ctx.SpawnNamed(props3, "doors")
		if err != nil {
			a.errLog.Panicln(err)
		}
		a.doors = pid3

		// props4 := actor.PropsFromProducer(func() actor.Actor { return &PingActor{} })
		// pid4, err := ctx.SpawnNamed(props4, "ping")
		// if err != nil {
		// 	a.errLog.Panicln(err)
		// }
		// a.ping = pid4

	case *persistence.RequestSnapshot:
		a.buildLog.Printf("snapshot internal state: inputs -> '%v', outputs -> '%v', rawInputs -> %v, rawOutpts -> %v\n",
			a.inputs, a.outputs, a.rawInputs, a.rawOutputs)
		snap := &messages.Snapshot{
			Inputs:     a.inputs,
			Outputs:    a.outputs,
			RawInputs:  a.rawInputs,
			RawOutputs: a.rawOutputs,
		}
		a.PersistSnapshot(snap)
		ctx.Send(a.pubsub, snap)

	case *MsgSendRegisters:

		if verifySum(a.outputs) <= 0 && verifySum(a.inputs) <= 0 {
			break
		}
		snap := &messages.Snapshot{
			Inputs:     a.inputs,
			Outputs:    a.outputs,
			RawInputs:  a.rawInputs,
			RawOutputs: a.rawOutputs,
		}
		ctx.Send(a.pubsub, snap)

	case *messages.Snapshot:
		if msg.GetInputs() != nil {
			a.inputs = msg.GetInputs()
		}
		if msg.GetOutputs() != nil {
			a.outputs = msg.GetOutputs()
		}
		if msg.GetRawInputs() != nil {
			a.rawInputs = msg.GetRawInputs()
		}
		if msg.GetRawOutputs() != nil {
			a.rawOutputs = msg.GetRawOutputs()
		}
		a.infoLog.Printf("recovered from snapshot, internal state changed to:\n\tinputs -> '%v', outputs -> '%v', rawInputs -> %v, rawOutpts -> %v\n",
			a.inputs, a.outputs, a.rawInputs, a.rawOutputs)
		// ctx.Send(a.pubsub, msg)
	case *persistence.ReplayComplete:
		a.infoLog.Printf("replay completed, internal state changed to:\n\tinputs -> '%v', outputs -> '%v'\n",
			a.inputs, a.outputs)
		snap := &messages.Snapshot{
			Inputs:     a.inputs,
			Outputs:    a.outputs,
			RawInputs:  a.rawInputs,
			RawOutputs: a.rawOutputs,
		}
		// a.PersistSnapshot(snap)
		ctx.Send(a.pubsub, snap)
	case *messages.Event:
		if a.Recovering() {
			// a.flagRecovering = true
			scenario := "received replayed event"
			a.buildLog.Printf("%s, internal state changed to\n\tinputs -> '%v', outputs -> '%v'\n",
				scenario, a.inputs, a.outputs)
		} else {
			a.PersistReceive(msg)
			scenario := "received new message"
			a.buildLog.Printf("%s, internal state changed to\n\tinputs -> '%v', outputs -> '%v'\n",
				scenario, a.inputs, a.outputs)
		}
		// a.buildLog.Printf("data ->'%v', rawinputs -> '%v', rawoutputs -> '%v' \n",
		// 	msg.GetValue(), a.rawInputs, a.rawOutputs)
		switch msg.GetType() {
		case messages.INPUT:
			id := msg.Id
			if _, ok := a.rawInputs[id]; !ok {
				a.rawInputs[id] = 0
			}
			if _, ok := a.inputs[id]; !ok {
				a.inputs[id] = 0
			}

			diff := msg.GetValue() - a.rawInputs[id]
			if diff > 0 && a.rawInputs[id] <= 0 {
				a.inputs[id] = 1
				if !a.Recovering() {
					ctx.Send(a.events, &messages.Event{Id: id, Type: messages.INPUT, Value: 1})
				}
			} else if diff > 0 {
				if v, ok := a.puertas[uint(id)]; !ok || v == a.openState[id] {
					a.inputs[id] += diff
					if !a.Recovering() {
						ctx.Send(a.events, &messages.Event{Id: id, Type: messages.INPUT, Value: diff})
					}
				}
			} else if diff < 0 {
				// a.inputs[id] += msg.GetValue()
				if !a.Recovering() {
					ctx.Send(a.events, msg)
				}
			}
			a.rawInputs[id] = msg.GetValue()
		case messages.OUTPUT:
			id := msg.Id
			if _, ok := a.rawOutputs[id]; !ok {
				a.rawOutputs[id] = 0
			}
			if _, ok := a.outputs[id]; !ok {
				a.outputs[id] = 0
			}
			diff := msg.GetValue() - a.rawOutputs[id]
			if diff > 0 && a.rawOutputs[id] <= 0 {
				a.outputs[id] = 1
				if !a.Recovering() {
					ctx.Send(a.events, &messages.Event{Id: id, Type: messages.OUTPUT, Value: 1})
				}
			} else if diff > 0 {
				if v, ok := a.puertas[uint(id)]; !ok || v == a.openState[id] {
					a.outputs[id] += diff
					if !a.Recovering() {
						ctx.Send(a.events, &messages.Event{Id: id, Type: messages.OUTPUT, Value: diff})
					}
				}
			} else if diff < 0 {
				// a.outputs[id] += msg.GetValue()
				if !a.Recovering() {
					ctx.Send(a.events, msg)
				}
			}
			a.rawOutputs[id] = msg.GetValue()
		case messages.TAMPERING:
			a.warnLog.Println("shelteralarm")
			ctx.Send(a.events, msg)
		}

		// if a.flagRecovering {
		// 	a.flagRecovering = false
		// 	snap := &messages.Snapshot{
		// 		Inputs:     a.inputs,
		// 		Outputs:    a.outputs,
		// 		RawInputs:  a.rawInputs,
		// 		RawOutputs: a.rawOutputs,
		// 	}
		// 	a.PersistSnapshot(snap)
		// }

	case *msgPingError:
		a.warnLog.Printf("counter keep alive error")
		ctx.Send(a.pubsub, msg)
	case *msgDoor:
		a.puertas[msg.id] = msg.value
		ctx.Send(a.events, msg)
	case *msgGPS:
		ctx.Send(a.events, msg)
	case *msgEvent:
		// a.buildLog.Printf("\"%s\" - msg: '%q'\n", ctx.Self().GetId(), msg)
		ctx.Send(a.pubsub, msg)
	case *actor.Terminated:
		a.warnLog.Printf("actor terminated: %s", msg.GetWho().GetAddress())
	}
}

func verifySum(data map[int32]int64) int64 {
	sum := int64(0)
	for _, v := range data {
		sum += v
	}
	return sum
}
