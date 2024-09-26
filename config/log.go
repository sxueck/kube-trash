package config

import (
	log "github.com/sirupsen/logrus"
)

func InitLogComponent() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetReportCaller(true)
}
