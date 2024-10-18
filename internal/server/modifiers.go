package server

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type resetValueWhenFamilyChanges struct{}

func (m resetValueWhenFamilyChanges) Description(_ context.Context) string {
	return "Resets value when ip_address_family changes; otherwise, keeps the prior state value."
}

func (m resetValueWhenFamilyChanges) MarkdownDescription(_ context.Context) string {
	return "Resets value when `ip_address_family` changes; otherwise, keeps the prior state value."
}

func (m resetValueWhenFamilyChanges) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.ConfigValue.IsNull() && !req.ConfigValue.IsUnknown() {
		resp.PlanValue = req.ConfigValue
		return
	}

	var priorFamily, configFamily types.String
	if diags := req.State.GetAttribute(ctx, req.Path.ParentPath().AtName("ip_address_family"), &priorFamily); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	if diags := req.Config.GetAttribute(ctx, req.Path.ParentPath().AtName("ip_address_family"), &configFamily); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if !priorFamily.Equal(configFamily) {
		resp.PlanValue = types.StringUnknown()
	} else {
		resp.PlanValue = req.StateValue
	}
}
