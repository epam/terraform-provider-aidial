package provider

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func newModelPlan(t *testing.T) modelResourceModel {
	t.Helper()

	features, diags := types.ObjectValueFrom(context.Background(), modelFeaturesAttrTypes, modelFeaturesModel{
		CacheSupported:   types.BoolValue(true),
		RateEndpoint:     types.StringValue("https://example.invalid/rate"),
		ReasoningEfforts: types.SetNull(types.StringType),
	})
	if diags.HasError() {
		t.Fatalf("failed to build features object: %v", diags)
	}

	limits, diags := types.ObjectValueFrom(context.Background(), modelLimitsAttrTypes, modelLimitsModel{
		MaxPromptTokens: types.Int64Value(1000),
	})
	if diags.HasError() {
		t.Fatalf("failed to build limits object: %v", diags)
	}

	pricing, diags := types.ObjectValueFrom(context.Background(), modelPricingAttrTypes, modelPricingModel{
		Unit: types.StringValue("token"),
	})
	if diags.HasError() {
		t.Fatalf("failed to build pricing object: %v", diags)
	}

	upstreams, diags := types.ListValueFrom(context.Background(), types.ObjectType{AttrTypes: modelUpstreamAttrTypes}, []modelUpstreamModel{
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

	return modelResourceModel{
		Bucket:      types.StringValue("b"),
		Path:        types.StringValue("p"),
		Name:        types.StringValue("my-model"),
		Type:        types.StringValue("CHAT"),
		DisplayName: types.StringNull(),
		Description: types.StringNull(),
		Endpoint:    types.StringNull(),
		UserRoles:   types.SetNull(types.StringType),
		Features:    features,
		Limits:      limits,
		Pricing:     pricing,
		Upstreams:   upstreams,
	}
}

func TestModelFromPlanNewFieldsRoundTrip(t *testing.T) {
	ctx := context.Background()
	plan := newModelPlan(t)

	plan.OverrideName = types.StringValue("anthropic.claude-v3-haiku")

	plan.Interfaces, _ = types.MapValueFrom(
		ctx,
		types.ObjectType{AttrTypes: map[string]attr.Type{
			"base_url": types.StringType,
		}},
		map[string]modelInterfaceModel{
			"anthropicMessages": {
				BaseURL: types.StringValue("http://bedrock-adapter.example"),
			},
		},
	)

	plan.DefaultHeaders, _ = types.MapValueFrom(
		ctx,
		types.StringType,
		map[string]string{},
	)

	plan.FieldsHashingOrder, _ = types.SetValueFrom(ctx, types.StringType, []string{"prefix.body.tools", "prefix.body.messages"})

	model, err := modelFromPlan(ctx, plan)
	if err != nil {
		t.Fatalf("modelFromPlan: %v", err)
	}

	if model.OverrideName == nil || *model.OverrideName != "anthropic.claude-v3-haiku" {
		t.Fatalf("unexpected override_name: %v", model.OverrideName)
	}

	if model.FieldsHashingOrder == nil || len(*model.FieldsHashingOrder) != 2 || (*model.FieldsHashingOrder)[0] != "prefix.body.tools" || (*model.FieldsHashingOrder)[1] != "prefix.body.messages" {
		t.Fatalf("unexpected fields_hashing_order: %v", model.FieldsHashingOrder)
	}

	if got := model.Interfaces["anthropicMessages"].BaseURL; got == nil ||
		*got != "http://bedrock-adapter.example" {
		t.Fatalf("unexpected interface base_url: %v", got)
	}

	if model.DefaultHeaders == nil {
		t.Fatal("expected default_headers to be an empty map, got nil")
	}
	if len(model.DefaultHeaders) != 0 {
		t.Fatalf("expected empty default_headers, got %v", model.DefaultHeaders)
	}
}

func TestSetModelStateNewFieldsRoundTrip(t *testing.T) {
	ctx := context.Background()
	state := modelResourceModel{Name: types.StringValue("m")}

	overrideName := "anthropic.claude-v3-haiku"
	baseURL := "http://bedrock-adapter.example"

	model := &client.Model{
		Name:         "m",
		Type:         "CHAT",
		OverrideName: &overrideName,
		Interfaces: map[string]client.ModelInterface{
			"anthropicMessages": {
				BaseURL: &baseURL,
			},
		},
		DefaultHeaders:     map[string]string{},
		FieldsHashingOrder: &[]string{"prefix.body.tools", "prefix.body.messages"},
	}

	if err := setModelState(ctx, &state, model); err != nil {
		t.Fatalf("setModelState: %v", err)
	}

	if got := state.OverrideName.ValueString(); got != overrideName {
		t.Fatalf("expected override_name %q, got %q", overrideName, got)
	}

	var fieldsHashingOrder []string
	if diags := state.FieldsHashingOrder.ElementsAs(ctx, &fieldsHashingOrder, false); diags.HasError() {
		t.Fatalf("fields_hashing_order ElementsAs: %v", diags)
	}
	if len(fieldsHashingOrder) != 2 || fieldsHashingOrder[0] != "prefix.body.tools" || fieldsHashingOrder[1] != "prefix.body.messages" {
		t.Fatalf("unexpected fields_hashing_order: %v", fieldsHashingOrder)
	}

	if state.Interfaces.IsNull() {
		t.Fatal("expected interfaces to be non-null")
	}

	var interfaces map[string]modelInterfaceModel
	if diags := state.Interfaces.ElementsAs(ctx, &interfaces, false); diags.HasError() {
		t.Fatalf("interfaces ElementsAs: %v", diags)
	}

	iface, ok := interfaces["anthropicMessages"]
	if !ok {
		t.Fatalf("expected anthropicMessages interface, got %v", interfaces)
	}
	if got := iface.BaseURL.ValueString(); got != baseURL {
		t.Fatalf("expected base_url %q, got %q", baseURL, got)
	}

	if state.DefaultHeaders.IsNull() {
		t.Fatal("expected default_headers to be an empty map, got null")
	}

	var headers map[string]string
	if diags := state.DefaultHeaders.ElementsAs(ctx, &headers, false); diags.HasError() {
		t.Fatalf("default_headers ElementsAs: %v", diags)
	}
	if len(headers) != 0 {
		t.Fatalf("expected empty default_headers, got %v", headers)
	}
}

func TestSetModelStateEmptyInterfacesAndHeadersAreNotNull(t *testing.T) {
	ctx := context.Background()
	state := modelResourceModel{Name: types.StringValue("m")}
	model := &client.Model{
		Name: "m",
		Type: "CHAT",
		// Interfaces и DefaultHeaders намеренно оставлены nil.
	}

	if err := setModelState(ctx, &state, model); err != nil {
		t.Fatalf("setModelState: %v", err)
	}

	if state.Interfaces.IsNull() {
		t.Fatal("expected interfaces to be an empty map, got null")
	}
	if state.DefaultHeaders.IsNull() {
		t.Fatal("expected default_headers to be an empty map, got null")
	}

	var interfaces map[string]modelInterfaceModel
	if diags := state.Interfaces.ElementsAs(ctx, &interfaces, false); diags.HasError() {
		t.Fatalf("interfaces ElementsAs: %v", diags)
	}
	if len(interfaces) != 0 {
		t.Fatalf("expected empty interfaces, got %v", interfaces)
	}

	var headers map[string]string
	if diags := state.DefaultHeaders.ElementsAs(ctx, &headers, false); diags.HasError() {
		t.Fatalf("default_headers ElementsAs: %v", diags)
	}
	if len(headers) != 0 {
		t.Fatalf("expected empty default_headers, got %v", headers)
	}
}

func TestModelFromPlanRoundTrip(t *testing.T) {
	plan := newModelPlan(t)

	model, err := modelFromPlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("modelFromPlan: %v", err)
	}
	if model.Name != "my-model" || model.Type != "CHAT" {
		t.Fatalf("unexpected name/type: %q/%q", model.Name, model.Type)
	}
	if model.Features == nil || model.Features.CacheSupported == nil || !*model.Features.CacheSupported {
		t.Fatalf("expected features.cache_supported=true, got %+v", model.Features)
	}
	if model.Limits == nil || model.Limits.MaxPromptTokens == nil || *model.Limits.MaxPromptTokens != 1000 {
		t.Fatalf("expected limits.max_prompt_tokens=1000, got %+v", model.Limits)
	}
	if model.Pricing == nil || model.Pricing.Unit != "token" {
		t.Fatalf("expected pricing.unit=token, got %+v", model.Pricing)
	}
	if len(model.Upstreams) != 1 || model.Upstreams[0].Endpoint == nil || *model.Upstreams[0].Endpoint != "https://example.invalid/v1" {
		t.Fatalf("unexpected upstreams: %+v", model.Upstreams)
	}
}

// TestSetModelStatePreservesConfiguredName mirrors the aidial_role
// regression: name is Required and must never be overwritten from the
// API response once known, only api_name should reflect it.
func TestSetModelStatePreservesConfiguredName(t *testing.T) {
	state := modelResourceModel{Name: types.StringValue("my-model")}
	model := &client.Model{Name: "canonical-name", Type: "CHAT"}

	if err := setModelState(context.Background(), &state, model); err != nil {
		t.Fatalf("setModelState: %v", err)
	}
	if state.Name.ValueString() != "my-model" {
		t.Fatalf("expected configured name to be preserved, got %q", state.Name.ValueString())
	}
	if state.APIName.ValueString() != "canonical-name" {
		t.Fatalf("expected api_name to reflect the API value, got %q", state.APIName.ValueString())
	}
}

func TestSetModelStateFillsNameOnImport(t *testing.T) {
	state := modelResourceModel{Name: types.StringNull()}
	model := &client.Model{Name: "imported-model", Type: "CHAT"}

	if err := setModelState(context.Background(), &state, model); err != nil {
		t.Fatalf("setModelState: %v", err)
	}
	if state.Name.ValueString() != "imported-model" {
		t.Fatalf("expected name to be filled from API on import, got %q", state.Name.ValueString())
	}
}

func TestSetModelStateNullNestedFieldsWhenAPIOmitsThem(t *testing.T) {
	state := modelResourceModel{Name: types.StringValue("m")}
	model := &client.Model{Name: "m", Type: "CHAT"}

	if err := setModelState(context.Background(), &state, model); err != nil {
		t.Fatalf("setModelState: %v", err)
	}
	if !state.Features.IsNull() {
		t.Fatalf("expected features to be null, got %v", state.Features)
	}
	if !state.Limits.IsNull() {
		t.Fatalf("expected limits to be null, got %v", state.Limits)
	}
	if !state.Pricing.IsNull() {
		t.Fatalf("expected pricing to be null, got %v", state.Pricing)
	}
	if !state.Upstreams.IsNull() {
		t.Fatalf("expected upstreams to be null, got %v", state.Upstreams)
	}
}

func TestSetModelStateStringSetsRoundTrip(t *testing.T) {
	ctx := context.Background()
	state := modelResourceModel{Name: types.StringValue("m")}
	model := &client.Model{
		Name:                 "m",
		Type:                 "CHAT",
		Dependencies:         &[]string{"dep-a", "dep-b"},
		DescriptionKeywords:  &[]string{"keyword-a", "keyword-b"},
		InputAttachmentTypes: &[]string{"image", "file"},
		Interceptors:         &[]string{"interceptor-a", "interceptor-b"},
	}

	if err := setModelState(ctx, &state, model); err != nil {
		t.Fatalf("setModelState: %v", err)
	}

	tests := []struct {
		name     string
		value    types.Set
		expected []string
	}{
		{
			name:     "dependencies",
			value:    state.Dependencies,
			expected: []string{"dep-a", "dep-b"},
		},
		{
			name:     "description_keywords",
			value:    state.DescriptionKeywords,
			expected: []string{"keyword-a", "keyword-b"},
		},
		{
			name:     "input_attachment_types",
			value:    state.InputAttachmentTypes,
			expected: []string{"image", "file"},
		},
		{
			name:     "interceptors",
			value:    state.Interceptors,
			expected: []string{"interceptor-a", "interceptor-b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value.IsNull() {
				t.Fatal("expected a non-null set")
			}

			var got []string
			if diags := tt.value.ElementsAs(ctx, &got, false); diags.HasError() {
				t.Fatalf("ElementsAs: %v", diags)
			}

			assertStringSetValues(t, got, tt.expected)
		})
	}
}

