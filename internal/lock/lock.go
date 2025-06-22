package lock

import (
	"os"

	"github.com/Phillezi/tunman-remaster/internal/defaults"
	"github.com/gofrs/flock"
	"go.uber.org/zap"
)

var (
	lockFile *flock.Flock
)

// Acquire tries to lock the daemon. Returns fatal if lock is held.
func Acquire() {
	lockFile = flock.New(defaults.LockPath)

	locked, err := lockFile.TryLock()
	if err != nil {
		zap.L().Fatal("failed to acquire lock", zap.Error(err))
	}
	if !locked {
		zap.L().Fatal("another instance is already running")
	}
}

// Release unlocks and removes the lock file.
func Release() {
	if lockFile != nil {
		_ = lockFile.Unlock()
		_ = os.Remove(lockFile.Path())
	}
}
