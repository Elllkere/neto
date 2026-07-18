package proxyroute

import (
	"fmt"
	"strings"

	"github.com/elllkere/neto/internal/config"
)

// Target describes the stable nftables -> TProxy inbound mapping for one
// custom outbound. The order follows UCI outbound section order.
type Target struct {
	Tag     string
	Chain   string
	Inbound string
	Port    int
}

func Targets(cfg config.Config) []Target {
	outbounds := cfg.EnabledCustomOutbounds()
	targets := make([]Target, 0, len(outbounds))
	for i, outbound := range outbounds {
		targets = append(targets, Target{
			Tag:     outbound.Tag,
			Chain:   fmt.Sprintf("to_proxy_%04d", i),
			Inbound: fmt.Sprintf("tproxy-%04d-in", i),
			Port:    cfg.Main.TProxyPort + i,
		})
	}
	return targets
}

func Find(targets []Target, tag string) (Target, bool) {
	tag = strings.TrimSpace(tag)
	for _, target := range targets {
		if target.Tag == tag {
			return target, true
		}
	}
	return Target{}, false
}
