// concurrency.go

package main

import (
	"sync"
)

// RecordLockManager manages locks for DNS records based on their FQDN or PTR names.
type RecordLockManager struct {
	locks sync.Map // map[string]*sync.Mutex
}

// AcquireLock acquires a mutex for the given key.
func (rlm *RecordLockManager) AcquireLock(key string) {
	mutexInterface, _ := rlm.locks.LoadOrStore(key, &sync.Mutex{})
	mutex := mutexInterface.(*sync.Mutex)
	mutex.Lock()
}

// ReleaseLock releases the mutex for the given key.
func (rlm *RecordLockManager) ReleaseLock(key string) {
	// Retrieve the mutex without removing it to avoid race conditions
	if mutexInterface, ok := rlm.locks.Load(key); ok {
		mutex := mutexInterface.(*sync.Mutex)
		mutex.Unlock()
	}
}
