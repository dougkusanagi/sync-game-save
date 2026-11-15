package gamepad

// New abre a leitura de controles do sistema.
func New() (Source, error) { return NewXInput() }
