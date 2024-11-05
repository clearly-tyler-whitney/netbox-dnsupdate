// concurrency.go

package main

import (
	"sync"
)

// RecordLockManager manages locks for DNS records based on their FQDN.
type RecordLockManager struct {
	locks sync.Map // map[string]*sync.Mutex
}

// AcquireLock acquires a mutex for the given fqdn.
func (rlm *RecordLockManager) AcquireLock(fqdn string) *sync.Mutex {
	mutexInterface, _ := rlm.locks.LoadOrStore(fqdn, &sync.Mutex{})
	mutex := mutexInterface.(*sync.Mutex)
	mutex.Lock()
	return mutex
}

// ReleaseLock releases the mutex for the given fqdn.
func (rlm *RecordLockManager) ReleaseLock(fqdn string, mutex *sync.Mutex) {
	mutex.Unlock()
	// Optional: Clean up the mutex from the map to prevent memory leaks.
	// Only do this if you are sure no other goroutines are waiting on it.
}
