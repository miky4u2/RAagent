package tasks

import (
	"encoding/json"
	"github.com/miky4u2/RAagent/agent/common"
	"github.com/miky4u2/RAagent/agent/config"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

type statusReq struct {
	UUID string `json:"uuid"`
}

type statusRes struct {
	Status    string   `json:"status"`
	ErrorMsgs []string `json:"errorMsgs"`
	Task      taskRes  `json:"task"`
}

// Status HTTP handler function
//
func Status(w http.ResponseWriter, req *http.Request) {
	var errMsgs []string

	// Check if IP is allowed, abort if not
	if !common.IsIPAllowed(req, config.Settings.AllowedIPs) {
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

	// Instantiate a taskStatusReq and taskStatusRes struct to be populated
	statusReq := statusReq{}
	response := statusRes{}

	// Populate the statusReq struct with received json request
	json.NewDecoder(req.Body).Decode(&statusReq)

	// Validate UUID and Populate task struct with task status
	task := taskRes{}

	// Validate received data
	if !regexp.MustCompile(`^[a-z0-9\-]+$`).MatchString(statusReq.UUID) {
		errMsgs = append(errMsgs, `'UUID' incorrect format`)
	} else {
		taskFile, err := os.Open(filepath.Join(config.AppBasePath, "tasks", statusReq.UUID+".status"))
		defer taskFile.Close()

		if err != nil {
			errMsgs = append(errMsgs, statusReq.UUID+` task does not exist`)
			log.Println(err)
		} else {
			jasonParser := json.NewDecoder(taskFile)
			err = jasonParser.Decode(&task)
			if err != nil {
				errMsgs = append(errMsgs, `Cannot parse data for task `+statusReq.UUID)
				log.Println(err)
			}
		}
	}

	// If we encountered errors, abort and respond with found errors
	if len(errMsgs) > 0 {
		response.Status = `Failed`
		response.ErrorMsgs = errMsgs

		res, err := json.Marshal(response)
		if err != nil {
			log.Println(err)
		}
		w.Write(res)

		return
	}

	// Respond with task status
	response.Status = `ok`
	response.Task = task

	res, err := json.Marshal(response)
	if err != nil {
		log.Println(err)
	}
	w.Write(res)

}
