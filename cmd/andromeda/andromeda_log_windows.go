package main

import (
	"github.com/mattn/go-colorable"
	log "github.com/sirupsen/logrus"
)

func init() {
	// log.SetFormatter(&log.JSONFormatter{})
	formatter := &log.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	}
	log.SetFormatter(formatter)
	log.SetOutput(colorable.NewColorableStdout())
}
