package main

import "testing"

func TestNumeroDoBuild(t *testing.T) {
	if got := formatarBuild("0c3fa96e2b1d4c5a", "2026-11-14T23:50:12Z"); got != "0c3fa96 (2026-11-14)" {
		t.Fatal(got)
	}
	if got := formatarBuild("", ""); got != "local" {
		t.Fatal(got)
	}
	if got := formatarBuild("0c3fa96e2b1d4c5a", "sem data"); got != "0c3fa96" {
		t.Fatal(got)
	}
	if s := (App{}).Sobre(); s.Nome != "Sync Game Save" || s.Versao != versaoApp || s.Build == "" {
		t.Fatal(s)
	}
}
