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

func TestBuiltinIncludesAgy(t *testing.T) {
	for _, adapter := range Builtin() {
		if adapter.ID() == "agy" {
			return
		}
	}
	t.Fatal("Builtin() does not include agy")
}
