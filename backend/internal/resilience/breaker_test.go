package resilience

import (
	"errors"
	"testing"

	"github.com/sony/gobreaker"
)

func TestCircuitBreakerOpensAfterFailures(t *testing.T) {
	name := "test_oauth_" + t.Name()
	for i := 0; i < 3; i++ {
		_, _ = Execute(name, func() (interface{}, error) {
			return nil, errors.New("upstream down")
		})
	}

	state := State(name)
	if state != gobreaker.StateOpen {
		t.Fatalf("expected open state, got %v", state)
	}

	_, err := Execute(name, func() (interface{}, error) {
		return "ok", nil
	})
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected circuit open error, got %v", err)
	}
}

func TestForceOpen(t *testing.T) {
	name := "test_force_" + t.Name()
	ForceOpen(name)
	if State(name) != gobreaker.StateOpen {
		t.Fatalf("expected forced open, got %v", State(name))
	}
}
