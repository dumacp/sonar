package business

import (
	"bytes"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/persistence"
	"github.com/dumacp/sonar/client/logs"
	"github.com/dumacp/sonar/client/messages"
)

//CountingActor struct
type CountingActor struct {
	persistence.Mixin
	// *logs.Logger
	flagRecovering bool
	inputs         map[int32]int64
	outputs        map[int32]int64
	rawInputs      map[int32]int64
	rawOutputs     map[int32]int64
	dayInputs      map[int64]int64
	dayOutputs     map[int32]int64

	puertas            map[uint]uint
	openState          map[int32]uint
	disableDoorGpio    bool
	gpsToConsole       bool
	disablePersistence bool

	listenner *ListenActor

	pubsub *actor.PID
	doors  *actor.PID
	events *actor.PID
	ping   *actor.PID
	listen *actor.PID

	counterType int
}

//NewCountingActor create CountingActor
func NewCountingActor(listenner *ListenActor) *CountingActor {
	count := &CountingActor{}
	count.listenner = listenner
	// count.logs.Logger = &logs.Logger{}
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

//DisableDoorGpioListen Disable DoorGpio
func (a *CountingActor) DisableDoorGpioListen(state bool) {
	if state {
		a.disableDoorGpio = true
	} else {
		a.disableDoorGpio = false
	}
}

//CounterType set counter type
func (a *CountingActor) CounterType(tp int) {
	a.counterType = tp
}

//SetGPStoConsole set gps to consolse
func (a *CountingActor) SetGPStoConsole(gpsConsole bool) {
	a.gpsToConsole = gpsConsole
}

// //DisablePersistence disable persistence
// func (a *CountingActor) DisablePersistence(disable bool) {
// 	a.disablePersistence = disable
// }

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
		if len(a.Mixin.Name()) <= 0 {
			logs.LogInfo.Println("disable persistence")
			a.disablePersistence = true
		}
		a.inputs = make(map[int32]int64)
		a.outputs = make(map[int32]int64)
		a.rawInputs = make(map[int32]int64)
		a.rawOutputs = make(map[int32]int64)
		// a.initlogs.logs.Logs()
		logs.LogInfo.Printf("actor started \"%s\"", ctx.Self().Id)

		pubsub := NewPubSubActor()
		// pubsub.Setlogs.LogError(a.errlogs.Log).
		// 	Setlogs.LogWarn(a.warnlogs.Log).
		// 	Setlogs.LogInfo(a.infologs.Log).
		// 	Setlogs.LogBuild(a.buildlogs.Log)
		// if a.debug {
		// 	pubsub.WithDebug()
		// }
		props1 := actor.PropsFromFunc(pubsub.Receive)
		pid1, err := ctx.SpawnNamed(props1, "pubsub")
		if err != nil {
			logs.LogError.Panicln(err)
		}
		a.pubsub = pid1

		events := NewEventActor()
		// events.Setlogs.LogError(a.errlogs.Log).
		// 	Setlogs.LogWarn(a.warnlogs.Log).
		// 	Setlogs.LogInfo(a.infologs.Log).
		// 	Setlogs.LogBuild(a.buildlogs.Log)
		// if a.debug {
		// 	events.WithDebug()
		// }
		props2 := actor.PropsFromProducer(func() actor.Actor { return events })
		pid2, err := ctx.SpawnNamed(props2, "events")
		if err != nil {
			logs.LogError.Panicln(err)
		}
		a.events = pid2

		props3 := actor.PropsFromProducer(func() actor.Actor { return &DoorsActor{} })
		pid3, err := ctx.SpawnNamed(props3, "doors")
		if err != nil {
			logs.LogError.Panicln(err)
		}
		a.doors = pid3

		// props4 := actor.PropsFromProducer(func() actor.Actor { return &PingActor{} })
		// pid4, err := ctx.SpawnNamed(props4, "ping")
		// if err != nil {
		// 	a.errlogs.Log.Panicln(err)
		// }
		// a.ping = pid4

		// a.listenner.Setlogs.LogError(a.errlogs.Log).Setlogs.LogWarn(a.warnlogs.Log).Setlogs.LogInfo(a.infologs.Log).Setlogs.LogBuild(a.buildlogs.Log)
		// if a.debug {
		// 	a.listenner.WithDebug()
		// }

		propsListen := actor.PropsFromFunc(a.listenner.Receive)
		a.listen, err = ctx.SpawnNamed(propsListen, "listenner")
		if err != nil {
			logs.LogError.Panicln(err)
		}

	case *persistence.RequestSnapshot:
		logs.LogInfo.Printf("snapshot internal state: inputs -> '%v', outputs -> '%v', rawInputs -> %v, rawOutpts -> %v\n",
			a.inputs, a.outputs, a.rawInputs, a.rawOutputs)
		snap := &messages.Snapshot{
			Inputs:     a.inputs,
			Outputs:    a.outputs,
			RawInputs:  a.rawInputs,
			RawOutputs: a.rawOutputs,
		}
		a.PersistSnapshot(snap)
		// ctx.Send(a.pubsub, snap)
		if reg := registers(a.inputs, a.outputs, a.counterType); reg != nil {
			ctx.Send(a.pubsub, reg)
		}

	case *MsgSendRegisters:

		if verifySum(a.outputs) <= 0 && verifySum(a.inputs) <= 0 {
			break
		}
		// snap := &messages.Snapshot{
		// 	Inputs:     a.inputs,
		// 	Outputs:    a.outputs,
		// 	RawInputs:  a.rawInputs,
		// 	RawOutputs: a.rawOutputs,
		// }
		// ctx.Send(a.pubsub, snap)
		if reg := registers(a.inputs, a.outputs, a.counterType); reg != nil {
			// log.Printf("registers: %v", reg)
			ctx.Send(a.pubsub, reg)
		}

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
		logs.LogInfo.Printf("recovered from snapshot, internal state changed to:\n\tinputs -> '%v', outputs -> '%v', rawInputs -> %v, rawOutpts -> %v\n",
			a.inputs, a.outputs, a.rawInputs, a.rawOutputs)
		// ctx.Send(a.pubsub, msg)
	case *persistence.ReplayComplete:
		logs.LogInfo.Printf("replay completed, internal state changed to:\n\tinputs -> '%v', outputs -> '%v'\n",
			a.inputs, a.outputs)
		// snap := &messages.Snapshot{
		// 	Inputs:     a.inputs,
		// 	Outputs:    a.outputs,
		// 	RawInputs:  a.rawInputs,
		// 	RawOutputs: a.rawOutputs,
		// }
		// a.PersistSnapshot(snap)
		// ctx.Send(a.pubsub, snap)
		if reg := registers(a.inputs, a.outputs, a.counterType); reg != nil {
			ctx.Send(a.pubsub, reg)
		}

	case *MsgSendEvents:
		ctx.Send(a.pubsub, msg)
	case *messages.Event:
		if !a.disablePersistence && a.Recovering() {
			// a.flagRecovering = true
			scenario := "received replayed event"
			logs.LogBuild.Printf("%s, internal state changed to\n\tinputs -> '%v', outputs -> '%v'\n",
				scenario, a.inputs, a.outputs)
		} else {
			if !a.disablePersistence {
				a.PersistReceive(msg)
			}
			scenario := "received new message"
			logs.LogBuild.Printf("%s, internal state changed to\n\tinputs -> '%v', outputs -> '%v'\n",
				scenario, a.inputs, a.outputs)
		}
		// a.buildlogs.Log.Printf("data ->'%v', rawinputs -> '%v', rawoutputs -> '%v' \n",
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
				// a.inputs[id] = 1
				// if !a.Recovering() {
				// 	sendEvent(ctx, a.events, a.counterType, id, 1, messages.INPUT)
				// 	// ctx.Send(a.events, &messages.Event{Id: id, Type: messages.INPUT, Value: 1})
				// }
			} else if diff > 0 {
				if v, ok := a.puertas[uint(id)]; a.disableDoorGpio || !ok || v == a.openState[id] {
					a.inputs[id] += diff
					if a.disablePersistence || !a.Recovering() {
						sendEvent(ctx, a.events, a.counterType, id, diff, messages.INPUT)
						// ctx.Send(a.events, &messages.Event{Id: id, Type: messages.INPUT, Value: diff})
					}
				}
			} else if diff < 0 {
				// a.inputs[id] += msg.GetValue()
				if a.disablePersistence || !a.Recovering() {
					sendEvent(ctx, a.events, a.counterType, id, msg.GetValue(), messages.INPUT)
					// ctx.Send(a.events, msg)
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
				// a.outputs[id] = 1
				// if !a.Recovering() {
				// 	sendEvent(ctx, a.events, a.counterType, id, 1, messages.OUTPUT)
				// 	// ctx.Send(a.events, &messages.Event{Id: id, Type: messages.OUTPUT, Value: 1})
				// }
			} else if diff > 0 {
				if v, ok := a.puertas[uint(id)]; a.disableDoorGpio || !ok || v == a.openState[id] {
					a.outputs[id] += diff
					if a.disablePersistence || !a.Recovering() {
						sendEvent(ctx, a.events, a.counterType, id, diff, messages.OUTPUT)
						// ctx.Send(a.events, &messages.Event{Id: id, Type: messages.OUTPUT, Value: diff})
					}
				}
			} else if diff < 0 {
				// a.outputs[id] += msg.GetValue()
				if a.disablePersistence || !a.Recovering() {
					sendEvent(ctx, a.events, a.counterType, id, msg.GetValue(), messages.OUTPUT)
					// ctx.Send(a.events, msg)
				}
			}
			a.rawOutputs[id] = msg.GetValue()
		case messages.TAMPERING:
			logs.LogWarn.Println("shelteralarm")
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
		logs.LogWarn.Printf("counter keep alive error")
		ctx.Send(a.pubsub, msg)
	case *msgDoor:
		a.puertas[msg.id] = msg.value
		ctx.Send(a.events, msg)
	case *msgGPS:
		ctx.Send(a.events, msg)
		if a.gpsToConsole && bytes.HasPrefix(msg.data, []byte("$GPRMC")) {
			ctx.Send(a.listen, msg)
		}
	case *MsgToTest:
		// logs.LogBuild.Printf("test data: %q", msg)
		ctx.Send(a.listen, msg)
	case *msgEvent:
		// a.buildlogs.Log.Printf("\"%s\" - msg: '%q'\n", ctx.Self().GetId(), msg)
		ctx.Send(a.pubsub, msg)
	case *actor.Terminated:
		logs.LogWarn.Printf("actor terminated: %s", msg.GetWho().GetAddress())
	case *actor.Stopped:
		logs.LogWarn.Printf("actor stopped, reason: %s", msg)
	}
}

