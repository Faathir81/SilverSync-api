package config

import (
	"os"

	"github.com/sirupsen/logrus"
)

var Logger = logrus.New()

func InitLogger() {
	Logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})
	Logger.SetOutput(os.Stdout)
	Logger.SetLevel(logrus.InfoLevel)
}
