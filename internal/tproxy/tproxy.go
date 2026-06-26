package tproxy

import (
	"strconv"
	"strings"
)

type Config struct {
	Mark  string
	Table int
}

type Command struct {
	Name string
	Args []string
}

type State struct {
	RulePresent  bool
	RoutePresent bool
}

func RuleAddCommand(cfg Config) Command {
	return Command{Name: "ip", Args: []string{"-4", "rule", "add", "fwmark", cfg.Mark, "table", tableString(cfg)}}
}

func RuleDelCommand(cfg Config) Command {
	return Command{Name: "ip", Args: []string{"-4", "rule", "del", "fwmark", cfg.Mark, "table", tableString(cfg)}}
}

func RouteAddCommand(cfg Config) Command {
	return Command{Name: "ip", Args: []string{"-4", "route", "add", "local", "default", "dev", "lo", "table", tableString(cfg)}}
}

func RouteDelCommand(cfg Config) Command {
	return Command{Name: "ip", Args: []string{"-4", "route", "del", "local", "default", "dev", "lo", "table", tableString(cfg)}}
}

func Inspect(ruleShow, routeShow string, cfg Config) State {
	return State{
		RulePresent:  RulePresent(ruleShow, cfg),
		RoutePresent: RoutePresent(routeShow),
	}
}

func PlanEnsure(ruleShow, routeShow string, cfg Config) []Command {
	state := Inspect(ruleShow, routeShow, cfg)
	var commands []Command
	if !state.RulePresent {
		commands = append(commands, RuleAddCommand(cfg))
	}
	if !state.RoutePresent {
		commands = append(commands, RouteAddCommand(cfg))
	}
	return commands
}

func PlanCleanup(ruleShow, routeShow string, cfg Config) []Command {
	state := Inspect(ruleShow, routeShow, cfg)
	var commands []Command
	if state.RulePresent {
		commands = append(commands, RuleDelCommand(cfg))
	}
	if state.RoutePresent {
		commands = append(commands, RouteDelCommand(cfg))
	}
	return commands
}

func RulePresent(ruleShow string, cfg Config) bool {
	marks := markForms(cfg.Mark)
	table := tableString(cfg)
	for _, raw := range strings.Split(ruleShow, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if i := strings.Index(line, ":"); i >= 0 {
			line = strings.TrimSpace(line[i+1:])
		}
		if !strings.Contains(line, "fwmark ") {
			continue
		}
		if !(strings.Contains(line, " lookup "+table) || strings.Contains(line, " table "+table)) {
			continue
		}
		for _, mark := range marks {
			if strings.Contains(line, "fwmark "+mark) {
				return true
			}
		}
	}
	return false
}

func RoutePresent(routeShow string) bool {
	for _, raw := range strings.Split(routeShow, "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "local default ") && strings.Contains(line, " dev lo") {
			return true
		}
	}
	return false
}

func RouteTableMissing(output string, exitCode int) bool {
	if exitCode == 2 {
		return true
	}
	output = strings.ToLower(output)
	return strings.Contains(output, "fib table does not exist") ||
		strings.Contains(output, "cannot find device") ||
		strings.Contains(output, "no such file or directory")
}

func tableString(cfg Config) string {
	return strconv.Itoa(cfg.Table)
}

func markForms(mark string) []string {
	mark = strings.TrimSpace(mark)
	if mark == "" {
		return nil
	}
	forms := []string{mark}
	if strings.HasPrefix(mark, "0x") || strings.HasPrefix(mark, "0X") {
		forms = append(forms, strings.TrimPrefix(strings.TrimPrefix(mark, "0x"), "0X"))
	}
	return forms
}
