package core

import (
	"errors"
	"sync"
	"time"
)

// stopControl coordena os pedidos com o laço que acompanha o processo. O prazo
// começa quando o pedido normal é enviado, não quando ele é solicitado.
type stopControl struct {
	mu        sync.Mutex
	requested bool
	sent      time.Time
	force     bool
	finished  bool
	stopErr   error
}

func (c *stopControl) request() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.finished {
		return false
	}
	c.requested = true
	return true
}

// takeRequest só marca o envio depois de a operação de fechamento ter sucesso.
func (c *stopControl) takeRequest(send func() error, now func() time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.finished || !c.requested || !c.sent.IsZero() || c.stopErr != nil {
		return nil
	}
	if err := send(); err != nil {
		c.stopErr = err
		return err
	}
	c.sent = now()
	return nil
}

func (c *stopControl) requestForce(now time.Time) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.finished {
		return false, nil
	}
	if c.sent.IsZero() || now.Sub(c.sent) < 10*time.Second {
		return false, errors.New("Aguarde o jogo responder ao pedido de fechamento.")
	}
	c.force = true
	return true, nil
}

func (c *stopControl) takeForce() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.finished || !c.force {
		return false
	}
	c.force = false
	return true
}

func (c *stopControl) finish() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.finished = true
}

func (c *stopControl) status() (bool, time.Time, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.finished, c.sent, c.stopErr
}
