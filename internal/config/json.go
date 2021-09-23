package config

import (
	"encoding/json"
	"io/ioutil"
)

type jsonReader struct {
}

func newJsonReader() jsonReader {
	return jsonReader{}
}

func (r jsonReader) Read(filePath string, cfg Configuration) error {
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(file, cfg)
}