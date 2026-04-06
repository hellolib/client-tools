package wecom

import (
	"testing"
)

func TestParseWecomConfig(t *testing.T) {
	cfg, err := ParseCliConfig()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("cfg: %+v", cfg)
}
