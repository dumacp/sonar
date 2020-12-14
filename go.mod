module github.com/dumacp/sonar

go 1.13

require (
	github.com/AsynkronIT/protoactor-go v0.0.0-20200815184336-b225d28383f2
	github.com/boltdb/bolt v1.3.1 // indirect
	github.com/brian-armstrong/gpio v0.0.0-20181227042754-72b0058bbbcb
	github.com/dumacp/go-actors v0.0.0-20200825200127-9826844a1688
	github.com/dumacp/pubsub v0.0.0-20200115200904-f16f29d84ee0
	github.com/eclipse/paho.mqtt.golang v1.2.0
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.4.2
	github.com/google/uuid v1.1.2 // indirect
	github.com/tarm/serial v0.0.0-20180830185346-98f6abe2eb07
	golang.org/x/exp/errors v0.0.0-20200917184745-18d7dbdd5567
)

replace github.com/brian-armstrong/gpio => /home/duma/go/src/github.com/brian-armstrong/gpio
