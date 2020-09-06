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
	"github.com/dumacp/sonar/client"
	"github.com/dumacp/sonar/client/messages"
	"golang.org/x/exp/errors/fmt"
)

const (
	showVersion = "1.0.5"
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

func init() {
	flag.BoolVar(&debug, "debug", false, "debug enable")
	flag.BoolVar(&logStd, "logStd", false, "log in stderr")
	flag.StringVar(&pathdb, "pathdb", "/SD/boltdbs/countingdb", "socket to listen events")
	flag.StringVar(&socket, "port", "/dev/ttyS2", "serial port")
	flag.IntVar(&baudRate, "baud", 19200, "baudrate")
	flag.IntVar(&loglevel, "loglevel", 0, "level log")
	flag.BoolVar(&version, "version", false, "show version")
	flag.BoolVar(&isZeroOpenStateDoor0, "zeroOpenStateDoor0", false, "Is Zero the open state in front door?")
	flag.BoolVar(&isZeroOpenStateDoor0, "zeroOpenStateDoor1", false, "Is Zero the open state in back door?")
}

func main() {

	flag.Parse()

	if version {
		fmt.Printf("version: %s\n", showVersion)
		os.Exit(2)
	}
	initLogs(debug, logStd)

	provider, err := newProvider(pathdb, 10)
	if err != nil {
		log.Fatalln(err)
	}

	rootContext := actor.EmptyRootContext

	counting := client.NewCountingActor()
	counting.SetZeroOpenStateDoor0(isZeroOpenStateDoor0)
	counting.SetZeroOpenStateDoor1(isZeroOpenStateDoor1)
	counting.SetLogError(errlog).SetLogWarn(warnlog).SetLogInfo(infolog).SetLogBuild(buildlog)
	if debug {
		counting.WithDebug()
	}

	propsCounting := actor.PropsFromProducer(func() actor.Actor { return counting }).WithReceiverMiddleware(persistence.Using(provider))
	pidCounting, err := rootContext.SpawnNamed(propsCounting, "counting")
	if err != nil {
		errlog.Println(err)
	}

	listenner := client.NewListen(socket, baudRate, pidCounting)
	listenner.SetLogError(errlog).SetLogWarn(warnlog).SetLogInfo(infolog).SetLogBuild(buildlog)
	if debug {
		listenner.WithDebug()
	}

	propsListen := actor.PropsFromFunc(listenner.Receive)
	pidListen, err := rootContext.SpawnNamed(propsListen, "listenner")
	if err != nil {
		errlog.Println(err)
	}

	time.Sleep(1 * time.Second)

	rootContext.Send(pidListen, &messages.CountingActor{
		Address: pidCounting.Address,
		ID:      pidCounting.Id})

	time.Sleep(3 * time.Second)

	rootContext.Send(pidCounting, &client.MsgSendRegisters{})

	//TEST
	// rootContext.PoisonFuture(pidListen).Wait()
	// pidListen, err = rootContext.SpawnNamed(propsListen, "listenner")
	// if err != nil {
	// 	errlog.Println(err)
	// }

	infolog.Printf("back door counter START --  version: %s\n", showVersion)

	go func() {
		t1 := time.NewTicker(45 * time.Second)
		defer t1.Stop()
		for range t1.C {
			rootContext.Send(pidCounting, &client.MsgSendRegisters{})
		}
	}()

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
