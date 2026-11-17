package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// api_name is derived from bucket and path, which force replacement, so it
// never changes in place. Without UseStateForUnknown every unrelated update
// plans it as "(known after apply)".
func TestAPINameKeepsStateAcrossUpdates(t *testing.T) {
	ctx := context.Background()
	checked := 0
	for _, newResource := range (&AidialProvider{}).Resources(ctx) {
		r := newResource()
		var schemaResp resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
		attribute, ok := schemaResp.Schema.Attributes["api_name"].(schema.StringAttribute)
		if !ok {
			continue
		}
		checked++
		var meta resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "aidial"}, &meta)

		// On an update the plan value is unknown and the state holds the name.
		existing := tftypes.NewValue(tftypes.String, "existing resource")
		req := planmodifier.StringRequest{
			State:       tfsdk.State{Raw: existing},
			Plan:        tfsdk.Plan{Raw: existing},
			StateValue:  types.StringValue("applications/x"),
			PlanValue:   types.StringUnknown(),
			ConfigValue: types.StringNull(),
		}
		resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
		for _, m := range attribute.PlanModifiers {
			m.PlanModifyString(ctx, req, resp)
		}
		if resp.PlanValue.IsUnknown() || resp.PlanValue.ValueString() != "applications/x" {
			t.Errorf("%s: api_name must keep its state value on update, got %v", meta.TypeName, resp.PlanValue)
		}
	}
	if checked < 7 {
		t.Fatalf("expected to check at least 7 resources with api_name, checked %d", checked)
	}
}
