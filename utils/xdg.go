package utils

import (
	"os"
	"path/filepath"
)

func EnsureDir(dirName string) {
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		os.MkdirAll(dirName, os.ModePerm)
	}
}

func GetCacheDir(name string) string {
	var basedir string
	if env := os.Getenv("XDG_CACHE_HOME"); env != "" {
		basedir = env
	} else {
		basedir = filepath.Join(os.Getenv("HOME"), ".cache")
	}
	return filepath.Join(basedir, name)
}
