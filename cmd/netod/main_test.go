package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDisabledConfigDoesNotRequireOutbound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "neto")
	if err := os.WriteFile(path, []byte(`
config main 'main'
	option enabled '0'
`), 0644); err != nil {
		t.Fatal(err)
	}

	opts := options{
		configPath:        path,
		outDir:            t.TempDir(),
		skipRuntimeChecks: true,
		skipSingBoxCheck:  true,
	}
	if err := commandCheck(opts); err != nil {
		t.Fatalf("disabled check should not require outbound: %v", err)
	}
	if err := commandApply(opts); err != nil {
		t.Fatalf("disabled apply should not require outbound: %v", err)
	}
}
