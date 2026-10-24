package render

import "testing"

func TestShadersCompile(t *testing.T) {
	if _, err := NewPipeline(); err != nil {
		t.Fatal(err)
	}
}