func assertStringSetValues(t *testing.T, got, expected []string) {
	t.Helper()

	if len(got) != len(expected) {
		t.Fatalf("expected values %v, got %v", expected, got)
	}

	gotValues := make(map[string]bool, len(got))
	for _, value := range got {
		gotValues[value] = true
	}

	for _, value := range expected {
		if !gotValues[value] {
			t.Errorf("expected value %q in set %v", value, got)
		}
	}
}

func TestSetModelStateUpstreamsRoundTrip(t *testing.T) {
	state := modelResourceModel{Name: types.StringValue("m")}
	secret := "upstream-key"
	model := &client.Model{
		Name: "m",
		Type: "CHAT",
		Upstreams: []client.ModelUpstream{
			{Endpoint: ptr("https://example.invalid/v1"), Key: &secret, Tier: func() *int { v := 0; return &v }(), Weight: func() *int { v := 1; return &v }()},
		},
	}

	if err := setModelState(context.Background(), &state, model); err != nil {
		t.Fatalf("setModelState: %v", err)
	}

	var upstreams []modelUpstreamModel
	if diags := state.Upstreams.ElementsAs(context.Background(), &upstreams, false); diags.HasError() {
		t.Fatalf("ElementsAs: %v", diags)
	}
	if len(upstreams) != 1 || upstreams[0].Endpoint.ValueString() != "https://example.invalid/v1" {
		t.Fatalf("unexpected upstreams: %+v", upstreams)
	}
	if upstreams[0].Key.ValueString() != "upstream-key" {
		t.Fatalf("expected upstream key to round-trip, got %q", upstreams[0].Key.ValueString())
	}
}

