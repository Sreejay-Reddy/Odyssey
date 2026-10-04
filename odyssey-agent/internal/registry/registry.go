package registry

import "fmt"

type Registered struct {
	Target       string
	FunctionName string
	TargetID     uint32
	TTLMS		 uint32
}

type Registry struct {
	byID   map[uint32]*Registered
	byName map[string]*Registered
}

type RegistryMessage struct {
	SDKID      [16]byte
	SessionID  [16]byte
	Registry   []Registered
}

func New() *Registry {
	return &Registry{
		byID:   make(map[uint32]*Registered),
		byName: make(map[string]*Registered),
	}
}

func (r *Registry) Add(target Registered) error {
	if _, exists := r.byID[target.TargetID]; exists {
		return fmt.Errorf("target ID %d is already registered", target.TargetID)
	}

	if _, exists := r.byName[target.Target]; exists {
		return fmt.Errorf("target %q is already registered", target.Target)
	}

	r.byID[target.TargetID] = &target
	r.byName[target.Target] = &target

	return nil
}

func (r *Registry) GetByID(targetID uint32) (*Registered, error) {
	target, ok := r.byID[targetID]
	if !ok {
		return nil, fmt.Errorf("target ID %d is not registered", targetID)
	}

	return target, nil
}

func (r *Registry) GetByName(target string) (*Registered, error) {
	registered, ok := r.byName[target]
	if !ok {
		return nil, fmt.Errorf("target %q is not registered", target)
	}

	return registered, nil
}