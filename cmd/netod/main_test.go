package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/elllkere/neto/internal/config"
)

func TestVersionCommand(t *testing.T) {
	var out bytes.Buffer
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	err = run([]string{"version"})
	_ = w.Close()
	os.Stdout = oldStdout
	if err != nil {
		t.Fatal(err)
	}
	if _, err := out.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "netod ") {
		t.Fatalf("unexpected version output: %q", out.String())
	}
}

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

func TestCommandImportURI(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "neto")
	importPath := filepath.Join(dir, "links.txt")
	if err := os.WriteFile(cfgPath, []byte(`
config main 'main'
	option enabled '1'
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(importPath, []byte("trojan://secret@example.com:443#Trojan"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := commandImportURI(importOptions{configPath: cfgPath, filePath: importPath}); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Outbounds) != 1 || cfg.Outbounds[0].Type != "trojan" || cfg.Outbounds[0].Server != "example.com" {
		t.Fatalf("unexpected outbounds: %+v", cfg.Outbounds)
	}
}

func TestCommandSubscriptionsUpdateDirect(t *testing.T) {
	dir := t.TempDir()
	subPath := filepath.Join(dir, "sub.txt")
	if err := os.WriteFile(subPath, []byte("vless://a3482e88-686a-4a58-8126-99c9df64b060@example.com:443#VLESS"), 0644); err != nil {
		t.Fatal(err)
	}
	argsPath := installFakeCurl(t, subPath)
	cfgPath := filepath.Join(dir, "neto")
	if err := os.WriteFile(cfgPath, []byte(`
config subscription 'sub1'
	option url 'https://example.com/sub'
	option update_via 'direct'
`), 0644); err != nil {
		t.Fatal(err)
	}

	if err := commandSubscriptionsUpdate(subscriptionOptions{configPath: cfgPath, name: "sub1"}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(cfgPath)
	if !strings.Contains(string(raw), "option subscription \"sub1\"") || !strings.Contains(string(raw), "example.com") {
		t.Fatalf("subscription node was not written:\n%s", raw)
	}
	cfg, err := config.LoadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Outbounds) != 1 || cfg.Outbounds[0].Type != "vless" {
		t.Fatalf("unexpected outbounds: %+v", cfg.Outbounds)
	}
	args, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(args), "--noproxy\n*\n") || !strings.Contains(string(args), "https://example.com/sub") {
		t.Fatalf("curl was not called for direct download as expected:\n%s", args)
	}
}

func TestCommandProvidersUpdateDirect(t *testing.T) {
	dir := t.TempDir()
	listPath := filepath.Join(dir, "domains.txt")
	if err := os.WriteFile(listPath, []byte("Example.COM.\nexample.org\n"), 0644); err != nil {
		t.Fatal(err)
	}
	argsPath := installFakeCurl(t, listPath)
	cachePath := filepath.Join(dir, "cache.txt")
	cfgPath := filepath.Join(dir, "neto")
	if err := os.WriteFile(cfgPath, []byte(`
config provider 'domains'
	option type 'domain'
	option url 'https://example.com/domains.txt'
	option local_path '`+cachePath+`'
	option update_via 'direct'
`), 0644); err != nil {
		t.Fatal(err)
	}

	if err := commandProvidersUpdate(providerOptions{configPath: cfgPath, name: "domains"}); err != nil {
		t.Fatal(err)
	}
	cache, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(cache) != "example.com\nexample.org\n" {
		t.Fatalf("unexpected provider cache:\n%s", cache)
	}
	raw, _ := os.ReadFile(cfgPath)
	if !strings.Contains(string(raw), `option item_count "2"`) || !strings.Contains(string(raw), `option local_path "`+cachePath+`"`) {
		t.Fatalf("provider metadata was not written:\n%s", raw)
	}
	args, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(args), "--noproxy\n*\n") || !strings.Contains(string(args), "https://example.com/domains.txt") {
		t.Fatalf("curl was not called for provider download as expected:\n%s", args)
	}
}

func installFakeCurl(t *testing.T, responsePath string) string {
	t.Helper()

	dir := t.TempDir()
	argsPath := filepath.Join(dir, "curl.args")
	curlPath := filepath.Join(dir, "curl")
	script := `#!/bin/sh
out=""
prev=""
for arg in "$@"; do
	printf "%s\n" "$arg" >> "$NETO_CURL_ARGS"
	if [ "$prev" = "output" ]; then
		out="$arg"
		prev=""
		continue
	fi
	if [ "$arg" = "--output" ]; then
		prev="output"
	fi
done
if [ -z "$out" ]; then
	exit 2
fi
cat "$NETO_CURL_RESPONSE" > "$out"
`
	if err := os.WriteFile(curlPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("NETO_CURL_RESPONSE", responsePath)
	t.Setenv("NETO_CURL_ARGS", argsPath)
	return argsPath
}
