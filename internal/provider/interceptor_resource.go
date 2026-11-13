package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &interceptorResource{}
	_ resource.ResourceWithImportState = &interceptorResource{}
)

type interceptorResource struct {
	client *client.Client
}

// interceptorModel is the state of aidial_interceptor and of the
// aidial_interceptor data source.
type interceptorModel struct {
	ID      types.String `tfsdk:"id"`
	Bucket  types.String `tfsdk:"bucket"`
	Path    types.String `tfsdk:"path"`
	Name    types.String `tfsdk:"name"`
	APIName types.String `tfsdk:"api_name"`

	interceptorBody
}

// interceptorBody holds every attribute of an interceptor except its identity.
// It is kept separate so that all three entry points (resource, data source,
// file data source) share one mapping.
type interceptorBody struct {
	Endpoint         types.String `tfsdk:"endpoint"`
	Description      types.String `tfsdk:"description"`
	DisplayName      types.String `tfsdk:"display_name"`
	DisplayVersion   types.String `tfsdk:"display_version"`
	IconURL          types.String `tfsdk:"icon_url"`
	CatalogSchemaID  types.String `tfsdk:"catalog_schema_id"`
	ForwardAuthToken types.Bool   `tfsdk:"forward_auth_token"`
	MaxRetryAttempts types.Int64  `tfsdk:"max_retry_attempts"`

	UserRoles      types.Set    `tfsdk:"user_roles"`
	DefaultHeaders types.Map    `tfsdk:"default_headers"`
	Features       types.Object `tfsdk:"features"`
	Interfaces     types.Map    `tfsdk:"interfaces"`

	Defaults          types.Dynamic `tfsdk:"defaults"`
	CatalogProperties types.Dynamic `tfsdk:"catalog_properties"`
}

func (r *interceptorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_interceptor"
}

func (r *interceptorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	str := func(description string) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Description: description}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an interceptor in DIAL Core (`PUT /v1/interceptors/{bucket}/{path}`). Models reference interceptors by name in their `interceptors` list.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "Resource identifier in the form <bucket>/<path>.",
			},
			"bucket": schema.StringAttribute{
				Required:      true,
				Validators:    []validator.String{stringvalidator.OneOf(platformBucket)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Target DIAL bucket. Must be \"platform\".",
			},
			"path": schema.StringAttribute{
				Required:      true,
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Interceptor path inside the bucket. DIAL Core accepts letters, digits and `._%:@[]()-`, so it cannot contain `/`.",
			},
			"name": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "Interceptor name sent in the DIAL API request body. DIAL Core derives the name from the bucket and path, so it can be omitted; then it is read back from the API.",
			},
			"api_name": schema.StringAttribute{
				Computed:    true,
				Description: "Canonical interceptor name returned by the DIAL API.",
			},
			"endpoint":          schema.StringAttribute{Required: true, Description: "URL DIAL Core calls to run the interceptor."},
			"description":       str("Interceptor description."),
			"display_name":      str("Human-readable interceptor name."),
			"display_version":   str("Version shown for the interceptor."),
			"icon_url":          str("URL of the interceptor icon."),
			"catalog_schema_id": str("URI (`$id`) of the catalog schema describing `catalog_properties`, for example the `schema_id` of an `aidial_catalog_schema`."),
			// DIAL Core fills these two in when they are not sent, so they are
			// computed too. UseStateForUnknown keeps them stable across updates.
			"forward_auth_token": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
				Description:   "Forward the caller's authorization token to the interceptor. DIAL Core defaults to false.",
			},
			"max_retry_attempts": schema.Int64Attribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Description:   "Maximum number of retry attempts. DIAL Core defaults to 1.",
			},
			"user_roles": schema.SetAttribute{Optional: true, ElementType: types.StringType, Description: "Roles allowed to use the interceptor."},
			"default_headers": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Additional headers sent with every request to the interceptor's endpoint.",
			},
			"features":   featuresResourceAttribute(),
			"interfaces": interfacesResourceAttribute(),
			"defaults": schema.DynamicAttribute{
				Optional:    true,
				Description: "Default request parameters (free-form object, sent to DIAL as-is).",
			},
			"catalog_properties": schema.DynamicAttribute{
				Optional:    true,
				Description: "Catalog metadata of the interceptor (free-form object, sent as `catalogProperties`).",
			},
		},
	}
}

