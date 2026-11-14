package core

import "sync"

// activeProcesses inclui o intervalo de espera após o fim dos processos.
var activeProcesses = struct {
	sync.Mutex
	items map[string][]*stopControl
}{items: make(map[string][]*stopControl)}

func registerProcess(prog string, control *stopControl) func() {
	activeProcesses.Lock()
	activeProcesses.items[prog] = append(activeProcesses.items[prog], control)
	activeProcesses.Unlock()
	return func() {
		control.finish()
		activeProcesses.Lock()
		defer activeProcesses.Unlock()
		items := activeProcesses.items[prog]
		for i, item := range items {
			if item == control {
				items = append(items[:i], items[i+1:]...)
				break
			}
		}
		if len(items) == 0 {
			delete(activeProcesses.items, prog)
		} else {
			activeProcesses.items[prog] = items
		}
	}
}

// ProcessRunning informa se o programa ainda está sendo acompanhado.
func ProcessRunning(prog string) bool {
	activeProcesses.Lock()
	defer activeProcesses.Unlock()
	return len(activeProcesses.items[prog]) != 0
}
