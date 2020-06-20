package handler

import (
	"archive/tar"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"errors"
	"github.com/miky4u2/RAagent/agent/common"
	"github.com/miky4u2/RAagent/agent/config"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Update HTTP handler function
//
func Update(w http.ResponseWriter, req *http.Request) {

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

	// Instantiate a updateReq and updateRes struct to be populated
	updateReq := struct {
		Type string `json:"type"`
	}{}

	updateRes := struct {
		Status    string   `json:"status"`
		ErrorMsgs []string `json:"errorMsgs"`
	}{}

	// Populate the updateReq struct with received json request
	json.NewDecoder(req.Body).Decode(&updateReq)

	// If action is invalid, abbort now
	if updateReq.Type != `full` && updateReq.Type != `modules` {
		log.Println(`Received incorrect update type`)
		updateRes.Status = "failed"
		updateRes.ErrorMsgs = append(updateRes.ErrorMsgs, `Invalid update type`)

		res, err := json.Marshal(updateRes)
		if err != nil {
			log.Println(err)
		}
		w.Write(res)
		return
	}

	// Pull tar.gz update file from server
	archivePath := filepath.Join(config.AppBasePath, `temp`, `update.tar.gz`)
	err := downloadUpdateFile(config.Settings.ServerURL, config.Settings.AgentID, archivePath, config.Settings.ValidateServerTLS)
	if err != nil {
		log.Println(`Error downloading update archive from server -`, err)
		updateRes.Status = "failed"
		updateRes.ErrorMsgs = append(updateRes.ErrorMsgs, `Error downloading update archive from server`)
		res, err := json.Marshal(updateRes)
		if err != nil {
			log.Println(err)
		}
		w.Write(res)
		return
	}

	// Delete all existing modules before updating
	modulesPath := filepath.Join(config.AppBasePath, `modules`)
	common.DeleteOldFiles(modulesPath, 0)

	// Update files
	err = updateFiles(archivePath, updateReq.Type)
	if err != nil {
		log.Println(`Error updating files -`, err)
		updateRes.Status = "failed"
		updateRes.ErrorMsgs = append(updateRes.ErrorMsgs, `Error updating files`)
		res, err := json.Marshal(updateRes)
		if err != nil {
			log.Println(err)
		}
		w.Write(res)
		return
	}

	// Remove tar.gz archive, ignore error in case the archive is missing
	_ = os.Remove(archivePath)

	// Send response
	updateRes.Status = "done"
	res, err := json.Marshal(updateRes)
	if err != nil {
		log.Println(err)
	}
	w.Write(res)
	log.Println(`Updates were successfully applied`)

	// if full update was requested, exit now and hope for the best. The startagent should take over
	// by updating the binary with the update's binary and restart
	if updateReq.Type == `full` {
		log.Println(`Restarting to finish executable update...`)
		go func() { time.Sleep(1 * time.Second); os.Exit(0) }()
	}

}

// Downloads update tar.gz file from server
//
func downloadUpdateFile(serverURL string, agentID string, destPath string, validateServerTLS bool) error {
	// Send download request to server to pull tar.gz archive file
	url := serverURL + `/api/download`
	downloadReqString := `{"agentID":"` + agentID + `","archive":"update"}`

	downloadReq, err := http.NewRequest("POST", url, strings.NewReader(downloadReqString))
	if err != nil {
		return err
	}
	
	downloadReq.Header.Set("Content-Type", "application/json")

	// Do we validate the server TLS certificate? true/false
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !validateServerTLS},
	}
	client := &http.Client{Transport: tr}
	downloadRes, err := client.Do(downloadReq)
	if err != nil {
		return err
	}
	defer downloadRes.Body.Close()

	// Check that we receive a status code 200
	if downloadRes.StatusCode != http.StatusOK {
		return errors.New(`Received incorrect status code while downloading update from server: ` + strconv.Itoa(downloadRes.StatusCode))
	}

	// Save tar.gz update archive file in temp folder
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, downloadRes.Body)
	if err != nil {
		return err
	}

	return err
}

// Extracts files from archive and update local files
//
func updateFiles(archivePath string, updateType string) error {

	// Unpack tar.gz update file
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gzf, err := gzip.NewReader(f)
	if err != nil {
		return err
	}

	tarReader := tar.NewReader(gzf)

	// Loop through archive paths
	for true {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		name := header.Name

		// Only process regular files
		if header.Typeflag == tar.TypeReg {

			var destpath string
			var mode int

			// Only copy those files for full updates
			if updateType == `full` {
				if strings.Contains(name, `cert.pem`) {
					log.Println(`Updating TLS certiticate`)
					destpath = filepath.Join(config.AppBasePath, `conf`, `cert.pem`)
					mode = int(0700)
					err = fileCopy(tarReader, destpath, os.FileMode(mode))
					if err != nil {
						return err
					}
				}
				if strings.Contains(name, `key.pem`) {
					log.Println(`Updating TLS key`)
					destpath = filepath.Join(config.AppBasePath, `conf`, `key.pem`)
					mode = int(0700)
					err = fileCopy(tarReader, destpath, os.FileMode(mode))
					if err != nil {
						return err
					}
				}
				if strings.Contains(name, `config.json`) {
					log.Println(`Updating config.json`)
					destpath = filepath.Join(config.AppBasePath, `conf`, `config.json`)
					mode = int(0700)
					err = fileCopy(tarReader, destpath, os.FileMode(mode))
					if err != nil {
						return err
					}
				}
				if strings.Contains(name, `bin/agent.exe`) {
					log.Println(`Preparing executable update`)
					destpath = filepath.Join(config.AppBasePath, `bin`, `update_agent.exe`)
					mode = int(0700)
					err = fileCopy(tarReader, destpath, os.FileMode(mode))
					if err != nil {
						return err
					}
				} else if strings.Contains(name, `bin/agent`) {
					log.Println(`Preparing executable update`)
					destpath = filepath.Join(config.AppBasePath, `bin`, `update_agent`)
					mode = int(0700)
					err = fileCopy(tarReader, destpath, os.FileMode(mode))
					if err != nil {
						return err
					}
				}
			}

			// Copy modules
			if strings.Contains(name, `modules/`) {
				moduleName := filepath.Base(name)
				log.Println(`Updating module:`, moduleName)
				destpath = filepath.Join(config.AppBasePath, `modules`, moduleName)
				mode = int(0700)
				err = fileCopy(tarReader, destpath, os.FileMode(mode))
				if err != nil {
					return err
				}
			}

		}

	}

	return err

}

// Copy a file from archive to final destination
//
func fileCopy(tarReader *tar.Reader, destPath string, perms os.FileMode) error {
	file, err := os.OpenFile(
		destPath,
		os.O_RDWR|os.O_CREATE|os.O_TRUNC,
		perms,
	)

	defer file.Close()

	if err != nil {
		return err
	}
	_, err = io.Copy(file, tarReader)

	return err
}
