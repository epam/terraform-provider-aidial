package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type catalogSchemaResource struct {
	client *client.Client
}

type catalogSchemaResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Bucket   types.String `tfsdk:"bucket"`
	Path     types.String `tfsdk:"path"`
	Schema   types.String `tfsdk:"schema"`
	SchemaID types.String `tfsdk:"schema_id"`
}

// parseCatalogSchema decodes a catalog schema document and returns it with its
// $id. DIAL Core requires $id to be a non-blank string.
func parseCatalogSchema(raw string) (map[string]any, string, error) {
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var doc map[string]any
	if err := decoder.Decode(&doc); err != nil {
		return nil, "", fmt.Errorf("schema must be a JSON object: %w", err)
	}
	if doc == nil {
		return nil, "", fmt.Errorf("schema must be a JSON object")
	}
	id, _ := doc["$id"].(string)
	if strings.TrimSpace(id) == "" {
		return nil, "", fmt.Errorf("schema must contain a non-blank string $id")
	}
	return doc, id, nil
}

// catalogSchemaValidator checks, at plan time, that schema is a JSON object with a $id.
type catalogSchemaValidator struct{}

func (catalogSchemaValidator) Description(context.Context) string {
	return "value must be a JSON object with a non-blank string $id"
}

func (v catalogSchemaValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (catalogSchemaValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if _, _, err := parseCatalogSchema(req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid catalog schema", err.Error())
	}
}

// schemaIDFromSchema plans schema_id as the $id of the planned schema, so it is
// known at plan time and only changes when the schema's $id does.
type schemaIDFromSchema struct{}

func (schemaIDFromSchema) Description(context.Context) string {
	return "schema_id mirrors the $id of schema."
}

func (m schemaIDFromSchema) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (schemaIDFromSchema) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	var schemaValue types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("schema"), &schemaValue)...)
	if resp.Diagnostics.HasError() || schemaValue.IsNull() || schemaValue.IsUnknown() {
		return
	}
	if _, id, err := parseCatalogSchema(schemaValue.ValueString()); err == nil {
		resp.PlanValue = types.StringValue(id)
	}
}

func (r *catalogSchemaResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_catalog_schema"
}

func (r *catalogSchemaResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a DIAL Core catalog schema (`PUT /v1/catalog_schemas/{bucket}/{path}`): a JSON Schema document that models reference through `catalog_schema_id`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"bucket": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(defaultBucket),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Target DIAL bucket. Defaults to \"" + defaultBucket + "\" when not set.",
			},
			"path": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Catalog schema path inside the target bucket.",
			},
			"schema": schema.StringAttribute{
				Required:    true,
				Validators:  []validator.String{catalogSchemaValidator{}},
				Description: "The catalog schema as a JSON object string (use `jsonencode`). Must contain a non-blank `$id`.",
			},
			"schema_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					schemaIDFromSchema{},
				},
				Description: "The `$id` of the schema; use it as a model's `catalog_schema_id`.",
			},
		},
	}
}

func (r *catalogSchemaResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Invalid provider data", "Expected *client.Client")
		return
	}

	r.client = c
}

// setCatalogSchemaState stores the API document, keeping the configured text
// (prior) when it is semantically equal so formatting and key order survive.
func setCatalogSchemaState(state *catalogSchemaResourceModel, prior types.String, api client.CatalogSchema) error {
	decoder := json.NewDecoder(bytes.NewReader(api))
	decoder.UseNumber()
	var apiDoc map[string]any
	if err := decoder.Decode(&apiDoc); err != nil {
		return fmt.Errorf("failed to decode catalog schema: %w", err)
	}
	id, _ := apiDoc["$id"].(string)

	state.Schema = types.StringValue(string(api))
	if !prior.IsNull() && !prior.IsUnknown() {
		if priorDoc, _, err := parseCatalogSchema(prior.ValueString()); err == nil && jsonEqual(priorDoc, apiDoc) {
			state.Schema = prior
		}
	}
	state.SchemaID = types.StringValue(id)
	return nil
}

func (r *catalogSchemaResource) save(ctx context.Context, plan *catalogSchemaResourceModel, action string, diags *diag.Diagnostics) bool {
	bucket, p := plan.Bucket.ValueString(), plan.Path.ValueString()

	if err := r.client.SaveCatalogSchema(ctx, bucket, p, client.CatalogSchema(plan.Schema.ValueString())); err != nil {
		diags.AddError("Failed to "+action+" catalog schema", err.Error())
		return false
	}

	saved, err := r.client.GetCatalogSchema(ctx, bucket, p)
	if err != nil {
		diags.AddError("Failed to read catalog schema after "+action, err.Error())
		return false
	}
	if saved == nil {
		diags.AddError(
			"Catalog schema not found after "+action,
			"The API returned success but the catalog schema could not be read.",
		)
		return false
	}

	plan.ID = types.StringValue(bucket + "/" + p)
	if err := setCatalogSchemaState(plan, plan.Schema, saved); err != nil {
		diags.AddError("Failed to store catalog schema state", err.Error())
		return false
	}
	return true
}

func (r *catalogSchemaResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan catalogSchemaResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.save(ctx, &plan, "create", &resp.Diagnostics) {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}
}

func (r *catalogSchemaResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state catalogSchemaResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	doc, err := r.client.GetCatalogSchema(ctx, state.Bucket.ValueString(), state.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read catalog schema", err.Error())
		return
	}
	if doc == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	if err := setCatalogSchemaState(&state, state.Schema, doc); err != nil {
		resp.Diagnostics.AddError("Failed to store catalog schema state", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *catalogSchemaResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan catalogSchemaResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.save(ctx, &plan, "update", &resp.Diagnostics) {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}
}

func (r *catalogSchemaResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state catalogSchemaResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteCatalogSchema(ctx, state.Bucket.ValueString(), state.Path.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete catalog schema", err.Error())
	}
}

func (r *catalogSchemaResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Use terraform import aidial_catalog_schema.example <bucket>/<path>.",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bucket"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), parts[1])...)
}
