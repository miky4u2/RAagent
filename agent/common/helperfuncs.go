package common

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

// Find takes a slice and looks for an element in it. Returns a bool.
//
func Find(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// FileExists checks if a file exists
//
func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// DeleteOldFiles deletes files older than provided days in provided dir
//
func DeleteOldFiles(dir string, days int) {
	tmpfiles, err := ioutil.ReadDir(dir)
	if err != nil {
		return
	}
	for _, file := range tmpfiles {
		if file.Mode().IsRegular() {
			if time.Now().Sub(file.ModTime()) > time.Duration(days*24)*time.Hour {
				// delete file
				_ = os.Remove(filepath.Join(dir, file.Name()))
			}
		}
	}
	return
}
