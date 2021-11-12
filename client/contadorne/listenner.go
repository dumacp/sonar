package contadorne

import (
	"time"

	"github.com/dumacp/sonar/client/logs"

	"github.com/dumacp/turnstilene"
)

func NewDev(port string, baudRate int) (turnstilene.Device, error) {
	dev, err := turnstilene.Turnstile(port, baudRate)
	if err != nil {
		return nil, err
	}
	return dev, nil
}

type Counter struct {
	Type  turnstilene.Events
	Value interface{}
}

func Listen(dev turnstilene.Device) chan *Counter {

	registers := dev.Listen()
	chCounters := make(chan *Counter, 0)

	for vinputs := range registers {

		go func(input turnstilene.Event) {

			logs.LogBuild.Printf("event: %v\n", input)

			select {
			case chCounters <- &Counter{
				Type:  input.Type,
				Value: input.Value,
			}:
			case <-time.After(3 * time.Second):
				logs.LogWarn.Println("timeout event report")
			}

		}(vinputs)
	}
	return chCounters
}
