package contador

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/dumacp/sonar/client/logs"
)

//NewSerial connect with device serial
func NewSerial(port string, bautRate int, readTimeout time.Duration) (*Device, error) {
	dev, err := NewDevice(port, bautRate, readTimeout)
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
	Id    int
	Value int64
}
type Output struct {
	Id    int
	Value int64
}
type Lock struct {
	Id    int
	Value int64
}

func SendData(dev *Device, data []byte) error {
	if err := dev.Send(data); err != nil {
		return err
	}
	return nil
}

//ListenRaw liste raw data
func ListenRaw(quit chan int, dev *Device) <-chan interface{} {

	ch := make(chan interface{}, 0)
	chlisten := dev.ListenRawChannel(quit)
	go func() {
		defer close(ch)
		first := true
		for v := range chlisten {

			logs.LogBuild.Printf("trama: %s\n", v)
			switch {
			case bytes.Contains(v, []byte("RPT")):
				split := strings.Split(string(v), ";")
				if len(split) < 17 {
					break
				}
				data := make([]int, 15)
				ok := true
				for i := range data {
					xint, err := strconv.Atoi(split[i+2])
					if err != nil {
						ok = false
						break
					}
					data[i] = xint
				}
				if !ok {
					continue
				}
				if first {
					//dev.reg = data
					first = false
					logs.LogInfo.Printf("actual door register: %s\n", v)
					// continue
				}
				select {
				case ch <- data:
				case <-time.After(time.Millisecond * 200):
				}
			}
		}
	}()
	return ch
}

//Listen function to listen events
func Listen(quit chan int, dev *Device) <-chan interface{} {
	logs.LogInfo.Println("listennnnn port")

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
				case ch <- &Input{Id: 0, Value: int64(counters[0])}:
				case <-time.After(1 * time.Second):
				}
			}
			if len(countersMem) <= 0 || counters[1] > countersMem[1] {
				select {
				case ch <- &Output{Id: 0, Value: int64(counters[1])}:
				case <-time.After(1 * time.Second):
				}
			}
			if len(countersMem) <= 0 || counters[2] > countersMem[2] {
				select {
				case ch <- &Input{Id: 1, Value: int64(counters[2])}:
				case <-time.After(1 * time.Second):
				}
			}
			if len(countersMem) <= 0 || counters[3] > countersMem[3] {
				select {
				case ch <- &Output{Id: 1, Value: int64(counters[3])}:
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
					case ch <- &Lock{Id: id, Value: int64(counters[14])}:
					case <-time.After(1 * time.Second):
					}
				}
			}
			countersMem = counters
		}
	}()

	return ch
}
