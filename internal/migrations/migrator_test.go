package migrations

import (
	"testing"
)

func TestApply_NilDB(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic or error when db is nil, got neither")
		}
	}()
	_ = Apply(nil)
}
