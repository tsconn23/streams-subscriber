package interfaces

import (
	"context"
	"sync"
)

type StreamSubscriber interface {
	Close() error
	Connect() error
	Subscribe(ctx context.Context, wg *sync.WaitGroup) bool
}
