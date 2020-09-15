package logs

import (
	"io/ioutil"
	"log"
	"os"
)

//LogError log error
var LogError = New("[ ERROR ] ", 3)

//LogWarn log Warning
var LogWarn = New("[ WARN ] ", 4)

//LogInfo log Info
var LogInfo = New("[ INFO ] ", 6)

//LogBuild log Debug
var LogBuild = New("[ BUILD ] ", 7)

//Logger struct to logger
type Logger struct {
	*log.Logger
}

//New create Logger
func New(prefix string, flag int) *Logger {
	return &Logger{log.New(os.Stderr, prefix, flag)}
}

//SetLogError set logs with ERROR level
func (logg *Logger) SetLogError(logger *log.Logger) {
	logg.Logger = logger
}

//Disable set logs with ERROR level
func (logg *Logger) Disable() {
	logg.Logger.SetOutput(ioutil.Discard)
}
