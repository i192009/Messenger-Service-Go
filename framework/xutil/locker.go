package xutil

import (
	"sync"
)

func HandlePanicAndUnlockRW(mutex *sync.RWMutex) {
	if r := recover(); r != nil {
		log.Errorf("panic: %v", r)
		mutex.Unlock()
		panic(r)
	}

	// Unlock the mutex
	mutex.Unlock()
}

func HandlePanicAndRUnlockRW(mutex *sync.RWMutex) {
	if r := recover(); r != nil {
		log.Errorf("panic: %v", r)
		mutex.RUnlock()
		panic(r)
	}

	// Unlock the mutex
	mutex.RUnlock()
}

func HandlePanicAndUnlock(mutex *sync.Mutex) {
	if r := recover(); r != nil {
		log.Errorf("panic: %v", r)
		mutex.Unlock()
		panic(r)
	}

	// Unlock the mutex
	mutex.Unlock()
}
