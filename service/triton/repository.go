package triton

import (
	"sync"
)

type mlyModelID string
type tritonModelName string

type Repository struct {
	mu    sync.Mutex
	usage map[tritonModelName]map[mlyModelID]struct{}
}

func (r *Repository) RegisterUsage(mlyID mlyModelID, tritonName tritonModelName) {
	r.mu.Lock()
	defer r.mu.Unlock()
	mlyUsages, ok := r.usage[tritonName]
	if !ok {
		mlyUsages = make(map[mlyModelID]struct{})
		r.usage[tritonName] = mlyUsages
	}

	mlyUsages[mlyID] = struct{}{}
}

// UnregisterUsage returns true if all usages of a model have been unregistered.
// The TritonClient should then actual unload the model on the server.
func (r *Repository) UnregisterUsage(mlyID mlyModelID, tritonName tritonModelName) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	mlyUsages, ok := r.usage[tritonName]
	if !ok {
		// this was never registered, so this is considered having been unregistered.
		return true
	}

	delete(mlyUsages, mlyID)
	return len(mlyUsages) == 0
}

func NewRepository() *Repository {
	return &Repository{
		usage: make(map[tritonModelName]map[mlyModelID]struct{}),
		mu:    sync.Mutex{},
	}
}
