// record_lock_manager.go

package main

import "sync"

// RecordLockManager manages locks for DNS records to prevent race conditions.
type RecordLockManager struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

// AcquireLock acquires a lock for the given key.
func (rlm *RecordLockManager) AcquireLock(key string) {
	rlm.mu.Lock()
	if rlm.locks == nil {
		rlm.locks = make(map[string]*sync.Mutex)
	}
	if _, exists := rlm.locks[key]; !exists {
		rlm.locks[key] = &sync.Mutex{}
	}
	lock := rlm.locks[key]
	rlm.mu.Unlock()

	lock.Lock()
}

// ReleaseLock releases the lock for the given key.
func (rlm *RecordLockManager) ReleaseLock(key string) {
	rlm.mu.Lock()
	defer rlm.mu.Unlock()
	if lock, exists := rlm.locks[key]; exists {
		lock.Unlock()
	}
}
