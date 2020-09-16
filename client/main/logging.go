package main

import (
	"log"
	"log/syslog"

	"github.com/dumacp/sonar/client/logs"
)

func newLog(logger *logs.Logger, prefix string, flags int, priority int) error {

	logg, err := syslog.NewLogger(syslog.Priority(priority), flags)
	if err != nil {
		return err
	}
	logger.SetLogError(logg)
	return nil
}

func initLogs(debug, logStd bool) {
	if logStd {
		return
	}
	newLog(logs.LogInfo, "[ warn ] ", log.LstdFlags, 4)
	newLog(logs.LogWarn, "[ info ] ", log.LstdFlags, 6)
	newLog(logs.LogError, "[ build ] ", log.LstdFlags, 7)
	newLog(logs.LogBuild, "[ error ] ", log.LstdFlags, 3)
	if !debug {
		logs.LogBuild.Disable()
	}
}
