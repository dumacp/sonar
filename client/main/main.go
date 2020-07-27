package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/persistence"
	"github.com/dumacp/sonar/client/business"
	"github.com/dumacp/sonar/client/logs"
	"golang.org/x/exp/errors/fmt"
)

const (
	showVersion = "1.0.12"
)

var debug bool
var logStd bool
var socket string
var pathdb string
var port string
var baudRate int
var version bool
var loglevel int
var isZeroOpenStateDoor0 bool
var isZeroOpenStateDoor1 bool
var disableDoorGpioListen bool
var typeCounterDoor int
var sendGpsToConsole bool
var simulate bool
var mqtt bool
var disablePersistence bool

func init() {
	flag.BoolVar(&debug, "debug", false, "debug enable")
	flag.BoolVar(&mqtt, "disablePublishEvents", false, "disable local to publish local events")
	flag.BoolVar(&disablePersistence, "disablePersistence", false, "disable persistence")
	flag.BoolVar(&logStd, "logStd", false, "log in stderr")
	flag.StringVar(&pathdb, "pathdb", "/SD/boltdbs/countingdb", "socket to listen events")
	flag.StringVar(&socket, "port", "/dev/ttyS2", "serial port")
	flag.IntVar(&baudRate, "baud", 19200, "baudrate")
	flag.IntVar(&typeCounterDoor, "typeCounterDoor", 0, "0: two counters (front and back), 1: front counter, 2: back counter")
	flag.IntVar(&loglevel, "loglevel", 0, "level log")
	flag.BoolVar(&version, "version", false, "show version")
	flag.BoolVar(&isZeroOpenStateDoor0, "zeroOpenStateDoor0", false, "Is Zero the open state in front door?")
	flag.BoolVar(&isZeroOpenStateDoor0, "zeroOpenStateDoor1", false, "Is Zero the open state in back door?")
	flag.BoolVar(&disableDoorGpioListen, "disableDoorGpioListen", false, "disable DoorGpio Listen")
	flag.BoolVar(&sendGpsToConsole, "sendGpsToConsole", false, "Send GPS frame to Sonar console?")
	flag.BoolVar(&simulate, "simulate", false, "Simulate Test data")
}

func main() {

	flag.Parse()

	if version {
		fmt.Printf("version: %s\n", showVersion)
		os.Exit(2)
	}
	initLogs(debug, logStd)

	var provider *provider
	var err error
	if !disablePersistence {
		provider, err = newProvider(pathdb, 10)
		if err != nil {
			log.Fatalln(err)
		}
	}

	rootContext := actor.EmptyRootContext

	listenner := business.NewListen(socket, baudRate)
	listenner.SendToConsole(sendGpsToConsole)
	// listenner.SetLogError(errlog).SetLogWarn(warnlog).SetLogInfo(infolog).SetLogBuild(buildlog)
	// if debug {
	// 	listenner.WithDebug()
	// }

	counting := business.NewCountingActor(listenner)
	counting.SetZeroOpenStateDoor0(isZeroOpenStateDoor0)
	counting.SetZeroOpenStateDoor1(isZeroOpenStateDoor1)
	counting.DisableDoorGpioListen(disableDoorGpioListen)
	counting.CounterType(typeCounterDoor)
	counting.SetGPStoConsole(sendGpsToConsole)

	// counting.SetLogError(errlog).SetLogWarn(warnlog).SetLogInfo(infolog).SetLogBuild(buildlog)
	// if debug {
	// 	counting.WithDebug()
	// }

	propsCounting := actor.PropsFromProducer(func() actor.Actor { return counting })
	if !disablePersistence {
		propsCounting = propsCounting.WithReceiverMiddleware(persistence.Using(provider))
	}

	pidCounting, err := rootContext.SpawnNamed(propsCounting, "counting")
	if err != nil {
		logs.LogError.Println(err)
	}

	if mqtt {
		rootContext.Send(pidCounting, &business.MsgSendEvents{Data: false})
	}

	// listenner := client.NewListen(socket, baudRate, pidCounting)
	// listenner.SetLogError(errlog).SetLogWarn(warnlog).SetLogInfo(infolog).SetLogBuild(buildlog)
	// if debug {
	// 	listenner.WithDebug()
	// }

	// propsListen := actor.PropsFromFunc(listenner.Receive)
	// pidListen, err := rootContext.SpawnNamed(propsListen, "listenner")
	// if err != nil {
	// 	errlog.Println(err)
	// }

	// time.Sleep(1 * time.Second)

	// rootContext.Send(pidListen, &messages.CountingActor{
	// 	Address: pidCounting.Address,
	// 	ID:      pidCounting.Id})

	time.Sleep(3 * time.Second)

	rootContext.Send(pidCounting, &business.MsgSendRegisters{})

	//TEST
	// rootContext.PoisonFuture(pidListen).Wait()
	// pidListen, err = rootContext.SpawnNamed(propsListen, "listenner")
	// if err != nil {
	// 	errlog.Println(err)
	// }

	logs.LogInfo.Printf("back door counter START --  version: %s\n", showVersion)

	go func() {
		t1 := time.NewTicker(5 * time.Second)
		defer t1.Stop()
		for range t1.C {
			rootContext.Send(pidCounting, &business.MsgSendRegisters{})
		}
	}()

	if simulate {
		day := time.Now().Day()
		inputsAll := 220449
		outputsAll := 24695
		inputs := 111
		outputs := 32

		funcUpdate := func() []byte {
			data := []byte(fmt.Sprintf(">S;0RPTC%d;%d;%d;10;0;%d;%d;0;0;10;8;3;0;0;0;34574;*",
				day, inputsAll, outputsAll, inputs, outputs))
			csum := byte(0)
			for _, v := range data {
				csum ^= v
			}
			data = append(data, []byte(fmt.Sprintf("%02X<", csum))...)
			data = append(data, []byte("\r\n")...)
			return data
		}

		go func() {
			tick1 := time.NewTicker(3 * time.Second)
			defer tick1.Stop()
			tick2 := time.NewTicker(10 * time.Second)
			defer tick2.Stop()
			tick3 := time.NewTicker(30 * time.Second)
			defer tick3.Stop()
			for {
				select {
				case <-tick1.C:
					// logs.LogBuild.Println(funcUpdate())
					rootContext.Send(pidCounting, &business.MsgToTest{Data: funcUpdate()})
				case <-tick2.C:
					inputs++
					inputsAll++
					// logs.LogBuild.Println(funcUpdate())
					rootContext.Send(pidCounting, &business.MsgToTest{Data: funcUpdate()})
				case <-tick3.C:
					outputs++
					outputsAll++
					// logs.LogBuild.Println(funcUpdate())
					rootContext.Send(pidCounting, &business.MsgToTest{Data: funcUpdate()})

				}
			}
		}()

	}

	// //TEST
	// {
	// 	msg1 := messages.Event{Id: 0, Value: 10, Type: messages.INPUT}
	// 	msg2 := messages.Event{Id: 0, Value: 1, Type: messages.OUTPUT}
	// 	msg3 := messages.Event{Id: 0, Value: 12, Type: messages.INPUT}
	// 	msg4 := messages.Event{Id: 0, Value: 5, Type: messages.OUTPUT}

	// 	rootContext.Send(pidCounting, &msg1)
	// 	rootContext.Send(pidCounting, &msg2)
	// 	rootContext.Send(pidCounting, &msg3)
	// 	rootContext.Send(pidCounting, &msg4)

	// }

	finish := make(chan os.Signal, 1)
	signal.Notify(finish, syscall.SIGINT)
	signal.Notify(finish, syscall.SIGTERM)
	<-finish
}
