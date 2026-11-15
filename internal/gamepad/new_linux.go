package gamepad

// New abre a leitura de controles do sistema (evdev).
func New() (Source, error) { return NewEvdev() }
