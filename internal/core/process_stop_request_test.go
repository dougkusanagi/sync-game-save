package core

import (
	"errors"
	"testing"
	"time"
)

func TestStopProcessSemExecucao(t *testing.T) {
	found, running, err := StopProcess("ausente")
	if found || running || err != nil {
		t.Fatal(found, running, err)
	}
}

func TestStopProcessAguardaFechamento(t *testing.T) {
	c := &stopControl{}
	end := registerProcess("pedido-teste", c)
	defer end()
	result := make(chan bool, 1)
	go func() { found, running, err := StopProcess("pedido-teste"); result <- found && !running && err == nil }()
	deadline := time.Now().Add(2 * time.Second)
	for {
		c.mu.Lock()
		requested := c.requested
		c.mu.Unlock()
		if requested {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("pedido não chegou")
		}
		time.Sleep(time.Millisecond)
	}
	if err := c.takeRequest(func() error { return nil }, time.Now); err != nil {
		t.Fatal(err)
	}
	c.finish()
	select {
	case ok := <-result:
		if !ok {
			t.Fatal("resultado incorreto")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("pedido não terminou")
	}
}

func TestStopProcessRetornaFalha(t *testing.T) {
	c := &stopControl{}
	end := registerProcess("falha-teste", c)
	defer end()
	want := errors.New("falha")
	c.request()
	_ = c.takeRequest(func() error { return want }, time.Now)
	found, _, err := StopProcess("falha-teste")
	if !found || !errors.Is(err, want) {
		t.Fatal(found, err)
	}
}
