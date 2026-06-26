package provider

import (
	"net"

	"github.com/elllkere/neto/internal/config"
	"github.com/elllkere/neto/internal/policy"
)

func LoadSubnetRuleCIDRs(cfg config.Config) ([]*net.IPNet, error) {
	var all []*net.IPNet
	for _, rule := range cfg.SubnetRules {
		for _, path := range rule.Files {
			cidrs, err := policy.LoadIPv4CIDRsFile(path)
			if err != nil {
				return nil, err
			}
			all = append(all, cidrs...)
		}
	}
	return policy.NormalizeIPv4CIDRs(all), nil
}
