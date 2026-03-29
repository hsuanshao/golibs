package ctx

import (
	"github.com/sirupsen/logrus"
)

// GetLogLevel returns log set level
func GetLogLevel() logrus.Level {
	l := logrus.GetLevel()

	return l
}

// SetDebugLevel for developer debug use case
func SetDebugLevel() {
	logrus.SetFormatter(&logrus.TextFormatter{ForceColors: true, FullTimestamp: true})
}