func TestFeaturesSkillsSupportedRoundTrip(t *testing.T) {
	ctx := context.Background()
	plan := newModelPlan(t)

	var features modelFeaturesModel
	if diags := plan.Features.As(ctx, &features, basetypes.ObjectAsOptions{}); diags.HasError() {
		t.Fatalf("features As: %v", diags)
	}
	features.SkillsSupported = types.BoolValue(true)
	updatedFeatures, diags := types.ObjectValueFrom(ctx, modelFeaturesAttrTypes, features)
	if diags.HasError() {
		t.Fatalf("failed to rebuild features object: %v", diags)
	}
	plan.Features = updatedFeatures

	model, err := modelFromPlan(ctx, plan)
	if err != nil {
		t.Fatalf("modelFromPlan: %v", err)
	}
	if model.Features == nil || model.Features.SkillsSupported == nil || !*model.Features.SkillsSupported {
		t.Fatalf("expected features.skills_supported=true, got %+v", model.Features)
	}

	state := modelResourceModel{Name: types.StringValue("m")}
	if err := setModelState(ctx, &state, model); err != nil {
		t.Fatalf("setModelState: %v", err)
	}

	var roundTripped modelFeaturesModel
	if diags := state.Features.As(ctx, &roundTripped, basetypes.ObjectAsOptions{}); diags.HasError() {
		t.Fatalf("features As: %v", diags)
	}
	if !roundTripped.SkillsSupported.ValueBool() {
		t.Fatal("expected skills_supported to round-trip as true")
	}
}

