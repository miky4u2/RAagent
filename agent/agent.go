package main

import (
	"github.com/miky4u2/RAagent/agent/config"
	"github.com/miky4u2/RAagent/agent/webserver"
	"log"
	"os"
	"path/filepath"
)

// Size to truncate log file
const logMaxSize int64 = 500000

func main() {

	// Load configuration settings
	err := config.Settings.Load()
	if err != nil {
		log.Fatal("Error loading config file. ", err)
	}

	// log to file is true, open log file and set log output to file
	// If no log file path provided, use default
	if config.Settings.LogToFile {
		logPath := config.Settings.LogFile
		if logPath == "" {
			logPath = filepath.Join(config.AppBasePath, `log`, `agent.log`)
		}

		// Start with fresh log file if log file is bigger than 500k
		fs, err := os.Stat(logPath)
		if err == nil {
			if fs.Size() > logMaxSize {
				_ = os.Remove(logPath)
			}
		}

		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		log.SetOutput(f)
	}

	// Start server
	log.Println(`RAagent`, config.Version, `starting`)
	err = webserver.Start()
	if err != nil {
		log.Fatal("Error starting agent ", config.Version, ` : `, err)
	}
}
