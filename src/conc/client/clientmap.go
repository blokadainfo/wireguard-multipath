package client

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

type ClientMap struct {
	m map[uuid.UUID]*Client
	l sync.RWMutex
}

func NewClientMap() *ClientMap {
	return &ClientMap{
		m: make(map[uuid.UUID]*Client),
		l: sync.RWMutex{},
	}
}

func (cm *ClientMap) AddClient(cliendId uuid.UUID, c *Client) error {
	cm.l.Lock()
	defer cm.l.Unlock()

	_, ok := cm.m[cliendId]
	if ok {
		return fmt.Errorf("client with that ID already exists")
	}

	cm.m[cliendId] = c
	return nil
}

func (cm *ClientMap) GetClient(clientId uuid.UUID) (*Client, bool) {
	cm.l.RLock()
	defer cm.l.RUnlock()

	c, ok := cm.m[clientId]
	return c, ok
}

func (cm *ClientMap) DelClient(clientId uuid.UUID) (bool, error) {
	cm.l.Lock()
	defer cm.l.Unlock()

	if c, ok := cm.m[clientId]; ok {
		err := c.CloseWgRoutine() // TODO: should this be a goroutine?
		delete(cm.m, clientId)
		return true, err
	}

	return false, nil
}

func (cm *ClientMap) DelInactiveClients() {
	cm.l.Lock()
	defer cm.l.Unlock()

	var errs []error
	for clientId, c := range cm.m {
		if !c.IsActive() {
			slog.Info("Deleting inactive client", "client_id", clientId.String())
			err := c.CloseWgRoutine() // TODO: should this be a goroutine?
			errs = append(errs, fmt.Errorf("failed to close wg routine for client with id %v: %v", clientId.String(), err))
			delete(cm.m, clientId)
		}
	}
}