func TestModelSchemaBucketDefaultsToPlatform(t *testing.T) {
	var schemaResp resource.SchemaResponse
	(&modelResource{}).Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	attr, ok := schemaResp.Schema.Attributes["bucket"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("bucket attribute is not a StringAttribute")
	}
	if attr.Required || !attr.Optional || !attr.Computed {
		t.Fatalf("bucket must be optional+computed, got required=%v optional=%v computed=%v", attr.Required, attr.Optional, attr.Computed)
	}

	// Test the plan modifier directly
	req := planmodifier.StringRequest{ConfigValue: types.StringNull(), PlanValue: types.StringUnknown()}
	modResp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
	bucketDefaultsToPlatform{}.PlanModifyString(context.Background(), req, modResp)
	if modResp.PlanValue.ValueString() != "platform" {
		t.Fatalf("bucket default = %q, want %q", modResp.PlanValue.ValueString(), "platform")
	}
}

func TestSetModelStateKeepsExplicitEmptyUpstreams(t *testing.T) {
	empty, diags := types.ListValue(types.ObjectType{AttrTypes: modelUpstreamAttrTypes}, []attr.Value{})
	if diags.HasError() {
		t.Fatalf("failed to build empty upstreams list: %v", diags)
	}
	state := modelResourceModel{Name: types.StringValue("m"), Upstreams: empty}
	model := &client.Model{Name: "m", Type: "CHAT"} // API omits empty upstreams

	if err := setModelState(context.Background(), &state, model); err != nil {
		t.Fatalf("setModelState: %v", err)
	}
	if state.Upstreams.IsNull() || state.Upstreams.IsUnknown() {
		t.Fatalf("expected upstreams to stay an empty list, got %v", state.Upstreams)
	}
	if n := len(state.Upstreams.Elements()); n != 0 {
		t.Fatalf("expected 0 upstreams, got %d", n)
	}
}

