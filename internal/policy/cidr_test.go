package policy

import (
	"reflect"
	"testing"
)

func TestNormalizeIPv4CIDRsDedupAndCollapse(t *testing.T) {
	got := CIDRStrings(NormalizeIPv4CIDRs(MustIPv4CIDRs(
		"192.0.2.0/25",
		"192.0.2.128/25",
		"192.0.2.1/32",
		"198.51.100.10",
		"198.51.100.10/32",
	)))
	want := []string{"192.0.2.0/24", "198.51.100.10/32"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestNormalizeIPv4CIDRsOverlaps(t *testing.T) {
	got := CIDRStrings(NormalizeIPv4CIDRs(MustIPv4CIDRs(
		"10.0.0.0/9",
		"10.128.0.0/9",
		"10.10.0.0/16",
	)))
	want := []string{"10.0.0.0/8"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestNormalizeIPv4CIDRsUpperAddressBoundary(t *testing.T) {
	got := CIDRStrings(NormalizeIPv4CIDRs(MustIPv4CIDRs(
		"224.0.0.0/4",
		"240.0.0.0/4",
	)))
	want := []string{"224.0.0.0/3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
