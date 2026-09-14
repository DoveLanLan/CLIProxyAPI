package synthesizer

import (
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
)

func TestCommandCodeConfigSynthesisAndReload(t *testing.T) {
	cfg, err := config.ParseConfigBytes([]byte(`commandcode-api-key:
  - api-key: " user_one "
    prefix: cc
    priority: 3
    disable-cooling: true
    models:
      - name: upstream-one
        alias: friendly
  - api-key: user_two
    models:
      - name: upstream-two
        alias: friendly
`))
	if err != nil {
		t.Fatal(err)
	}
	makeAuths := func() *SynthesisContext {
		return &SynthesisContext{Config: cfg, Now: time.Now(), IDGenerator: NewStableIDGenerator()}
	}
	first, err := NewConfigSynthesizer().Synthesize(makeAuths())
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 {
		t.Fatalf("auths=%d", len(first))
	}
	if first[0].Provider != "commandcode" || first[0].Prefix != "cc" || first[0].Attributes["priority"] != "3" || first[0].AuthKind() != "apikey" || first[0].AuthSourceKind() != "config" {
		t.Fatalf("incorrect synthesis %+v", first[0])
	}
	if first[0].ID == first[1].ID {
		t.Fatal("credentials collide")
	}
	cfg.CommandCodeKey[0].Models[0].Name = "upstream-changed"
	second, err := NewConfigSynthesizer().Synthesize(makeAuths())
	if err != nil {
		t.Fatal(err)
	}
	if first[0].ID != second[0].ID {
		t.Fatal("model reload changed credential identity")
	}
	if first[0].Attributes["models_hash"] == second[0].Attributes["models_hash"] {
		t.Fatal("model reload did not change hash")
	}
	cfg.CommandCodeKey = nil
	last, err := NewConfigSynthesizer().Synthesize(makeAuths())
	if err != nil || len(last) != 0 {
		t.Fatalf("removal failed %v %v", last, err)
	}
}
