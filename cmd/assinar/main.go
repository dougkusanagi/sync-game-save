// Comando assinar: gera o par de chaves do atualizador e o manifesto assinado de um release.
//
//	assinar chaves <privada.pem> <publica.pem>
//	assinar manifesto --chave <privada.pem> --versao X.Y.Z --arquivo <exe> --url <url do exe> [--notas <arquivo>] --saida latest-wails.json
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"sync-game-save/internal/atualizacao"
)

func main() {
	if err := executar(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}
}

func executar(args []string) error {
	if len(args) == 0 {
		return errors.New("use `assinar chaves` ou `assinar manifesto`")
	}
	switch args[0] {
	case "chaves":
		if len(args) != 3 {
			return errors.New("uso: assinar chaves <privada.pem> <publica.pem>")
		}
		if _, err := os.Stat(args[1]); err == nil {
			return fmt.Errorf("%s já existe; não vou sobrescrever uma chave", args[1])
		}
		priv, pub, err := atualizacao.GerarChaves()
		if err != nil {
			return err
		}
		if err := os.WriteFile(args[1], priv, 0o600); err != nil {
			return err
		}
		return os.WriteFile(args[2], pub, 0o644)
	case "manifesto":
		fs := flag.NewFlagSet("manifesto", flag.ContinueOnError)
		chave := fs.String("chave", "", "chave privada em PEM")
		versao := fs.String("versao", "", "versão do release")
		arquivo := fs.String("arquivo", "", "executável a publicar")
		url := fs.String("url", "", "URL pública do executável")
		notas := fs.String("notas", "", "arquivo com as notas da versão")
		saida := fs.String("saida", "latest-wails.json", "arquivo de saída")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		bruto, err := os.ReadFile(*chave)
		if err != nil {
			return err
		}
		priv, err := atualizacao.LerChavePrivada(bruto)
		if err != nil {
			return err
		}
		texto := ""
		if *notas != "" {
			b, err := os.ReadFile(*notas)
			if err != nil {
				return err
			}
			texto = string(b)
		}
		out, err := atualizacao.Gerar(atualizacao.Entrada{Versao: *versao, Notas: texto, Arquivo: *arquivo, URL: *url}, priv)
		if err != nil {
			return err
		}
		return os.WriteFile(*saida, out, 0o644)
	}
	return fmt.Errorf("comando desconhecido: %s", args[0])
}
