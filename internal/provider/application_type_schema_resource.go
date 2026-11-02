package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type applicationTypeSchemaResource struct {
	client *client.Client
}

type applicationTypeSchemaResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Bucket   types.String `tfsdk:"bucket"`
	Path     types.String `tfsdk:"path"`
	Schema   types.String `tfsdk:"schema"`
	SchemaID types.String `tfsdk:"schema_id"`
}

// applicationTypeSchemaValidator checks, at plan time, that schema is a JSON object with a $id.
type applicationTypeSchemaValidator struct{}

func (applicationTypeSchemaValidator) Description(context.Context) string {
	return "value must be a JSON object with a non-blank string $id"
}

func (v applicationTypeSchemaValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (applicationTypeSchemaValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if _, _, err := parseSchemaDocument(req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid application type schema", err.Error())
	}
}

func (r *applicationTypeSchemaResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_type_schema"
}

func (r *applicationTypeSchemaResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a DIAL Core custom application type schema (`PUT /v1/schemas/{bucket}/{path}`): a JSON Schema document, identified by its `$id`, that describes a custom application type.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"bucket": schema.StringAttribute{
				Required:   true,
				Validators: []validator.String{stringvalidator.OneOf(platformBucket)},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Target DIAL bucket. Must be \"platform\".",
			},
			"path": schema.StringAttribute{
				Required:   true,
				Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Application type schema path inside the target bucket.",
			},
			"schema": schema.StringAttribute{
				Required:    true,
				Validators:  []validator.String{applicationTypeSchemaValidator{}},
				Description: "The application type schema as a JSON object string (use `jsonencode`). Must contain a non-blank `$id`.",
			},
			"schema_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					schemaIDFromSchema{},
				},
				Description: "The `$id` of the schema.",
			},
		},
	}
}

func (r *applicationTypeSchemaResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// setApplicationTypeSchemaState stores the API document, keeping the
// configured text (prior) when it is semantically equal so formatting and key
// order survive.
func setApplicationTypeSchemaState(state *applicationTypeSchemaResourceModel, prior types.String, api client.ApplicationTypeSchema) error {
	decoder := json.NewDecoder(bytes.NewReader(api))
	decoder.UseNumber()
	var apiDoc map[string]any
	if err := decoder.Decode(&apiDoc); err != nil {
		return fmt.Errorf("failed to decode application type schema: %w", err)
	}
	id, _ := apiDoc["$id"].(string)

	state.Schema = types.StringValue(string(api))
	if !prior.IsNull() && !prior.IsUnknown() {
		if priorDoc, _, err := parseSchemaDocument(prior.ValueString()); err == nil && jsonEqual(priorDoc, apiDoc) {
			state.Schema = prior
		}
	}
	state.SchemaID = types.StringValue(id)
	return nil
}

func (r *applicationTypeSchemaResource) save(ctx context.Context, plan *applicationTypeSchemaResourceModel, action string, diags *diag.Diagnostics) bool {
	bucket, p := plan.Bucket.ValueString(), plan.Path.ValueString()

	if err := r.client.SaveApplicationTypeSchema(ctx, bucket, p, client.ApplicationTypeSchema(plan.Schema.ValueString())); err != nil {
		diags.AddError("Failed to "+action+" application type schema", err.Error())
		return false
	}

	saved, err := r.client.GetApplicationTypeSchema(ctx, bucket, p)
	if err != nil {
		diags.AddError("Failed to read application type schema after "+action, err.Error())
		return false
	}
	if saved == nil {
		diags.AddError(
			"Application type schema not found after "+action,
			"The API returned success but the application type schema could not be read.",
		)
		return false
	}

	plan.ID = types.StringValue(bucket + "/" + p)
	if err := setApplicationTypeSchemaState(plan, plan.Schema, saved); err != nil {
		diags.AddError("Failed to store application type schema state", err.Error())
		return false
	}
	return true
}

func (r *applicationTypeSchemaResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan applicationTypeSchemaResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.save(ctx, &plan, "create", &resp.Diagnostics) {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}
}

func (r *applicationTypeSchemaResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state applicationTypeSchemaResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	doc, err := r.client.GetApplicationTypeSchema(ctx, state.Bucket.ValueString(), state.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read application type schema", err.Error())
		return
	}
	if doc == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	if err := setApplicationTypeSchemaState(&state, state.Schema, doc); err != nil {
		resp.Diagnostics.AddError("Failed to store application type schema state", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *applicationTypeSchemaResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan applicationTypeSchemaResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.save(ctx, &plan, "update", &resp.Diagnostics) {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}
}

func (r *applicationTypeSchemaResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state applicationTypeSchemaResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteApplicationTypeSchema(ctx, state.Bucket.ValueString(), state.Path.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete application type schema", err.Error())
	}
}

func (r *applicationTypeSchemaResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Use terraform import aidial_application_type_schema.example <bucket>/<path>.",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bucket"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), parts[1])...)
}
