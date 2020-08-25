package client

import (
	"fmt"
	"log"
	"time"

	"github.com/dumacp/sonar/contador"
)

//NewSerial connect with device serial
func NewSerial(port string, bautRate int) (*contador.Device, error) {
	dev, err := contador.NewDevice(port, bautRate)
	if err != nil {
		return nil, err
	}
	flagSuccess := false
	for range []int{1, 2, 3} {
		if err := dev.Connect(); err == nil {
			flagSuccess = true
			break
		}
		log.Println(err)
		time.Sleep(3 * time.Second)
	}
	if !flagSuccess {
		return nil, fmt.Errorf("don't connection with port serial")
	}
	return dev, nil
}

type Input struct {
	id    int
	value int64
}
type Output struct {
	id    int
	value int64
}
type Lock struct {
	id    int
	value int64
}

//Listen function to listen events
func Listen(quit chan int, dev *contador.Device, wError *log.Logger) <-chan interface{} {
	wError.Println("listennnnn port")

	ch := make(chan interface{})

	chRegisters := dev.ListenChannelv2(quit)

	go func() {
		defer close(ch)
		var countersMem []int
		for counters := range chRegisters {
			// verifysum := 0
			// for _, v := range counters[:5] {
			// 	verifysum += v
			// }
			// if verifysum <= 0 {
			// 	continue
			// }

			if len(countersMem) <= 0 || counters[0] > countersMem[0] {
				select {
				case ch <- &Input{id: 0, value: int64(counters[0])}:
				case <-time.After(1 * time.Second):
				}
			}
			if len(countersMem) <= 0 || counters[1] > countersMem[1] {
				select {
				case ch <- &Output{id: 0, value: int64(counters[1])}:
				case <-time.After(1 * time.Second):
				}
			}
			if len(countersMem) <= 0 || counters[2] > countersMem[2] {
				select {
				case ch <- &Input{id: 1, value: int64(counters[2])}:
				case <-time.After(1 * time.Second):
				}
			}
			if len(countersMem) <= 0 || counters[3] > countersMem[3] {
				select {
				case ch <- &Output{id: 1, value: int64(counters[3])}:
				case <-time.After(1 * time.Second):
				}
			}
			if len(countersMem) <= 0 || counters[14] > countersMem[14] {

				if len(countersMem) > 0 {
					id := 0
					if counters[13] > countersMem[13] {
						id = 1
					}
					select {
					case ch <- &Lock{id: id, value: int64(counters[14])}:
					case <-time.After(1 * time.Second):
					}
				}
			}
			countersMem = counters
		}
	}()

	return ch
}
