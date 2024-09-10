package logging

import (
	"github.com/FimGroup/logging"
	"github.com/sirupsen/logrus"
)

var Manager logging.LoggerManager

func init() {
	manager, err := logging.NewLoggerManager("logs/bootstrap", 7, 50*1024*1024, 10, logrus.TraceLevel, true, true)
	if err != nil {
		panic(err)
	}
	Manager = manager
}
