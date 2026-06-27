package provider

import (
	"net"

	"github.com/elllkere/neto/internal/config"
	"github.com/elllkere/neto/internal/policy"
	"github.com/elllkere/neto/internal/ruleengine"
)

func LoadRuleCIDRs(cfg config.Config) (map[int][]*net.IPNet, error) {
	out := map[int][]*net.IPNet{}
	for i, rule := range cfg.Rules {
		if !ruleengine.HasIPMatch(rule) {
			continue
		}
		var all []*net.IPNet
		for _, value := range rule.IPCIDRs {
			cidr, err := policy.ParseIPv4CIDR(value)
			if err != nil {
				return nil, err
			}
			all = append(all, cidr)
		}
		for _, path := range rule.Files {
			cidrs, err := policy.LoadIPv4CIDRsFile(path)
			if err != nil {
				return nil, err
			}
			all = append(all, cidrs...)
		}
		out[i] = policy.NormalizeIPv4CIDRs(all)
	}
	return out, nil
}
