package config

import (
	"errors"
	"path"
	"strings"
)

// NewReader returns a type that will hydrate an ApplicationConfig instance from a file.
// Currently only "json" is supported as a value for the readerType parameter. Intention
// is to extend to TOML at some point.
func NewReader(readerType string) (Reader, error) {
	var reader Reader
	if readerType == "json" {
		reader = newJsonReader()
	} else {
		return reader, errors.New("Unsupported readerType value: " + readerType)
	}
	return reader, nil
}

func GetFileExtension(cfgPath string) string {
	tokens := strings.Split(path.Base(cfgPath), ".")
	if len(tokens) == 2 {
		return tokens[1]
	}
	return tokens[0]
}