func verifySum(data map[int32]int64) int64 {
	sum := int64(0)
	for _, v := range data {
		sum += v
	}
	return sum
}

func sendEvent(ctx actor.Context, dts *actor.PID, tp int, id int32, value int64, msgType messages.Event_EventType) {
	idPuerta := id
	switch {
	case tp == 2 && id == 0:
		idPuerta = 1
	case tp == 2 && id == 1:
		return
	}
	ctx.Send(dts, &messages.Event{Id: idPuerta, Type: msgType, Value: value})
}

func registers(inputs, outputs map[int32]int64, tp int) *register {
	// log.Printf("counter type = %d", tp)
	reg := &register{}
	regv := make([]int64, 0)
	if tp == 2 {
		// log.Println("counter type 2")
		if _, ok := inputs[0]; !ok {
			return nil
		}
		if _, ok := outputs[0]; !ok {
			return nil
		}

		regv = append(regv, inputs[0])
		regv = append(regv, outputs[0])
	} else {
		sortInputs := SortMap(inputs)
		for _, v := range sortInputs {
			regv = append(regv, v)
		}
		sortOutputs := SortMap(outputs)
		for _, v := range sortOutputs {
			regv = append(regv, v)
		}
		if len(regv) < 4 {
			return nil
		}
	}
	reg.Registers = regv
	return reg
}
