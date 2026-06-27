package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/elllkere/neto/internal/config"
	"github.com/elllkere/neto/internal/dnsproxy"
	"github.com/elllkere/neto/internal/importer"
	"github.com/elllkere/neto/internal/nft"
	"github.com/elllkere/neto/internal/provider"
	"github.com/elllkere/neto/internal/singbox"
	"github.com/elllkere/neto/internal/status"
	"github.com/elllkere/neto/internal/tproxy"
)

const (
	defaultOutDir = "/tmp/neto"
	nftFileName   = "neto.nft"
	sbFileName    = "sing-box.json"
)

var version = "dev"

type options struct {
	configPath        string
	outDir            string
	skipRuntimeChecks bool
	skipSingBoxCheck  bool
}

type importOptions struct {
	configPath string
	filePath   string
}

type subscriptionOptions struct {
	configPath string
	name       string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "netod:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return errors.New("missing command")
	}

	switch args[0] {
	case "version":
		fmt.Printf("netod %s\n", version)
		return nil
	case "check":
		opts, err := parseOptions(args[0], args[1:], true)
		if err != nil {
			return err
		}
		return commandCheck(opts)
	case "compile":
		opts, err := parseOptions(args[0], args[1:], false)
		if err != nil {
			return err
		}
		_, _, _, err = compile(opts)
		return err
	case "apply":
		opts, err := parseOptions(args[0], args[1:], false)
		if err != nil {
			return err
		}
		return commandApply(opts)
	case "status":
		opts, err := parseOptions(args[0], args[1:], false)
		if err != nil {
			return err
		}
		return commandStatus(opts)
	case "debug":
		opts, err := parseOptions(args[0], args[1:], false)
		if err != nil {
			return err
		}
		return commandDebug(opts)
	case "run":
		opts, err := parseOptions(args[0], args[1:], false)
		if err != nil {
			return err
		}
		return commandRun(opts)
	case "import-uri":
		opts, err := parseImportOptions(args[1:])
		if err != nil {
			return err
		}
		return commandImportURI(opts)
	case "subscriptions":
		opts, err := parseSubscriptionOptions(args[1:])
		if err != nil {
			return err
		}
		return commandSubscriptionsUpdate(opts)
	default:
		usage()
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func parseOptions(command string, args []string, checkFlags bool) (options, error) {
	fs := flag.NewFlagSet("netod "+command, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	opts := options{
		configPath: config.DefaultPath,
		outDir:     defaultOutDir,
	}
	fs.StringVar(&opts.configPath, "config", opts.configPath, "path to UCI config")
	fs.StringVar(&opts.outDir, "out-dir", opts.outDir, "runtime output directory")
	if checkFlags {
		fs.BoolVar(&opts.skipRuntimeChecks, "skip-runtime-checks", false, "skip nft command validation")
		fs.BoolVar(&opts.skipSingBoxCheck, "skip-singbox-check", false, "skip sing-box binary validation")
	}
	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	if fs.NArg() != 0 {
		return options{}, fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	return opts, nil
}

func parseImportOptions(args []string) (importOptions, error) {
	fs := flag.NewFlagSet("netod import-uri", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	opts := importOptions{configPath: config.DefaultPath}
	fs.StringVar(&opts.configPath, "config", opts.configPath, "path to UCI config")
	fs.StringVar(&opts.filePath, "file", opts.filePath, "file containing share links")
	if err := fs.Parse(args); err != nil {
		return importOptions{}, err
	}
	if opts.filePath == "" {
		return importOptions{}, fmt.Errorf("import-uri requires -file")
	}
	if fs.NArg() != 0 {
		return importOptions{}, fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	return opts, nil
}

func parseSubscriptionOptions(args []string) (subscriptionOptions, error) {
	if len(args) == 0 || args[0] != "update" {
		return subscriptionOptions{}, fmt.Errorf("usage: netod subscriptions update [name] [options]")
	}
	fs := flag.NewFlagSet("netod subscriptions update", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	opts := subscriptionOptions{configPath: config.DefaultPath}
	fs.StringVar(&opts.configPath, "config", opts.configPath, "path to UCI config")
	if err := fs.Parse(args[1:]); err != nil {
		return subscriptionOptions{}, err
	}
	if fs.NArg() > 1 {
		return subscriptionOptions{}, fmt.Errorf("unexpected argument %q", fs.Arg(1))
	}
	if fs.NArg() == 1 {
		opts.name = fs.Arg(0)
	}
	return opts, nil
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: netod <version|check|compile|apply|status|debug|run|import-uri|subscriptions> [options]")
}

func commandCheck(opts options) error {
	cfg, err := config.LoadFile(opts.configPath)
	if err != nil {
		return err
	}
	printWarnings(cfg)
	if !cfg.Main.Enabled {
		fmt.Println("config ok: neto is disabled")
		return nil
	}

	_, _, sbPath, err := compile(opts)
	if err != nil {
		return err
	}
	if !opts.skipRuntimeChecks {
		if err := requireCommand("nft"); err != nil {
			return err
		}
		if err := command("nft", "-c", "-f", nftPath(opts)).Run(); err != nil {
			return fmt.Errorf("nft validation failed: %w", err)
		}
	}
	if !opts.skipSingBoxCheck {
		if !singbox.BinaryExists(cfg.Main.SingBoxBin) {
			return fmt.Errorf("sing-box binary is missing or not executable: %s", cfg.Main.SingBoxBin)
		}
		if err := singbox.CheckBinary(cfg.Main.SingBoxBin, sbPath); err != nil {
			return err
		}
	}
	fmt.Println("config ok")
	return nil
}

func commandApply(opts options) error {
	cfg, err := config.LoadFile(opts.configPath)
	if err != nil {
		return err
	}
	if !cfg.Main.Enabled {
		_ = deleteNftTable()
		_ = cleanupRouting(cfg.Main.Mark, cfg.Main.Table)
		fmt.Println("neto is disabled; nft table and routing removed")
		return nil
	}
	cfg, nftPath, _, err := compile(opts)
	if err != nil {
		return err
	}
	if err := requireCommand("nft"); err != nil {
		return err
	}
	if err := requireCommand("ip"); err != nil {
		return err
	}
	if out, err := command("nft", "-c", "-f", nftPath).CombinedOutput(); err != nil {
		return fmt.Errorf("nft validation failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	_ = deleteNftTable()
	if out, err := command("nft", "-f", nftPath).CombinedOutput(); err != nil {
		return fmt.Errorf("nft apply failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	if err := ensureRouting(cfg.Main.Mark, cfg.Main.Table); err != nil {
		_ = deleteNftTable()
		_ = cleanupRouting(cfg.Main.Mark, cfg.Main.Table)
		return err
	}
	fmt.Println("nft and routing applied")
	return nil
}

func commandStatus(opts options) error {
	cfg, err := config.LoadFile(opts.configPath)
	if err != nil {
		return err
	}
	fmt.Println(status.Summary(cfg))
	return nil
}

func commandRun(opts options) error {
	cfg, err := config.LoadFile(opts.configPath)
	if err != nil {
		return err
	}
	if !cfg.Main.Enabled {
		fmt.Println("neto is disabled")
		return nil
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return dnsproxy.New(cfg).Run(ctx)
}

func commandImportURI(opts importOptions) error {
	data, err := os.ReadFile(opts.filePath)
	if err != nil {
		return err
	}
	nodes, err := importer.ParseLinks(string(data))
	if err != nil {
		return err
	}
	count, err := importer.ApplyNodes(nodes, importer.ApplyOptions{
		ConfigPath: opts.configPath,
		Source:     "manual",
	})
	if err != nil {
		return err
	}
	fmt.Printf("imported nodes: %d\n", count)
	return nil
}

func commandSubscriptionsUpdate(opts subscriptionOptions) error {
	cfg, err := config.LoadFile(opts.configPath)
	if err != nil {
		return err
	}
	var matched int
	var updated int
	for _, sub := range cfg.Subscriptions {
		if opts.name != "" && sub.Name != opts.name {
			continue
		}
		matched++
		if !sub.Enabled {
			fmt.Printf("subscription %s is disabled; skipped\n", sub.Name)
			continue
		}
		body, err := fetchSubscription(cfg, sub)
		if err != nil {
			return fmt.Errorf("subscription %s: %w", sub.Name, err)
		}
		nodes, err := importer.ParseLinks(string(body))
		if err != nil {
			return fmt.Errorf("subscription %s: %w", sub.Name, err)
		}
		count, err := importer.ApplyNodes(nodes, importer.ApplyOptions{
			ConfigPath:   opts.configPath,
			Source:       sub.Name,
			Subscription: true,
			Replace:      true,
		})
		if err != nil {
			return fmt.Errorf("subscription %s: %w", sub.Name, err)
		}
		fmt.Printf("subscription %s: imported nodes: %d\n", sub.Name, count)
		updated += count
	}
	if matched == 0 {
		if opts.name != "" {
			return fmt.Errorf("subscription %q not found", opts.name)
		}
		return fmt.Errorf("no subscriptions configured")
	}
	fmt.Printf("subscriptions updated nodes: %d\n", updated)
	return nil
}

func commandDebug(opts options) error {
	cfg, err := config.LoadFile(opts.configPath)
	if err != nil {
		return err
	}
	fmt.Println("=== neto config ===")
	fmt.Printf("config: %s\n", opts.configPath)
	fmt.Printf("outbounds: %s\n", status.OutboundsSummary(cfg))
	printWarnings(cfg)
	fmt.Println("=== generated files ===")
	printPath("nft", nftPath(opts))
	printPath("sing-box", singBoxPath(opts))
	fmt.Println("=== lan scope ===")
	fmt.Printf("lan_subnets4: %s\n", debugList(cfg.Main.LANSubnets))
	fmt.Printf("lan_ifaces: %s\n", debugList(cfg.Main.LANIfaces))
	fmt.Println("=== netod status ===")
	fmt.Println(status.Summary(cfg))
	fmt.Println("=== nft table ===")
	printCommand("nft", "list", "table", "inet", "neto")
	fmt.Println("=== ip rules ===")
	printCommand("ip", "-4", "rule", "show")
	fmt.Println("=== ip route table ===")
	printCommand("ip", "-4", "route", "show", "table", strconv.Itoa(cfg.Main.Table))
	fmt.Println("=== processes ===")
	printCommand("sh", "-c", "ps w | grep -E 'netod|sing-box' | grep -v grep")
	fmt.Println("=== listeners ===")
	if _, err := exec.LookPath("ss"); err == nil {
		printCommand("sh", "-c", "ss -lnp | grep -E '5353|15353|16001' || true")
	} else {
		printCommand("sh", "-c", "netstat -lnp 2>/dev/null | grep -E '5353|15353|16001' || netstat -ln 2>/dev/null | grep -E '5353|15353|16001' || true")
	}
	return nil
}

func debugList(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ",")
}

func fetchSubscription(cfg config.Config, sub config.Subscription) ([]byte, error) {
	switch sub.UpdateVia {
	case "", "direct":
		return fetchURLWithCurl(sub.URL, "")
	case "proxy":
		return fetchURLViaProxy(cfg, sub)
	default:
		return nil, fmt.Errorf("unsupported update_via %q", sub.UpdateVia)
	}
}

func fetchURLViaProxy(cfg config.Config, sub config.Subscription) ([]byte, error) {
	if !singbox.BinaryExists(cfg.Main.SingBoxBin) {
		return nil, fmt.Errorf("sing-box binary is missing or not executable: %s", cfg.Main.SingBoxBin)
	}
	port, err := freeLocalPort()
	if err != nil {
		return nil, err
	}
	proxyJSON, err := singbox.GenerateProxyClient(cfg, sub.UpdateOutbound, port)
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "neto-subscription-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "sing-box.json")
	if err := os.WriteFile(path, append(proxyJSON, '\n'), 0600); err != nil {
		return nil, err
	}
	if err := singbox.CheckBinary(cfg.Main.SingBoxBin, path); err != nil {
		return nil, err
	}

	var stderr bytes.Buffer
	cmd := exec.Command(cfg.Main.SingBoxBin, "run", "-c", path)
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()
	if err := waitForPort(port, 5*time.Second); err != nil {
		return nil, fmt.Errorf("temporary sing-box did not start: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	return fetchURLWithCurl(sub.URL, fmt.Sprintf("http://127.0.0.1:%d", port))
}

func fetchURLWithCurl(rawURL string, proxy string) ([]byte, error) {
	if err := requireCommand("curl"); err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "neto-subscription-curl-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	const maxBody = 16 << 20
	outPath := filepath.Join(dir, "subscription.txt")
	args := []string{
		"-fsSL",
		"--connect-timeout", "15",
		"--max-time", "60",
		"--max-filesize", strconv.Itoa(maxBody),
		"--user-agent", "neto/1",
		"--output", outPath,
	}
	if proxy != "" {
		args = append(args, "--proxy", proxy)
	} else {
		args = append(args, "--noproxy", "*")
	}
	args = append(args, rawURL)

	out, err := command("curl", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("curl failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	st, err := os.Stat(outPath)
	if err != nil {
		return nil, err
	}
	if st.Size() > maxBody {
		return nil, fmt.Errorf("subscription response is too large")
	}
	return os.ReadFile(outPath)
}

func freeLocalPort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

func waitForPort(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", addr)
}

func compile(opts options) (config.Config, string, string, error) {
	cfg, err := config.LoadFile(opts.configPath)
	if err != nil {
		return config.Config{}, "", "", err
	}
	cidrs, err := provider.LoadRuleCIDRs(cfg)
	if err != nil {
		return config.Config{}, "", "", err
	}
	nftText, err := nft.Generate(nft.Input{Config: cfg, RuleCIDRs: cidrs})
	if err != nil {
		return config.Config{}, "", "", err
	}
	sbJSON, err := singbox.Generate(cfg)
	if err != nil {
		return config.Config{}, "", "", err
	}
	if err := os.MkdirAll(opts.outDir, 0755); err != nil {
		return config.Config{}, "", "", err
	}
	nftPath := nftPath(opts)
	sbPath := singBoxPath(opts)
	if err := os.WriteFile(nftPath, []byte(nftText), 0644); err != nil {
		return config.Config{}, "", "", err
	}
	if err := os.WriteFile(sbPath, append(sbJSON, '\n'), 0644); err != nil {
		return config.Config{}, "", "", err
	}
	return cfg, nftPath, sbPath, nil
}

func nftPath(opts options) string {
	return filepath.Join(opts.outDir, nftFileName)
}

func singBoxPath(opts options) string {
	return filepath.Join(opts.outDir, sbFileName)
}

func requireCommand(name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("required command %q not found", name)
	}
	return nil
}

func command(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.Env = os.Environ()
	return cmd
}

func ensureRouting(mark string, table int) error {
	cfg := tproxy.Config{Mark: mark, Table: table}
	rules, routes, err := readRoutingState(cfg)
	if err != nil {
		return err
	}
	for _, c := range tproxy.PlanEnsure(rules, routes, cfg) {
		if out, err := command(c.Name, c.Args...).CombinedOutput(); err != nil {
			if strings.Contains(string(out), "File exists") {
				continue
			}
			return fmt.Errorf("%s %s failed: %w: %s", c.Name, strings.Join(c.Args, " "), err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

func readRoutingState(cfg tproxy.Config) (string, string, error) {
	rules, err := command("ip", "-4", "rule", "show").Output()
	if err != nil {
		return "", "", fmt.Errorf("ip rule show failed: %w", err)
	}
	routes, err := command("ip", "-4", "route", "show", "table", fmt.Sprintf("%d", cfg.Table)).CombinedOutput()
	if err != nil {
		if tproxy.RouteTableMissing(string(routes), exitCode(err)) {
			return string(rules), "", nil
		}
		return "", "", fmt.Errorf("ip route show table %d failed: %w", cfg.Table, err)
	}
	return string(rules), string(routes), nil
}

func deleteNftTable() error {
	_ = requireCommand("nft")
	_, err := command("nft", "delete", "table", "inet", "neto").CombinedOutput()
	return err
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

func printPath(label string, path string) {
	if st, err := os.Stat(path); err == nil {
		fmt.Printf("%s: %s exists size=%d\n", label, path, st.Size())
		return
	}
	fmt.Printf("%s: %s missing\n", label, path)
}

func printCommand(name string, args ...string) {
	out, err := exec.Command(name, args...).CombinedOutput()
	if len(out) > 0 {
		fmt.Print(string(out))
	}
	if err != nil {
		fmt.Printf("%s %s: %v\n", name, strings.Join(args, " "), err)
	}
}

func printWarnings(cfg config.Config) {
	for _, warning := range cfg.Warnings {
		fmt.Fprintln(os.Stderr, "warning:", warning)
	}
}

func cleanupRouting(mark string, table int) error {
	if err := requireCommand("ip"); err != nil {
		return err
	}
	cfg := tproxy.Config{Mark: mark, Table: table}
	for {
		rules, routes, err := readRoutingState(cfg)
		if err != nil {
			return err
		}
		commands := tproxy.PlanCleanup(rules, routes, cfg)
		if len(commands) == 0 {
			return nil
		}
		for _, c := range commands {
			if out, err := command(c.Name, c.Args...).CombinedOutput(); err != nil {
				return fmt.Errorf("%s %s failed: %w: %s", c.Name, strings.Join(c.Args, " "), err, strings.TrimSpace(string(out)))
			}
		}
	}
}
