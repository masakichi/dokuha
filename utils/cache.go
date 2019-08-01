package utils

import (
	"encoding/gob"
	"os"
	"path/filepath"
)

var CacheDir string

func getCachePath(name string) string {
	return filepath.Join(CacheDir, name+".gob")
}

func SetCache(name string, e interface{}) error {
	path := getCachePath(name)
	cacheFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer cacheFile.Close()
	enc := gob.NewEncoder(cacheFile)
	enc.Encode(e)
	return nil
}

func LoadCache(name string, e interface{}) error {
	path := getCachePath(name)
	cacheFile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer cacheFile.Close()
	dec := gob.NewDecoder(cacheFile)
	dec.Decode(e)
	return nil
}
