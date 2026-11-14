package core

import "time"

// StopProcess solicita o fechamento e espera o jogo terminar ou dez segundos
// após o envio. O primeiro resultado indica se havia uma execução acompanhada.
func StopProcess(prog string) (found, stillRunning bool, err error) {
	activeProcesses.Lock()
	items := activeProcesses.items[prog]
	var control *stopControl
	if len(items) > 0 {
		control = items[0]
	}
	activeProcesses.Unlock()
	if control == nil || !control.request() {
		return false, false, nil
	}
	for {
		finished, sent, failure := control.status()
		if failure != nil {
			return true, false, failure
		}
		if finished {
			return true, false, nil
		}
		if !sent.IsZero() && time.Since(sent) >= 10*time.Second {
			return true, true, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// ForceStopProcess só aceita a força após o prazo do pedido normal. No Windows,
// a ação atinge apenas os processos associados ao grupo do jogo.
func ForceStopProcess(prog string) (bool, error) {
	if err := forceStopAvailable(); err != nil {
		return false, err
	}
	activeProcesses.Lock()
	items := activeProcesses.items[prog]
	var control *stopControl
	if len(items) > 0 {
		control = items[0]
	}
	activeProcesses.Unlock()
	if control == nil {
		return false, nil
	}
	accepted, err := control.requestForce(time.Now())
	if err != nil || !accepted {
		return accepted, err
	}
	for {
		finished, done, failure := control.forceStatus()
		if done {
			return failure == nil, failure
		}
		if finished {
			return false, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
}
