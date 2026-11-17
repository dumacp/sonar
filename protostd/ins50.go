package protostd

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tarm/serial"
)

type Device interface {
	ReadData() (*DeviceData, error)
	Open() error
	Close() error
	Conf() *serial.Config
}

type device struct {
	conf *serial.Config
	port *serial.Port
}

func NewDevice(port string, baud int, timeout time.Duration) Device {

	conf := &serial.Config{Name: port, Baud: baud, ReadTimeout: timeout}

	return &device{conf: conf}
}

func (dev *device) Open() error {

	if dev.conf == nil {
		return fmt.Errorf("serial device conf is empty")
	}
	p, err := serial.OpenPort(dev.conf)
	if err != nil {
		return err
	}

	dev.port = p

	return nil
}

func (dev *device) Close() error {
	return dev.port.Close()
}

func (dev *device) Conf() *serial.Config {
	return dev.conf
}

func (dev *device) ReadData() (*DeviceData, error) {

	r := bufio.NewReader(dev.port)
	response, err := r.ReadBytes('<')
	if err != nil {
		return nil, err
	}
	fmt.Printf("raw data: %q\n", response)

	if !strings.Contains(string(response), "RPT") {
		fmt.Printf("unkown response: %q\n", response)
		return nil, nil
	}

	buff := bytes.NewBuffer(response)

	scan := bufio.NewScanner(buff)

	result := make([][]byte, 0)

	for scan.Scan() {
		result = append(result, scan.Bytes())
	}

	if len(result) <= 0 || len(result[len(result)-1]) <= 0 {
		return nil, fmt.Errorf("bad response: %q", response)
	}

	resp := result[len(result)-1]

	varSplit := bytes.Split(resp, []byte(";"))

	if len(varSplit) < 17 {
		return nil, fmt.Errorf("length response is wrong: len = %d", len(varSplit))
	}

	re, err := regexp.Compile(`[0-9]+`)
	if err != nil {
		return nil, err
	}
	numbers := make([]int, 0)
	for i := range make([]int, len(varSplit)-2) {
		if v := re.Find(varSplit[i+2]); v != nil {
			num, err := strconv.Atoi(string(v))
			if err != nil {
				return nil, err
			}
			numbers = append(numbers, num)
		}
	}

	return &DeviceData{Data: numbers, Raw: resp}, nil
}
