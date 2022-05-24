package ins50

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/tarm/serial"
)

const timeout = 30 * time.Second

type Device interface {
	ReadData() (*DeviceData, error)
	Close()
}

type device struct {
	conf *serial.Config
	port *serial.Port
}

func NewDevice(port string, baud int) (Device, error) {

	conf := &serial.Config{Name: port, Baud: baud, ReadTimeout: timeout}

	p, err := serial.OpenPort(conf)
	if err != nil {
		return nil, err
	}

	return &device{conf: conf, port: p}, nil
}

func (dev *device) Close() {
	dev.port.Close()
}

func (dev *device) ReadData() (*DeviceData, error) {
	if _, err := dev.port.Write([]byte(">S;AQPCV<")); err != nil {
		return nil, err
	}
	r := bufio.NewReader(dev.port)
	resp, err := r.ReadBytes('<')
	if err != nil {
		return nil, err
	}
	if !bytes.Contains(resp, []byte("ARP")) {
		return nil, fmt.Errorf("bad response: %q", resp)
	}
	varSplit := bytes.Split(resp, []byte(";"))

	if len(varSplit) < 10 {
		return nil, fmt.Errorf("length response is wrong: len = %d", len(varSplit))
	}

	re, err := regexp.Compile(`[0-9]+`)
	if err != nil {
		return nil, err
	}
	numbers := make([]int, 0)
	for i := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9} {
		if v := re.Find(varSplit[i]); v != nil {
			num, err := strconv.Atoi(string(v))
			if err != nil {
				return nil, err
			}
			numbers = append(numbers, num)
		}
	}

	return &DeviceData{Data: numbers}, nil
}
