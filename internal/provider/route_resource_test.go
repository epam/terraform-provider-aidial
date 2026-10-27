package provider

import (
	"context"
	"testing"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func newRoutePlan(t *testing.T) routeResourceModel {
	t.Helper()

	methods, diags := types.SetValueFrom(context.Background(), types.StringType, []string{"GET", "POST"})
	if diags.HasError() {
		t.Fatalf("failed to build methods set: %v", diags)
	}

	response, diags := types.ObjectValueFrom(context.Background(), routeResponseAttrTypes, routeResponseModel{
		Body:   types.StringValue("ok"),
		Status: types.Int64Value(200),
	})
	if diags.HasError() {
		t.Fatalf("failed to build response object: %v", diags)
	}

	upstreams, diags := types.ListValueFrom(context.Background(), types.ObjectType{AttrTypes: routeUpstreamAttrTypes}, []modelUpstreamModel{
		{
			Endpoint:          types.StringValue("https://example.invalid/v1"),
			ResponsesEndpoint: types.StringNull(),
			ID:                types.StringNull(),
			ExtraData:         types.MapNull(types.StringType),
			Key:               types.StringNull(),
			SecretExtraData:   types.MapNull(types.StringType),
			Tier:              types.Int64Value(0),
			Weight:            types.Int64Value(1),
		},
	})
	if diags.HasError() {
		t.Fatalf("failed to build upstreams list: %v", diags)
	}

	return routeResourceModel{
		Bucket:          types.StringValue("b"),
		Path:            types.StringValue("p"),
		Name:            types.StringValue("my-route"),
		Order:           types.Int64Null(),
		Methods:         methods,
		Paths:           types.SetNull(types.StringType),
		Permissions:     types.SetNull(types.StringType),
		UserRoles:       types.SetNull(types.StringType),
		Response:        response,
		AttachmentPaths: types.ObjectNull(routeAttachmentPathsAttrTypes),
		Upstreams:       upstreams,
	}
}

func TestRouteFromPlanRoundTrip(t *testing.T) {
	plan := newRoutePlan(t)

	route, err := routeFromPlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("routeFromPlan: %v", err)
	}
	if route.Name != "my-route" {
		t.Fatalf("expected name %q, got %q", "my-route", route.Name)
	}
	if len(route.Methods) != 2 || route.Methods[0] != "GET" || route.Methods[1] != "POST" {
		t.Fatalf("unexpected methods: %v", route.Methods)
	}
	if route.Response == nil || route.Response.Body == nil || *route.Response.Body != "ok" {
		t.Fatalf("expected response.body=ok, got %+v", route.Response)
	}
	if len(route.Upstreams) != 1 || route.Upstreams[0].Endpoint == nil || *route.Upstreams[0].Endpoint != "https://example.invalid/v1" {
		t.Fatalf("unexpected upstreams: %+v", route.Upstreams)
	}
}

// TestSetRouteStatePreservesConfiguredName guards against the same
// provider-consistency bug that aidial_role and aidial_model already hit:
// name is Required and must never be overwritten from the API response.
func TestSetRouteStatePreservesConfiguredName(t *testing.T) {
	state := routeResourceModel{Name: types.StringValue("my-route")}
	route := &client.Route{Name: "canonical-name"}

	if err := setRouteState(context.Background(), &state, route); err != nil {
		t.Fatalf("setRouteState: %v", err)
	}
	if state.Name.ValueString() != "my-route" {
		t.Fatalf("expected configured name to be preserved, got %q", state.Name.ValueString())
	}
	if state.APIName.ValueString() != "canonical-name" {
		t.Fatalf("expected api_name to reflect the API value, got %q", state.APIName.ValueString())
	}
}

func TestSetRouteStateFillsNameOnImport(t *testing.T) {
	state := routeResourceModel{Name: types.StringNull()}
	route := &client.Route{Name: "imported-route"}

	if err := setRouteState(context.Background(), &state, route); err != nil {
		t.Fatalf("setRouteState: %v", err)
	}
	if state.Name.ValueString() != "imported-route" {
		t.Fatalf("expected name to be filled from API on import, got %q", state.Name.ValueString())
	}
}

