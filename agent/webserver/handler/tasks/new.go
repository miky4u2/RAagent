package tasks

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"github.com/miky4u2/RAagent/agent/common"
	"github.com/miky4u2/RAagent/agent/config"
	"github.com/satori/go.uuid"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

type taskReq struct {
	Name      string   `json:"name"`
	Mode      string   `json:"mode"`
	NotifyURL string   `json:"notifyURL"`
	Module    string   `json:"module"`
	Args      []string `json:"args"`
}

type taskRes struct {
	UUID      string   `json:"UUID"`
	Status    string   `json:"status"`
	ErrorMsgs []string `json:"errorMsgs"`
	StartTime string   `json:"startTime"`
	EndTime   string   `json:"endTime"`
	Duration  string   `json:"duration"`
	Output    string   `json:"output"`
	EndPoint  string   `json:"endPoint"`
	taskReq
}

// New HTTP handler function
//
func New(w http.ResponseWriter, req *http.Request) {

	startTime := time.Now()
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

	// Instantiate a task and taskRes response struct to be populated
	task := taskReq{}
	response := taskRes{}

	// Populate the task struct with received json request
	json.NewDecoder(req.Body).Decode(&task)

	// Build module path
	modulePath := filepath.Join(config.AppBasePath, `modules`, task.Module)

	// Create a new UUID for the task
	taskUUID := uuid.NewV4().String()

	// Validate received data
	if !regexp.MustCompile(`^[a-zA-Z0-9]+[a-zA-Z0-9\.\-_]*$`).MatchString(task.Module) {
		errMsgs = append(errMsgs, `'module' incorrect format. Must only contain [a-zA-Z0-9.-_] and not start with dot`)
	} else if !common.FileExists(modulePath) {
		errMsgs = append(errMsgs, `'module' not supported`)
	}

	if len(task.Name) < 1 || len(task.Name) > 100 {
		errMsgs = append(errMsgs, `'name' incorrect length, must be between 1 and 100 chars`)
	}
	if len(task.NotifyURL) > 200 {
		errMsgs = append(errMsgs, `'notifyURL' too long, max 255 chars`)
	}
	if len(task.NotifyURL) > 1 && !regexp.MustCompile(`^http(s?)\://`).MatchString(task.NotifyURL) {
		errMsgs = append(errMsgs, `'notifyURL' must start with http:// or https://`)
	}
	if task.Mode != "attached" && task.Mode != "detached" {
		errMsgs = append(errMsgs, `'mode' must be 'attached' or 'detached'`)
	}

	// Pre populate response
	response.Status = "in progress"
	response.UUID = taskUUID
	response.ErrorMsgs = errMsgs
	response.StartTime = startTime.Format("2006-01-02 15:04:05")
	response.EndPoint = req.RequestURI
	response.Name = task.Name
	response.Mode = task.Mode
	response.NotifyURL = task.NotifyURL
	response.Module = task.Module
	response.Args = task.Args

	// If we have any errors, abort now and send the response with the error messages
	if len(errMsgs) > 0 {
		endTime := time.Now()
		duration := endTime.Sub(startTime)

		response.Status = `failed`
		response.EndTime = endTime.Format("2006-01-02 15:04:05")
		response.Duration = duration.String()

		res, err := json.Marshal(response)
		if err != nil {
			log.Println(err)
		}
		w.Write(res)
		return
	}

	// If 'mode' is 'attached', execute module now and send response
	if task.Mode == "attached" {
		taskExec(&response, modulePath, startTime, config.Settings.TaskHistoryKeepDays, config.Settings.ValidateNotifyTLS)

		res, err := json.Marshal(response)
		if err != nil {
			log.Println(err)
		}
		w.Write(res)
		return
	}

	// If 'mode' is 'detached', instantiate a Go routine for this task's execution and
	// send 'in progress' response. The task status can then be monitored via api call to /tasks/status
	if task.Mode == "detached" {
		go taskExec(&response, modulePath, startTime, config.Settings.TaskHistoryKeepDays, config.Settings.ValidateNotifyTLS)

		// Send response
		res, err := json.Marshal(response)
		if err != nil {
			log.Println(err)
		}
		w.Write(res)
		return
	}

}

// Executes module
//
func taskExec(response *taskRes, modulePath string, startTime time.Time, taskHistoryKeepDays int, validateNotifyTLS bool) {

	var errMsgs []string

	// Create a new uuid.status file for this task
	// This file is used when a task status is queried via /tasks/status
	fileContent, err := json.MarshalIndent(response, "", " ")
	if err != nil {
		log.Println(err)
	}
	err = ioutil.WriteFile(filepath.Join(config.AppBasePath, "tasks", response.UUID+".status"), fileContent, 0644)
	if err != nil {
		log.Println(err)
	}

	cmd := exec.Command(modulePath, response.Args[0:]...)

	output := bytes.Buffer{}
	cmd.Stderr = &output
	cmd.Stdout = &output

	err = cmd.Run()
	if err != nil {
		errMsgs = append(errMsgs, `Module execution error: `+err.Error())
		log.Println(`Module '`+response.Module+`' execution error: `, err)
	}

	// Write response to uuid.status file
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	if len(errMsgs) > 0 {
		response.Status = "failed"
	} else {
		response.Status = "done"
	}
	response.ErrorMsgs = errMsgs
	response.Output = base64.StdEncoding.EncodeToString(output.Bytes())
	response.EndTime = endTime.Format("2006-01-02 15:04:05")
	response.Duration = duration.String()

	fileContent, err = json.MarshalIndent(response, "", " ")
	if err != nil {
		log.Println(err)
	}
	err = ioutil.WriteFile(filepath.Join(config.AppBasePath, "tasks", response.UUID+".status"), fileContent, 0644)
	if err != nil {
		log.Println(err)
	}

	// Notify URL if a url is provided
	if response.NotifyURL != "" {
		notifyURL(response, validateNotifyTLS)

	}

	// Tidy up, remove old task files
	common.DeleteOldFiles(filepath.Join(config.AppBasePath, "tasks"), taskHistoryKeepDays)

	return
}

// Sends json response to provided notifyURL
//
func notifyURL(response *taskRes, validateNotifyTLS bool) {

	url := response.NotifyURL
	notifyContent, _ := json.Marshal(response)

	notifyReq, err := http.NewRequest("POST", url, bytes.NewReader(notifyContent))
	if err != nil {
		log.Println(`Failed creating notification request to `+url, `:`, err)
		return
	}
	notifyReq.Header.Set("Content-Type", "application/json")

	// Do we validate the notifyURL TLS certificate? true/false
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !validateNotifyTLS},
	}
	client := &http.Client{Transport: tr}
	notifyRes, err := client.Do(notifyReq)
	if err != nil {
		log.Println(`Failed sending notification to `+url, `:`, err)
		return
	}
	defer notifyRes.Body.Close()

	// Check that we receive a status code 200
	if notifyRes.StatusCode != http.StatusOK {
		log.Println(`Failed sending notification to `+url+` : received status code`, strconv.Itoa(notifyRes.StatusCode))
	}

	return
}
