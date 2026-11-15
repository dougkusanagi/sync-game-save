package catalog

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const ManifestURL = "https://raw.githubusercontent.com/mtkennerly/ludusavi-manifest/master/data/manifest.yaml"

const maxManifest = 200 << 20

// Fetch baixa o manifesto de url só se ele mudou (ETag). Devolve true se baixou algo novo.
// O arquivo novo entra por troca atômica; em erro, o antigo continua valendo.
func Fetch(manifest, url string) (bool, error) {
	etagFile := strings.TrimSuffix(manifest, ".yaml") + ".etag"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(manifest); err == nil {
		if tag, err := os.ReadFile(etagFile); err == nil {
			req.Header.Set("If-None-Match", strings.TrimSpace(string(tag)))
		}
	}
	resp, err := (&http.Client{Timeout: 5 * time.Minute}).Do(req)
	if err != nil {
		return false, fmt.Errorf("falha ao baixar o manifesto: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusNotModified:
		return false, nil
	case http.StatusOK:
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxManifest+1))
		if err != nil {
			return false, fmt.Errorf("falha ao baixar o manifesto: %w", err)
		}
		if len(body) > maxManifest {
			return false, fmt.Errorf("falha ao baixar o manifesto: arquivo maior que o limite")
		}
		tmp := manifest + ".tmp"
		if err := os.WriteFile(tmp, body, 0644); err != nil {
			return false, err
		}
		if err := os.Rename(tmp, manifest); err != nil {
			_ = os.Remove(tmp)
			return false, err
		}
		if tag := resp.Header.Get("ETag"); tag != "" {
			if err := os.WriteFile(etagFile, []byte(tag), 0644); err != nil {
				return true, err
			}
		}
		return true, nil
	}
	return false, fmt.Errorf("falha ao baixar o manifesto: resposta HTTP %d", resp.StatusCode)
}
