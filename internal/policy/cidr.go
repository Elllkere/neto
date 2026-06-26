package policy

import (
	"bufio"
	"fmt"
	"math/bits"
	"net"
	"os"
	"sort"
	"strings"
)

type IPv4CIDR struct {
	Start uint32
	End   uint32
}

func LoadIPv4CIDRsFile(path string) ([]*net.IPNet, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cidrs []*net.IPNet
	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(stripProviderComment(scanner.Text()))
		if line == "" {
			continue
		}
		ipnet, err := ParseIPv4CIDR(line)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, lineNo, err)
		}
		cidrs = append(cidrs, ipnet)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return NormalizeIPv4CIDRs(cidrs), nil
}

func ParseIPv4CIDR(s string) (*net.IPNet, error) {
	if !strings.Contains(s, "/") {
		s += "/32"
	}
	ip, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		return nil, err
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return nil, fmt.Errorf("not an IPv4 CIDR: %q", s)
	}
	ones, bits := ipnet.Mask.Size()
	if bits != 32 {
		return nil, fmt.Errorf("not an IPv4 CIDR: %q", s)
	}
	ipnet.IP = ip4.Mask(ipnet.Mask)
	ipnet.Mask = net.CIDRMask(ones, 32)
	return ipnet, nil
}

func NormalizeIPv4CIDRs(input []*net.IPNet) []*net.IPNet {
	if len(input) == 0 {
		return nil
	}

	ranges := make([]IPv4CIDR, 0, len(input))
	for _, ipnet := range input {
		if ipnet == nil {
			continue
		}
		ip4 := ipnet.IP.To4()
		ones, total := ipnet.Mask.Size()
		if ip4 == nil || total != 32 || ones < 0 {
			continue
		}
		start := ipv4ToUint32(ip4.Mask(ipnet.Mask))
		size := uint64(1) << uint(32-ones)
		end := start + uint32(size-1)
		ranges = append(ranges, IPv4CIDR{Start: start, End: end})
	}
	if len(ranges) == 0 {
		return nil
	}

	sort.Slice(ranges, func(i, j int) bool {
		if ranges[i].Start == ranges[j].Start {
			return ranges[i].End < ranges[j].End
		}
		return ranges[i].Start < ranges[j].Start
	})

	merged := make([]IPv4CIDR, 0, len(ranges))
	for _, r := range ranges {
		if len(merged) == 0 {
			merged = append(merged, r)
			continue
		}
		last := &merged[len(merged)-1]
		if uint64(r.Start) <= uint64(last.End)+1 {
			if r.End > last.End {
				last.End = r.End
			}
			continue
		}
		merged = append(merged, r)
	}

	out := make([]*net.IPNet, 0, len(merged))
	for _, r := range merged {
		out = append(out, rangeToCIDRs(r.Start, r.End)...)
	}
	return out
}

func MustIPv4CIDRs(values ...string) []*net.IPNet {
	out := make([]*net.IPNet, 0, len(values))
	for _, v := range values {
		ipnet, err := ParseIPv4CIDR(v)
		if err != nil {
			panic(err)
		}
		out = append(out, ipnet)
	}
	return NormalizeIPv4CIDRs(out)
}

func CIDRStrings(cidrs []*net.IPNet) []string {
	out := make([]string, 0, len(cidrs))
	for _, c := range cidrs {
		out = append(out, c.String())
	}
	return out
}

func rangeToCIDRs(start, end uint32) []*net.IPNet {
	var out []*net.IPNet
	for start <= end {
		tz := 32
		if start != 0 {
			tz = bits.TrailingZeros32(start)
		}
		blockSize := uint64(1) << uint(tz)
		remaining := uint64(end) - uint64(start) + 1
		for blockSize > remaining {
			blockSize >>= 1
			tz--
		}
		prefix := 32 - (bits.Len64(blockSize) - 1)
		out = append(out, &net.IPNet{
			IP:   uint32ToIPv4(start),
			Mask: net.CIDRMask(prefix, 32),
		})
		if blockSize >= remaining {
			break
		}
		start += uint32(blockSize)
	}
	return out
}

func ipv4ToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

func uint32ToIPv4(v uint32) net.IP {
	return net.IPv4(byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

func stripProviderComment(s string) string {
	if i := strings.IndexByte(s, '#'); i >= 0 {
		return s[:i]
	}
	return s
}
