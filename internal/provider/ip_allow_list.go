package provider

import (
	"context"
	"net"
	"net/netip"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// normalizeIPAllowList expands bare IPs to /32 or /128, canonicalizes CIDRs,
// de-duplicates, and sorts so config and AhaSend API forms compare equal.
func normalizeIPAllowList(entries []string) []string {
	if len(entries) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(entries))
	out := make([]string, 0, len(entries))
	for _, raw := range entries {
		canon := normalizeIPAllowListEntry(raw)
		if canon == "" {
			continue
		}
		if _, ok := seen[canon]; ok {
			continue
		}
		seen[canon] = struct{}{}
		out = append(out, canon)
	}
	sort.Strings(out)
	return out
}

// normalizeIPAllowListEntry returns AhaSend-style canonical CIDR, or the trimmed
// input when it is not a parseable IP/CIDR (leave validation to the API).
func normalizeIPAllowListEntry(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if strings.Contains(s, "/") {
		if prefix, err := netip.ParsePrefix(s); err == nil {
			return prefix.Masked().String()
		}
		if _, ipNet, err := net.ParseCIDR(s); err == nil {
			return ipNet.String()
		}
		return s
	}
	if addr, err := netip.ParseAddr(s); err == nil {
		bits := 32
		if addr.Is6() {
			bits = 128
		}
		return netip.PrefixFrom(addr, bits).String()
	}
	if ip := net.ParseIP(s); ip != nil {
		if ip.To4() != nil {
			return ip.String() + "/32"
		}
		return ip.String() + "/128"
	}
	return s
}

// normalizeIPAllowListModifier canonicalizes known plan values so bare IPs match
// AhaSend's stored /32 and /128 forms and avoid perpetual diffs.
type normalizeIPAllowListModifier struct{}

func normalizeIPAllowListPlanModifier() planmodifier.List {
	return normalizeIPAllowListModifier{}
}

func (m normalizeIPAllowListModifier) Description(_ context.Context) string {
	return "Normalizes bare IPs to /32 or /128 CIDR form to match AhaSend storage."
}

func (m normalizeIPAllowListModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m normalizeIPAllowListModifier) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	var ips []string
	diags := req.PlanValue.ElementsAs(ctx, &ips, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	normalized := normalizeIPAllowList(ips)
	list, listDiags := types.ListValueFrom(ctx, types.StringType, normalized)
	resp.Diagnostics.Append(listDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.PlanValue = list
}
