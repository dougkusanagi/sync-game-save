//go:build !windows || !sgs_e2e

package main

import "github.com/wailsapp/wails/v3/pkg/application"

func opcoesWindowsTeste() application.WindowsOptions { return application.WindowsOptions{} }
