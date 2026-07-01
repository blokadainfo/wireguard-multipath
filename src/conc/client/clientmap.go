package client

import (
	"errors"
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

var (
	ErrClientAlreadyExists = errors.New("client with that ID already exists")
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
		return ErrClientAlreadyExists
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
		err := c.CloseWgRoutine()
		delete(cm.m, clientId)
		return true, err
	}

	return false, nil
}

func (cm *ClientMap) DelInactiveClients() {
	cm.l.Lock()
	defer cm.l.Unlock()

	for clientId, c := range cm.m {
		if !c.IsActive() {
			slog.Info("Deleting inactive client", "client_id", clientId.String())

			if err := c.CloseWgRoutine(); err != nil {
				slog.Error("Failed to close wg routine", "client_id", clientId.String(), "error", err)
			}

			delete(cm.m, clientId)
		}
	}
}