func (r *interceptorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// interceptorFromPlan converts planned values to the API shape. Null and
// unknown attributes are not sent; a configured empty set is sent as [].
func interceptorFromPlan(ctx context.Context, plan interceptorModel) (*client.Interceptor, error) {
	ic := &client.Interceptor{
		Name:             plan.Name.ValueString(),
		Endpoint:         stringPointer(plan.Endpoint),
		Description:      stringPointer(plan.Description),
		DisplayName:      stringPointer(plan.DisplayName),
		DisplayVersion:   stringPointer(plan.DisplayVersion),
		IconURL:          stringPointer(plan.IconURL),
		CatalogSchemaID:  stringPointer(plan.CatalogSchemaID),
		ForwardAuthToken: boolPointer(plan.ForwardAuthToken),
		MaxRetryAttempts: int64Pointer(plan.MaxRetryAttempts),
	}

	userRoles, err := stringSetToAPI(ctx, plan.UserRoles)
	if err != nil {
		return nil, fmt.Errorf("invalid user_roles: %w", err)
	}
	ic.UserRoles = userRoles

	headers, err := optionalStringMap(ctx, plan.DefaultHeaders, "default_headers")
	if err != nil {
		return nil, err
	}
	ic.DefaultHeaders = headers

	ic.Features = featuresFromObject(ctx, plan.Features)

	interfaces, err := interfacesFromMap(ctx, plan.Interfaces)
	if err != nil {
		return nil, err
	}
	ic.Interfaces = interfaces

	if ic.Defaults, err = freeFormObjectFromPlan("defaults", plan.Defaults); err != nil {
		return nil, err
	}
	if ic.CatalogProperties, err = freeFormObjectFromPlan("catalog_properties", plan.CatalogProperties); err != nil {
		return nil, err
	}
	return ic, nil
}

// setInterceptorState maps an API interceptor into state. Existing values in
// state act as the prior value for free-form attributes.
func setInterceptorState(ctx context.Context, state *interceptorModel, ic *client.Interceptor) error {
	state.APIName = types.StringValue(ic.Name)
	if state.Name.IsNull() || state.Name.IsUnknown() {
		state.Name = types.StringValue(ic.Name)
	}
	return setInterceptorBodyState(ctx, &state.interceptorBody, ic)
}

func setInterceptorBodyState(ctx context.Context, state *interceptorBody, ic *client.Interceptor) error {
	state.Endpoint = stringValue(ic.Endpoint)
	state.Description = stringValue(ic.Description)
	state.DisplayName = stringValue(ic.DisplayName)
	state.DisplayVersion = stringValue(ic.DisplayVersion)
	state.IconURL = stringValue(ic.IconURL)
	state.CatalogSchemaID = stringValue(ic.CatalogSchemaID)
	state.ForwardAuthToken = boolValue(ic.ForwardAuthToken)
	state.MaxRetryAttempts = int64Value(ic.MaxRetryAttempts)

	userRoles, err := stringSetFromAPI(ctx, ic.UserRoles)
	if err != nil {
		return fmt.Errorf("failed to set user_roles: %w", err)
	}
	state.UserRoles = userRoles

	headers := ic.DefaultHeaders
	if headers == nil {
		headers = map[string]string{}
	}
	headersValue, diags := types.MapValueFrom(ctx, types.StringType, headers)
	if diags.HasError() {
		return fmt.Errorf("failed to set default_headers: %s", diags.Errors()[0].Summary())
	}
	state.DefaultHeaders = headersValue

	features, err := featuresToObject(ctx, ic.Features)
	if err != nil {
		return err
	}
	state.Features = features

	interfaces, err := interfacesToMap(ctx, state.Interfaces, ic.Interfaces)
	if err != nil {
		return err
	}
	state.Interfaces = interfaces

	if state.Defaults, err = freeFormObjectToState(state.Defaults, ic.Defaults); err != nil {
		return fmt.Errorf("failed to set defaults: %w", err)
	}
	if state.CatalogProperties, err = freeFormObjectToState(state.CatalogProperties, ic.CatalogProperties); err != nil {
		return fmt.Errorf("failed to set catalog_properties: %w", err)
	}
	return nil
}

func interceptorID(bucket, path types.String) types.String {
	return types.StringValue(bucket.ValueString() + "/" + path.ValueString())
}

// save PUTs the planned interceptor and re-reads it, because PUT returns only
// the name.
func (r *interceptorResource) save(ctx context.Context, plan *interceptorModel, verb string) error {
	ic, err := interceptorFromPlan(ctx, *plan)
	if err != nil {
		return fmt.Errorf("invalid interceptor configuration: %w", err)
	}
	bucket, path := plan.Bucket.ValueString(), plan.Path.ValueString()
	if err := r.client.SaveInterceptor(ctx, bucket, path, ic); err != nil {
		return fmt.Errorf("failed to %s interceptor: %w", verb, err)
	}
	saved, err := r.client.GetInterceptor(ctx, bucket, path)
	if err != nil {
		return fmt.Errorf("failed to read %sd interceptor: %w", verb, err)
	}
	if saved == nil {
		return fmt.Errorf("interceptor not found after %s: the API returned success but the interceptor could not be read", verb)
	}
	plan.ID = interceptorID(plan.Bucket, plan.Path)
	if err := setInterceptorState(ctx, plan, saved); err != nil {
		return fmt.Errorf("failed to store interceptor state: %w", err)
	}
	return nil
}

func (r *interceptorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan interceptorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.save(ctx, &plan, "create"); err != nil {
		resp.Diagnostics.AddError("Failed to create interceptor", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *interceptorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state interceptorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ic, err := r.client.GetInterceptor(ctx, state.Bucket.ValueString(), state.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read interceptor", err.Error())
		return
	}
	if ic == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	if err := setInterceptorState(ctx, &state, ic); err != nil {
		resp.Diagnostics.AddError("Failed to store interceptor state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *interceptorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan interceptorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.save(ctx, &plan, "update"); err != nil {
		resp.Diagnostics.AddError("Failed to update interceptor", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *interceptorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state interceptorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteInterceptor(ctx, state.Bucket.ValueString(), state.Path.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete interceptor", err.Error())
	}
}

func (r *interceptorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Use terraform import aidial_interceptor.example <bucket>/<path>.",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bucket"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), parts[1])...)
}
