package handler

import (
	"encoding/base64"
	"encoding/json"
	"github.com/miky4u2/RAagent/agent/common"
	"github.com/miky4u2/RAagent/agent/config"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Ctl HTTP handler function
//
func Ctl(w http.ResponseWriter, req *http.Request) {

	// Check if IP is allowed, abort if not. (Must be server IP)
	if !common.IsIPAllowed(req, config.Settings.ServerIP) {
		http.Error(w, http.StatusText(403), http.StatusForbidden)
		return
	}

	// POST method only
	if req.Method != "POST" {
		http.Error(w, http.StatusText(403), http.StatusForbidden)
		return
	}

	// Prepare response header
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	// Instantiate a ctlReq and ctlRes struct to be populated
	ctlReq := struct {
		Type string `json:"type"`
	}{}

	ctlRes := struct {
		Status    string   `json:"status"`
		ErrorMsgs []string `json:"errorMsgs"`
		Output    string   `json:"output"`
	}{}

	// Populate the ctlReq struct with received json request
	json.NewDecoder(req.Body).Decode(&ctlReq)

	// If control Type is invalid, abbort now, respond with error
	if ctlReq.Type != `status` && ctlReq.Type != `restart` && ctlReq.Type != `stop` {
		log.Println(`Received incorrect Type`)
		ctlRes.Status = "failed"
		ctlRes.ErrorMsgs = append(ctlRes.ErrorMsgs, `Invalid Type`)

		res, err := json.Marshal(ctlRes)
		if err != nil {
			log.Println(err)
		}
		w.Write(res)
		return
	}

	// If we get here, the received control Type is valid
	ctlRes.Status = `done`
	var output string

	// If control Type is status
	if ctlReq.Type == `status` {
		// Get list of available modules
		modulePath := filepath.Join(config.AppBasePath, `modules`)
		files, _ := ioutil.ReadDir(modulePath)

		output = `Version ` + config.Version + ` alive and kicking !!`
		output += "\nAvaiable modules: "
		for _, f := range files {
			output += `[` + f.Name() + `]`
		}
		output += "\n"
	}

	// If control Type is restart
	if ctlReq.Type == `restart` {
		output = `Version ` + config.Version + ` restarting now...` + "\n"
		emptyFile, _ := os.Create(filepath.Join(config.AppBasePath, `bin`, `agent_restart`))
		emptyFile.Close()
		log.Println(`Received Ctl Restart, RAagent will now attempt to restart...`)
		go func() { time.Sleep(2 * time.Second); os.Exit(0) }()
	}

	// If control Type is stop
	if ctlReq.Type == `stop` {
		output = `Version ` + config.Version + ` shutting down now...` + "\n"
		log.Println(`Received Ctl Stop, RAagent will now shutting down...`)
		go func() { time.Sleep(2 * time.Second); os.Exit(0) }()
	}

	// Encode output and send response
	ctlRes.Output = base64.StdEncoding.EncodeToString([]byte(output))
	res, err := json.Marshal(ctlRes)
	if err != nil {
		log.Println(err)
	}
	w.Write(res)

	return
}
