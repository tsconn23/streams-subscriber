package interfaces

type StreamSubscriber interface {
	Close() error
	Connect() error
	Read() error
}
