package core

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestStopControlAguardaEnvioEDezSegundos(t *testing.T) {
	var c stopControl
	start := time.Unix(100, 0)
	if !c.request() {
		t.Fatal("pedido recusado")
	}
	if _, err := c.requestForce(start.Add(time.Hour)); err == nil {
		t.Fatal("força aceita antes do envio")
	}
	if err := c.takeRequest(func() error { return errors.New("falha") }, func() time.Time { return start }); err == nil {
		t.Fatal("falha ignorada")
	}
	if _, err := c.requestForce(start.Add(time.Hour)); err == nil {
		t.Fatal("força aceita após falha")
	}
	if _, _, err := c.status(); err == nil {
		t.Fatal("falha não disponível ao solicitante")
	}
	c = stopControl{}
	c.request()
	calls := 0
	send := func() error { calls++; return nil }
	if err := c.takeRequest(send, func() time.Time { return start }); err != nil {
		t.Fatal(err)
	}
	c.request()
	if err := c.takeRequest(send, func() time.Time { return start.Add(time.Second) }); err != nil || calls != 1 {
		t.Fatal(err, calls)
	}
	if _, err := c.requestForce(start.Add(10*time.Second - time.Nanosecond)); err == nil {
		t.Fatal("força antecipada")
	}
	if ok, err := c.requestForce(start.Add(10 * time.Second)); !ok || err != nil {
		t.Fatal(ok, err)
	}
	if !c.takeForce() || c.takeForce() {
		t.Fatal("pedido de força não foi consumido uma vez")
	}
	c.finish()
	if c.request() {
		t.Fatal("pedido aceito após encerramento")
	}
	if ok, err := c.requestForce(start.Add(time.Hour)); ok || err != nil {
		t.Fatal(ok, err)
	}
}

func TestStopControlPedidosConcorrentes(t *testing.T) {
	var c stopControl
	start := time.Unix(100, 0)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.request()
			_ = c.takeRequest(func() error { return nil }, func() time.Time { return start })
			_, _ = c.requestForce(start.Add(10 * time.Second))
		}()
	}
	wg.Wait()
	if !c.takeForce() {
		t.Fatal("pedido de força ausente")
	}
}
