package bootstrap

import (
	"context"
	"sync"
)

// BootstrapHandler defines the contract each bootstrap handler must fulfill.  Implementation returns true if the
// handler completed successfully, false if it did not.
type BootstrapHandler func(
	ctx context.Context,
	wg *sync.WaitGroup) (success bool)
