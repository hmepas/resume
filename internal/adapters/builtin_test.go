package adapters

import "testing"

func TestBuiltinIncludesPrimeAgent(t *testing.T) {
	for _, adapter := range Builtin() {
		if adapter.ID() == "prime-agent" {
			return
		}
	}
	t.Fatal("Builtin() does not include prime-agent")
}
