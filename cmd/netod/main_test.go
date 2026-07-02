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

func TestValidateGeneratedSingBoxRejectsLegacyRuleSets(t *testing.T) {
	for _, raw := range []string{
		`{"route":{"rule_set":[{"tag":"main-user-domains"}]}}`,
		`{"route":{"rules":[{"rule-set":"main-user-domains"}]}}`,
		`{"route":{"rule_set":[{"path":"/tmp/sing-box/rulesets/main-user-domains-ruleset.json"}]}}`,
	} {
		if err := validateGeneratedSingBox([]byte(raw)); err == nil {
			t.Fatalf("expected legacy rule-set config to be rejected: %s", raw)
		}
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

func TestCommandProvidersUpdateScript(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.txt")
	envPath := filepath.Join(dir, "env.txt")
	scriptPath := filepath.Join(dir, "provider-script")
	script := `#!/bin/sh
printf "%s\n" "$NETO_PROVIDER_NAME" > "` + envPath + `"
printf "1.1.1.1\n8.8.8.0/24\n2001:db8::/32\n"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(dir, "neto")
	if err := os.WriteFile(cfgPath, []byte(`
config provider 'json_ips'
	option type 'ip'
	option source 'script'
	option script_path '`+scriptPath+`'
	option local_path '`+cachePath+`'
	option update_via 'direct'
`), 0644); err != nil {
		t.Fatal(err)
	}

	if err := commandProvidersUpdate(providerOptions{configPath: cfgPath, name: "json_ips"}); err != nil {
		t.Fatal(err)
	}
	cache, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(cache) != "1.1.1.1/32\n8.8.8.0/24\n" {
		t.Fatalf("unexpected provider cache:\n%s", cache)
	}
	env, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(env) != "json_ips\n" {
		t.Fatalf("script did not receive provider env:\n%s", env)
	}
	raw, _ := os.ReadFile(cfgPath)
	if !strings.Contains(string(raw), `option item_count "2"`) ||
		!strings.Contains(string(raw), `option local_path "`+cachePath+`"`) ||
		!strings.Contains(string(raw), `option last_update "`) {
		t.Fatalf("provider metadata was not written:\n%s", raw)
	}
}

func TestCommandProvidersUpdateScriptPrefersOutputFile(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.txt")
	scriptPath := filepath.Join(dir, "provider-script")
	script := `#!/bin/sh
printf "not-an-ip\n"
printf "0.0.0.0/32\n" > "$NETO_PROVIDER_OUTPUT"
printf "9.9.9.9\n8.8.8.0/24\n" > "$NETO_PROVIDER_OUTPUT"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(dir, "neto")
	if err := os.WriteFile(cfgPath, []byte(`
config provider 'json_ips'
	option type 'ip'
	option source 'script'
	option script_path '`+scriptPath+`'
	option local_path '`+cachePath+`'
	option update_via 'direct'
`), 0644); err != nil {
		t.Fatal(err)
	}

	if err := commandProvidersUpdate(providerOptions{configPath: cfgPath, name: "json_ips"}); err != nil {
		t.Fatal(err)
	}
	cache, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(cache) != "8.8.8.0/24\n9.9.9.9/32\n" {
		t.Fatalf("unexpected provider cache:\n%s", cache)
	}
}

func TestCommandLogsSingBoxReadsTailAndClears(t *testing.T) {
	dir := t.TempDir()
	oldPath := singBoxLogPath
	singBoxLogPath = filepath.Join(dir, "sing-box.log")
	t.Cleanup(func() {
		singBoxLogPath = oldPath
	})

	line := strings.Repeat("x", 1024) + "\n"
	if err := os.WriteFile(singBoxLogPath, []byte(strings.Repeat(line, 160)), 0644); err != nil {
		t.Fatal(err)
	}

	tail, err := readFileTail(singBoxLogPath, 4<<10)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(tail), "[older log lines omitted]\n") {
		t.Fatalf("expected truncated log prefix, got %q", string(tail[:min(len(tail), 80)]))
	}
	if len(tail) > 4200 {
		t.Fatalf("log tail should be capped, got %d bytes", len(tail))
	}

	if err := os.WriteFile(singBoxLogPath, []byte("current log\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := captureStdout(t, func() error {
		return commandLogs([]string{"sing-box"})
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != "current log\n" {
		t.Fatalf("unexpected command log output: %q", out)
	}

	if _, err := captureStdout(t, func() error {
		return commandLogs([]string{"sing-box", "clear"})
	}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(singBoxLogPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Fatalf("log was not cleared: %q", string(data))
	}
}

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	runErr := fn()
	_ = w.Close()
	os.Stdout = oldStdout

	var out bytes.Buffer
	if _, err := out.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	return out.String(), runErr
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
