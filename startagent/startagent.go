package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Version of start_agent
const version = `1.0.0`

// AppBasePath is the base path of our application
var appBasePath string

// ProcessMonitor type
//
type processMonitor struct {
	CmdName string
	CmdArgs []string
	Output  *[]byte
	Err     error
}

func main() {

	// work out application base path
	appBasePath, _ = os.Executable()
	appBasePath = filepath.Dir(appBasePath)

	processStateListener := &processStateListener{monitor: make(chan bool)}
	for true {

		agentRestart := false

		// attempt to remove any update file that should not be here on initial startup
		_ = os.Remove(filepath.Join(appBasePath, `agent_update.exe`))
		_ = os.Remove(filepath.Join(appBasePath, `agent_update`))
		_ = os.Remove(filepath.Join(appBasePath, `agent_restart`))

		// Try to start the agent
		fork(processStateListener, filepath.Join(appBasePath, `agent.exe`))

		// waiting for monitor unblocking response when the agent exits gracefully
		_ = <-processStateListener.monitor

		// If an update binary exists, we assume that the agent placed it there and wishes to be updated and restarted
		if fileExists(filepath.Join(appBasePath, `update_agent.exe`)) {
			_ = os.Remove(filepath.Join(appBasePath, `agent.exe`))
			_ = os.Rename(filepath.Join(appBasePath, `update_agent.exe`), filepath.Join(appBasePath, `agent.exe`))
			agentRestart = true
		} else if fileExists(filepath.Join(appBasePath, `update_agent`)) {
			_ = os.Remove(filepath.Join(appBasePath, `agent`))
			_ = os.Rename(filepath.Join(appBasePath, `update_agent`), filepath.Join(appBasePath, `agent`))
			agentRestart = true
		}

		// If the agent wants us to restart
		if fileExists(filepath.Join(appBasePath, `agent_restart`)) {
			_ = os.Remove(filepath.Join(appBasePath, `agent_restart`))
			agentRestart = true
		}

		if agentRestart == false {
			break
		}

	}
}

// Forks a process for given command.
// Returns a processMonitor to the processStateListener
func fork(processStateListener *processStateListener, cmdName string, cmdArgs ...string) {
	go func() {
		processMonitor := &processMonitor{
			CmdArgs: cmdArgs,
			CmdName: cmdName,
		}
		args := strings.Join(cmdArgs, ",")
		command := exec.Command(cmdName, args)
		output, err := command.Output()
		if err != nil {
			processMonitor.Err = err
			processStateListener.OnError(processMonitor)
		}
		processMonitor.Output = &output
		processStateListener.OnComplete(processMonitor)
	}()
}

// ProcessStateListener type
//
type processStateListener struct {
	monitor chan bool
}

// Callback when process is completed
//
func (processStateListener *processStateListener) OnComplete(processMonitor *processMonitor) {
	processStateListener.monitor <- true
}

// Callback when process encounters an error
//
func (processStateListener *processStateListener) OnError(processMonitor *processMonitor) {
	log.Println("Error starting agent:", processMonitor.Err)
	os.Exit(2)
}

// fileExists checks if a file exists
//
func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
