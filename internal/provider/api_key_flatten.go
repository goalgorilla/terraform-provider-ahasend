package provider

import (
	"context"
	"time"

	"github.com/AhaSend/ahasend-go/models/responses"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// apiKeyFlattened holds Terraform attribute values shared by parent and sub-account API key resources.
type apiKeyFlattened struct {
	ID          types.String
	Label       types.String
	PublicKey   types.String
	CreatedAt   types.String
	UpdatedAt   types.String
	Scopes      types.Set
	IPAllowList types.List
}

// flattenAPIKeyCommon maps an API key response into shared Terraform attributes.
// A nil IPAllowList becomes an empty list (not null) so state stays consistent.
// Scopes are a set so API response order does not cause apply/read inconsistencies.
func flattenAPIKeyCommon(ctx context.Context, key *responses.APIKey) (apiKeyFlattened, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := apiKeyFlattened{
		ID:        types.StringValue(key.ID.String()),
		Label:     types.StringValue(key.Label),
		PublicKey: types.StringValue(key.PublicKey),
		CreatedAt: types.StringValue(key.CreatedAt.UTC().Format(time.RFC3339)),
		UpdatedAt: types.StringValue(key.UpdatedAt.UTC().Format(time.RFC3339)),
	}

	scopeStrings := make([]string, 0, len(key.Scopes))
	for _, s := range key.Scopes {
		scopeStrings = append(scopeStrings, s.Scope)
	}
	scopes, scopeDiags := types.SetValueFrom(ctx, types.StringType, scopeStrings)
	diags.Append(scopeDiags...)
	out.Scopes = scopes

	ips := key.IPAllowList
	if ips == nil {
		ips = []string{}
	}
	ipList, ipDiags := types.ListValueFrom(ctx, types.StringType, normalizeIPAllowList(ips))
	diags.Append(ipDiags...)
	out.IPAllowList = ipList

	return out, diags
}
