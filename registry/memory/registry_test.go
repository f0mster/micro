package memory

import (
	"testing"

	tests "github.com/f0mster/micro/internal/test"
)

func TestRegister(t *testing.T) {
	m1 := New()
	tests.Registry_Test(m1, m1, t)
}
