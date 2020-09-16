/*
Package implements a binary for read serial port nmea.

*/
package contador

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tarm/serial"
)

type Device struct {
	port *serial.Port
	conf *serial.Config
	ok   bool
	acc  []int
	reg  []int
}

func NewDevice(portName string, baudRate int) (*Device, error) {
	// log.Println("port serial config ...")
	config := &serial.Config{
		Name: portName,
		Baud: baudRate,
		//ReadTimeout: time.Second * 3,
	}
	/**
	s, err := serial.OpenPort(config)
	if err != nil {
		return nil, err
	}
	/**/
	dev := &Device{
		conf: config,
		ok:   false,
		acc:  make([]int, 4),
		reg:  make([]int, 8),
	}
	return dev, nil
}

func (dev *Device) Connect() error {
	s, err := serial.OpenPort(dev.conf)
	if err != nil {
		return err
	}
	dev.port = s
	dev.ok = true
	return nil
}

func (dev *Device) Close() bool {
	dev.ok = false
	if err := dev.port.Close(); err != nil {
		// log.Println(err)
		return false
	}
	return true
}

func (dev *Device) ListenRegisters() {

	ch := dev.read()
	first := true
	for v := range ch {
		// log.Printf("trama: %s\n", v)
		if strings.Contains(v, "RPT") {
			split := strings.Split(v, ";")
			data := make([]int, 8)
			ok := true
			for i, _ := range data {
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
				dev.reg = data
				first = false
				continue
			}

			for i := 0; i < 4; i++ {
				if data[i] >= dev.reg[i] {
					dev.acc[i] += data[i] - dev.reg[i]
				} else {
					if data[i+4] >= dev.reg[i+4] {
						dev.acc[i] += data[i+4] - dev.reg[i+4]
					} else {
						dev.acc[i] = 0
					}
				}
			}
			dev.reg = data
		}
	}
}

func (dev *Device) ListenChannel() <-chan []int {

	registers := make(chan []int, 0)
	acc := make([]int, 4)
	lastacc := make([]int, 4)
	ch := dev.read()
	go func() {
		first := true
		for v := range ch {
			// fmt.Printf("trama: %s\n", v)
			if strings.Contains(v, "RPT") {
				split := strings.Split(v, ";")
				data := make([]int, 8)
				ok := true
				for i, _ := range data {
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
					dev.reg = data
					first = false
					continue
				}

				for i := 0; i < 4; i++ {
					if data[i] >= dev.reg[i] {
						acc[i] += data[i] - dev.reg[i]
					} else {
						if data[i+4] >= dev.reg[i+4] {
							acc[i] += data[i+4] - dev.reg[i+4]
						} else {
							acc[i] = 0
						}
					}
				}
				dev.reg = data
			}
			sumLast := 0
			sumNow := 0
			for i, v := range lastacc {
				sumLast += v
				sumNow += acc[i]
			}

			if sumLast == sumNow {
				continue
			}
			select {
			case registers <- acc:
				acc = []int{0, 0, 0, 0}
			default:
			}

		}
	}()
	return registers
}
func (dev *Device) Contadores() ([]int, error) {
	if !dev.ok {
		return nil, fmt.Errorf("device Error")
	}
	temp := dev.acc
	dev.acc = make([]int, 4)
	return temp, nil
}

func (dev *Device) Registros() ([]int, error) {
	if !dev.ok {
		return nil, fmt.Errorf("device Error")
	}

	return dev.reg, nil
}

func (dev *Device) ContadorUP_puerta1() (int, error) {
	if !dev.ok {
		return -1, fmt.Errorf("device Error")
	}
	temp := dev.acc[0]
	dev.acc[0] = 0
	return temp, nil
}

func (dev *Device) ContadorUP_puerta2() (int, error) {
	if !dev.ok {
		return -1, fmt.Errorf("device Error")
	}
	temp := dev.acc[2]
	dev.acc[2] = 0
	return temp, nil
}

func (dev *Device) ContadorDOWN_puerta1() (int, error) {
	if !dev.ok {
		return -1, fmt.Errorf("device Error")
	}
	temp := dev.acc[1]
	dev.acc[1] = 0
	return temp, nil
}

func (dev *Device) ContadorDOWN_puerta2() (int, error) {
	if !dev.ok {
		return -1, fmt.Errorf("device Error")
	}
	temp := dev.acc[3]
	dev.acc[3] = 0
	return temp, nil
}

func (dev *Device) read() chan string {

	if !dev.ok {
		// log.Println("Device is closed")
		return nil
	}
	ch := make(chan string)

	//buf := make([]byte, 128)

	countError := 0
	go func() {
		defer close(ch)
		bf := bufio.NewReader(dev.port)
		for {
			b, err := bf.ReadBytes('<')
			if err != nil {
				// log.Println(err)
				if countError > 3 {
					dev.Close()
					return
				}
				time.Sleep(1 * time.Second)
				countError++
				continue
			}
			data := string(b[:])
			// log.Printf("serial input: %q\n", data)
			ch <- data
		}
	}()
	return ch
}
