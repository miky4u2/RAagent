package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Settings : Configuration settings
//
var Settings config = config{}

// config type to load and hold configuration settings
//
type config struct {
	ServerIP            []string `json:"serverIP"`
	ServerURL           string   `json:"serverURL"`
	ValidateServerTLS   bool     `json:"validateServerTLS"`
	AgentID             string   `json:"agentID"`
	AgentBindIP         string   `json:"agentBindIP"`
	AgentBindPort       string   `json:"agentBindPort"`
	AllowedIPs          []string `json:"allowedIPs"`
	RateLimit           int      `json:"rateLimit"`
	RateLimitBurst      int      `json:"rateLimitBurst"`
	LogFile             string   `json:"logFile"`
	LogToFile           bool     `json:"logToFile"`
	ValidateNotifyTLS   bool     `json:"validateNotifyTLS"`
	TaskHistoryKeepDays int      `json:"taskHistoryKeepDays"`
}

// Version of RAagent
const Version = `0.1.1`

// AppBasePath is the base path of our application
var AppBasePath string

// Load configuration settings
//
func (c *config) Load() error {

	// work out application base path
	AppBasePath, _ = os.Executable()
	AppBasePath = filepath.Dir(AppBasePath)
	AppBasePath = strings.TrimSuffix(AppBasePath, `bin`)

	filename := filepath.Join(AppBasePath, "conf", "config.json")

	configFile, err := os.Open(filename)

	if err != nil {
		return err
	}

	defer configFile.Close()

	jasonParser := json.NewDecoder(configFile)
	err = jasonParser.Decode(c)

	if err != nil {
		return err
	}

	// Validate data
	if !regexp.MustCompile(`^[a-zA-Z0-9]+[a-zA-Z0-9\.\-_]*$`).MatchString(c.AgentID) {
		err = errors.New(`'agentID' must only contain [a-zA-Z0-9.-_] and not start with dot`)
		return err
	}

	if c.TaskHistoryKeepDays < 1 {
		c.TaskHistoryKeepDays = 7
	}
	if c.RateLimit < 1 {
		c.RateLimit = 1
	}
	if c.RateLimitBurst < c.RateLimit {
		c.RateLimitBurst = c.RateLimit
	}

	return err
}
