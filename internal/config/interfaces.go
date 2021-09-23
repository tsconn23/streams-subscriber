package config

type Reader interface {
	Read(filePath string, cfg Configuration) error
}

type Configuration interface {
	AsString() string
}