func TestModelUpstreamSecretsAreSensitiveAndNotComputed(t *testing.T) {
	var schemaResp resource.SchemaResponse
	(&modelResource{}).Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	upstreams := schemaResp.Schema.Attributes["upstreams"].(schema.ListNestedAttribute)
	key := upstreams.NestedObject.Attributes["key"].(schema.StringAttribute)
	secret := upstreams.NestedObject.Attributes["secret_extra_data"].(schema.MapAttribute)
	if key.Computed || !key.Sensitive {
		t.Fatalf("upstream key must be sensitive and not computed, got computed=%v sensitive=%v", key.Computed, key.Sensitive)
	}
	if secret.Computed || !secret.Sensitive {
		t.Fatalf("upstream secret_extra_data must be sensitive and not computed, got computed=%v sensitive=%v", secret.Computed, secret.Sensitive)
	}
}

// DIAL Core never returns upstream key/secretExtraData (writeOnly), so the
// configured values must survive the post-apply read.
func TestSetModelStateKeepsWriteOnlyUpstreamFields(t *testing.T) {
	secret, _ := types.MapValueFrom(context.Background(), types.StringType, map[string]string{"token": "s"})
	upstreamType := types.ObjectType{AttrTypes: modelUpstreamAttrTypes}
	upstream := func(endpoint string, key types.String, secret types.Map) attr.Value {
		return types.ObjectValueMust(modelUpstreamAttrTypes, map[string]attr.Value{
			"endpoint": types.StringValue(endpoint), "responses_endpoint": types.StringNull(), "id": types.StringNull(),
			"extra_data": types.MapNull(types.StringType), "key": key, "secret_extra_data": secret,
			"tier": types.Int64Null(), "weight": types.Int64Null(),
		})
	}
	planned := types.ListValueMust(upstreamType, []attr.Value{
		upstream("https://a", types.StringValue("k-a"), secret),
		upstream("https://b", types.StringNull(), types.MapNull(types.StringType)),
	})
	state := modelResourceModel{Name: types.StringValue("m"), Upstreams: planned}
	model := &client.Model{Name: "m", Type: "CHAT", Upstreams: []client.ModelUpstream{{Endpoint: ptr("https://a")}, {Endpoint: ptr("https://b")}}}

	if err := setModelState(context.Background(), &state, model); err != nil {
		t.Fatalf("setModelState: %v", err)
	}
	got := state.Upstreams.Elements()
	first := got[0].(types.Object).Attributes()
	second := got[1].(types.Object).Attributes()
	if first["key"] != types.StringValue("k-a") || !first["secret_extra_data"].Equal(secret) {
		t.Fatalf("configured key/secret_extra_data lost: key=%v secret=%v", first["key"], first["secret_extra_data"])
	}
	if !second["key"].(types.String).IsNull() || !second["secret_extra_data"].(types.Map).IsNull() {
		t.Fatalf("unset key/secret_extra_data must stay null: key=%v secret=%v", second["key"], second["secret_extra_data"])
	}
}

