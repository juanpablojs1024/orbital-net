package memory

import (
	"context"
	"errors"
	"sync"
	"time"

	discovery "satellite-coms/pkg/registry"
)

type serviceName string
type instanceID string

type Registry struct {
	sync.RWMutex
	serviceAddrs map[serviceName]map[instanceID]*serviceInstance
}

type serviceInstance struct {
	hostPort   string
	lastActive time.Time
}

func NewRegistry() *Registry {
	return &Registry{serviceAddrs: map[serviceName]map[instanceID]*serviceInstance{}}
}

func (r *Registry) Register(ctx context.Context, serviceN string, instanceId string, hostPort string) error {
	r.Lock()
	defer r.Unlock()
	sName := serviceName(serviceN)
	iID := instanceID(instanceId)
	if _, ok := r.serviceAddrs[sName]; !ok {
		r.serviceAddrs[sName] = make(map[instanceID]*serviceInstance)
	}
	r.serviceAddrs[sName][iID] = &serviceInstance{hostPort: hostPort, lastActive: time.Now()}

	return nil
}

func (r *Registry) Deregister(ctx context.Context, instanceId string, serviceN string) error {
	r.Lock()
	defer r.Unlock()
	sName := serviceName(serviceN)
	iID := instanceID(instanceId)
	if _, ok := r.serviceAddrs[sName]; !ok {
		return nil
	}
	delete(r.serviceAddrs[sName], iID)
	return nil
}

func (r *Registry) ReportHealthyState(instanceId string, serviceN string) error {
	r.Lock()
	defer r.Unlock()
	sName := serviceName(serviceN)
	iID := instanceID(instanceId)
	if _, ok := r.serviceAddrs[sName]; !ok {
		return errors.New("Service is not registered yet")
	}
	if _, ok := r.serviceAddrs[sName][iID]; !ok {
		return errors.New("Service instance is not registered yet")
	}
	r.serviceAddrs[sName][iID].lastActive = time.Now()
	return nil
}

func (r *Registry) ServiceAddress(ctx context.Context, serviceN string) ([]string, error) {
	r.RLock()
	defer r.RUnlock()
	sName := serviceName(serviceN)
	if len(r.serviceAddrs[sName]) == 0 {
		return nil, discovery.ErrNotFound
	}
	var res []string
	for _, i := range r.serviceAddrs[sName] {
		if i.lastActive.Before(time.Now().Add(-5 * time.Second)) {
			continue
		}
		res = append(res, i.hostPort)
	}
	return res, nil
}
