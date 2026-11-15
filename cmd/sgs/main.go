// Comando sgs: sincroniza, abre um jogo e envia o save (ver internal/cli).
package main

import (
	"fmt"
	"os"

	"sync-game-save/internal/cli"
)

func main() {
	code, err := cli.Exec(os.Args[1:], os.Stdout, os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		code = 1
	}
	os.Exit(code)
}