func TestModelTokenizerModelRoundTrip(t *testing.T) {
	ctx := context.Background()
	plan := newModelPlan(t)
	plan.TokenizerModel = types.StringValue("gpt-4-0314")

	model, err := modelFromPlan(ctx, plan)
	if err != nil {
		t.Fatalf("modelFromPlan: %v", err)
	}
	body, _ := json.Marshal(model)
	if !strings.Contains(string(body), `"tokenizerModel":"gpt-4-0314"`) {
		t.Fatalf("tokenizer_model must be sent as tokenizerModel, got %s", body)
	}

	state := modelResourceModel{Name: types.StringValue("m")}
	if err := setModelState(ctx, &state, model); err != nil {
		t.Fatalf("setModelState: %v", err)
	}
	if state.TokenizerModel.ValueString() != "gpt-4-0314" {
		t.Fatalf("tokenizer_model must round-trip, got %v", state.TokenizerModel)
	}

	unset := modelResourceModel{Name: types.StringValue("m")}
	if err := setModelState(ctx, &unset, &client.Model{Name: "m", Type: "CHAT"}); err != nil {
		t.Fatalf("setModelState: %v", err)
	}
	if !unset.TokenizerModel.IsNull() {
		t.Fatalf("tokenizer_model must be null when the API omits it, got %v", unset.TokenizerModel)
	}
}

func TestModelMaxInputAttachmentsRoundTrip(t *testing.T) {
	ctx := context.Background()
	plan := newModelPlan(t)
	plan.MaxInputAttachments = types.Int64Value(30)

	model, err := modelFromPlan(ctx, plan)
	if err != nil {
		t.Fatalf("modelFromPlan: %v", err)
	}
	body, _ := json.Marshal(model)
	if !strings.Contains(string(body), `"maxInputAttachments":30`) {
		t.Fatalf("max_input_attachments must be sent as maxInputAttachments, got %s", body)
	}

	state := modelResourceModel{Name: types.StringValue("m")}
	if err := setModelState(ctx, &state, model); err != nil {
		t.Fatalf("setModelState: %v", err)
	}
	if state.MaxInputAttachments.ValueInt64() != 30 {
		t.Fatalf("max_input_attachments must round-trip, got %v", state.MaxInputAttachments)
	}

	unset := modelResourceModel{Name: types.StringValue("m")}
	if err := setModelState(ctx, &unset, &client.Model{Name: "m", Type: "CHAT"}); err != nil {
		t.Fatalf("setModelState: %v", err)
	}
	if !unset.MaxInputAttachments.IsNull() {
		t.Fatalf("max_input_attachments must be null when the API omits it, got %v", unset.MaxInputAttachments)
	}
}