func TestSetRouteStateNullNestedFieldsWhenAPIOmitsThem(t *testing.T) {
	state := routeResourceModel{Name: types.StringValue("r")}
	route := &client.Route{Name: "r"}

	if err := setRouteState(context.Background(), &state, route); err != nil {
		t.Fatalf("setRouteState: %v", err)
	}
	if !state.Response.IsNull() {
		t.Fatalf("expected response to be null, got %v", state.Response)
	}
	if !state.AttachmentPaths.IsNull() {
		t.Fatalf("expected attachment_paths to be null, got %v", state.AttachmentPaths)
	}
	if !state.Upstreams.IsNull() {
		t.Fatalf("expected upstreams to be null, got %v", state.Upstreams)
	}
}

func TestSetRouteStateAttachmentPathsRoundTrip(t *testing.T) {
	state := routeResourceModel{Name: types.StringValue("r")}
	route := &client.Route{
		Name: "r",
		AttachmentPaths: &client.RouteAttachmentPath{
			RequestBody: []string{"a", "b"},
		},
	}

	if err := setRouteState(context.Background(), &state, route); err != nil {
		t.Fatalf("setRouteState: %v", err)
	}

	var ap routeAttachmentPathsModel
	if diags := state.AttachmentPaths.As(context.Background(), &ap, basetypes.ObjectAsOptions{}); diags.HasError() {
		t.Fatalf("As: %v", diags)
	}

	var requestBody []string
	if diags := ap.RequestBody.ElementsAs(context.Background(), &requestBody, false); diags.HasError() {
		t.Fatalf("ElementsAs: %v", diags)
	}
	if len(requestBody) != 2 || requestBody[0] != "a" || requestBody[1] != "b" {
		t.Fatalf("unexpected request_body: %v", requestBody)
	}
}

// TestRouteUpstreamExtraDataIsMap guards against a schema/Go-type mismatch:
// routeUpstreamAttrTypes (and modelUpstreamModel.ExtraData, shared with
// aidial_model) model upstreams[].extra_data as a map[string]string, but the
// route resource's declared schema.Attribute once typed it as a plain
// string, which would reject any non-empty extra_data config at plan time.
func TestRouteUpstreamExtraDataIsMap(t *testing.T) {
	ctx := context.Background()
	extraData, diags := types.MapValueFrom(ctx, types.StringType, map[string]string{"region": "us-east"})
	if diags.HasError() {
		t.Fatalf("failed to build extra_data map: %v", diags)
	}

	upstreamsList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: routeUpstreamAttrTypes}, []modelUpstreamModel{
		{
			Endpoint:          types.StringValue("https://example.invalid/v1"),
			ResponsesEndpoint: types.StringNull(),
			ID:                types.StringNull(),
			ExtraData:         extraData,
			Key:               types.StringNull(),
			SecretExtraData:   types.MapNull(types.StringType),
			Tier:              types.Int64Value(0),
			Weight:            types.Int64Value(1),
		},
	})
	if diags.HasError() {
		t.Fatalf("failed to build upstreams list: %v", diags)
	}

	plan := newRoutePlan(t)
	plan.Upstreams = upstreamsList

	route, err := routeFromPlan(ctx, plan)
	if err != nil {
		t.Fatalf("routeFromPlan: %v", err)
	}
	if len(route.Upstreams) != 1 || route.Upstreams[0].ExtraData["region"] != "us-east" {
		t.Fatalf("expected extra_data to round-trip as a map, got: %+v", route.Upstreams)
	}

	state := routeResourceModel{Name: types.StringValue("r")}
	if err := setRouteState(ctx, &state, route); err != nil {
		t.Fatalf("setRouteState: %v", err)
	}

	var upstreams []modelUpstreamModel
	if diags := state.Upstreams.ElementsAs(ctx, &upstreams, false); diags.HasError() {
		t.Fatalf("ElementsAs: %v", diags)
	}
	var gotExtraData map[string]string
	if diags := upstreams[0].ExtraData.ElementsAs(ctx, &gotExtraData, false); diags.HasError() {
		t.Fatalf("extra_data ElementsAs: %v", diags)
	}
	if gotExtraData["region"] != "us-east" {
		t.Fatalf("expected extra_data to round-trip through state as a map, got: %v", gotExtraData)
	}
}
