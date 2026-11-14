//go:build windows && sgs_e2e

package main

import (
	"path/filepath"
	"testing"
)

func TestOpcoesWindowsTesteExigemPortaValida(t *testing.T) {
	t.Setenv("SGS_DATA_DIR", t.TempDir())
	for _, port := range []string{"", "0", "80", "65536", "9222 --outro"} {
		t.Setenv("SGS_E2E_PORT", port)
		if got := opcoesWindowsTeste(); len(got.AdditionalBrowserArgs) != 0 || got.WebviewUserDataPath != "" {
			t.Fatalf("porta %q aceita: %+v", port, got)
		}
	}
	t.Setenv("SGS_E2E_PORT", "19227")
	got := opcoesWindowsTeste()
	if len(got.AdditionalBrowserArgs) != 1 || got.AdditionalBrowserArgs[0] != "--remote-debugging-port=19227" || got.WebviewUserDataPath != filepath.Join(pastaDados(), "webview-e2e") {
		t.Fatal(got)
	}
}
