package providers_test

import (
	"strings"
	"testing"

	"p2pollo/internal/scraper/providers"
)

func TestLeet_Search(t *testing.T) {
	leet := providers.NewLeet()
	results, err := leet.Search("inception 2010")
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("no results returned")
	}

	t.Logf("%d resultados:", len(results))
	for i, r := range results {
		mag := r.MagnetLink
		if len(mag) > 60 {
			mag = mag[:60] + "..."
		}
		t.Logf("  [%d] %s | %s | seeds=%d | %s", i+1, r.Name, r.Size, r.Seeds, mag)
		if !strings.HasPrefix(r.MagnetLink, "magnet:") {
			t.Errorf("resultado %d tiene magnet inválido: %s", i+1, r.MagnetLink)
		}
	}
}
