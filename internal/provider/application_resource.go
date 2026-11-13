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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &applicationResource{}
	_ resource.ResourceWithImportState = &applicationResource{}
)

type applicationResource struct {
	client *client.Client
}

// applicationModel is the state of aidial_application and of the
// aidial_application data source.
type applicationModel struct {
	ID      types.String `tfsdk:"id"`
	Bucket  types.String `tfsdk:"bucket"`
	Path    types.String `tfsdk:"path"`
	Name    types.String `tfsdk:"name"`
	APIName types.String `tfsdk:"api_name"`

	applicationBody
}

// applicationBody holds every attribute of an application except its identity.
// It is kept separate so that all three entry points (resource, data source,
// file data source) share one mapping.
type applicationBody struct {
	Endpoint                types.String `tfsdk:"endpoint"`
	Description             types.String `tfsdk:"description"`
	DisplayName             types.String `tfsdk:"display_name"`
	DisplayVersion          types.String `tfsdk:"display_version"`
	IconURL                 types.String `tfsdk:"icon_url"`
	CatalogSchemaID         types.String `tfsdk:"catalog_schema_id"`
	EditorURL               types.String `tfsdk:"editor_url"`
	ViewerURL               types.String `tfsdk:"viewer_url"`
	ApplicationTypeSchemaID types.String `tfsdk:"application_type_schema_id"`
	ForwardAuthToken        types.Bool   `tfsdk:"forward_auth_token"`
	MaxRetryAttempts        types.Int64  `tfsdk:"max_retry_attempts"`

	UserRoles      types.Set    `tfsdk:"user_roles"`
	Interceptors   types.Set    `tfsdk:"interceptors"`
	DefaultHeaders types.Map    `tfsdk:"default_headers"`
	Features       types.Object `tfsdk:"features"`
	Interfaces     types.Map    `tfsdk:"interfaces"`

	Defaults              types.Dynamic `tfsdk:"defaults"`
	ApplicationProperties types.Dynamic `tfsdk:"application_properties"`
	CatalogProperties     types.Dynamic `tfsdk:"catalog_properties"`
}

func (r *applicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

func (r *applicationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	str := func(description string) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Description: description}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an application in DIAL Core (`PUT /v1/applications/{bucket}/{path}`). An application is either endpoint-based (`endpoint`) or schema-based (`application_type_schema_id` with `application_properties`); DIAL Core rejects both together. Code-based applications (`function`), `mcp`, `routes` and `external_services` are not managed: a PUT replaces the whole document, so they are dropped on the next apply.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "Resource identifier in the form <bucket>/<path>.",
			},
			"bucket": schema.StringAttribute{
				Required:      true,
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Target DIAL bucket, for example `platform` or `public`.",
			},
			"path": schema.StringAttribute{
				Required:      true,
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Application path inside the bucket. `/` separates folders; DIAL Core accepts `/` only in the `public` bucket.",
			},
			"name": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "Application name sent in the DIAL API request body. DIAL Core derives the name from the bucket and path, so it can be omitted; then it is read back from the API.",
			},
			"api_name": schema.StringAttribute{
				Computed:    true,
				Description: "Canonical application name returned by the DIAL API.",
			},
			"endpoint": schema.StringAttribute{
				Optional:    true,
				Description: "URL DIAL Core calls to run the application. Set exactly one of `endpoint` and `application_type_schema_id`.",
				Validators:  []validator.String{stringvalidator.ExactlyOneOf(path.MatchRoot("application_type_schema_id"))},
			},
			"application_type_schema_id": str("URI (`$id`) of the application type schema, for example the `schema_id` of an `aidial_application_type_schema`. Schema-based applications take no `endpoint`."),
			"editor_url":                 str("URL of the application editor."),
			"viewer_url":                 str("URL of the application viewer."),
			"description":                str("Application description."),
			"display_name":               str("Human-readable application name."),
			"display_version":            str("Version shown for the application."),
			"icon_url":                   str("URL of the application icon."),
			"catalog_schema_id":          str("URI (`$id`) of the catalog schema describing `catalog_properties`, for example the `schema_id` of an `aidial_catalog_schema`."),
			// DIAL Core fills these two in when they are not sent, so they are
			// computed too. UseStateForUnknown keeps them stable across updates.
			"forward_auth_token": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
				Description:   "Forward the caller's authorization token to the application. DIAL Core defaults to false.",
			},
			"max_retry_attempts": schema.Int64Attribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Description:   "Maximum number of retry attempts. DIAL Core defaults to 1.",
			},
			"user_roles": schema.SetAttribute{Optional: true, ElementType: types.StringType, Description: "Roles allowed to use the application."},
			"interceptors": schema.SetAttribute{
				Optional:      true,
				Computed:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
				Description:   "Names of interceptors applied to the application's requests. DIAL Core defaults to none.",
			},
			"default_headers": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Additional headers sent with every request to the application's endpoint.",
			},
			"features":   featuresResourceAttribute(),
			"interfaces": interfacesResourceAttribute(),
			"defaults": schema.DynamicAttribute{
				Optional:    true,
				Description: "Default request parameters (free-form object, sent to DIAL as-is).",
			},
			"application_properties": schema.DynamicAttribute{
				Optional:    true,
				Description: "Values for the properties of the application type schema (free-form object, sent to DIAL as-is).",
			},
			"catalog_properties": schema.DynamicAttribute{
				Optional:    true,
				Description: "Catalog metadata of the application (free-form object).",
			},
		},
	}
}

func (r *applicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// applicationFromPlan converts planned values to the API shape. Null and
// unknown attributes are not sent; a configured empty set is sent as [].
func applicationFromPlan(ctx context.Context, plan applicationModel) (*client.Application, error) {
	ic := &client.Application{
		Name:                    plan.Name.ValueString(),
		Endpoint:                stringPointer(plan.Endpoint),
		Description:             stringPointer(plan.Description),
		DisplayName:             stringPointer(plan.DisplayName),
		DisplayVersion:          stringPointer(plan.DisplayVersion),
		IconURL:                 stringPointer(plan.IconURL),
		CatalogSchemaID:         stringPointer(plan.CatalogSchemaID),
		EditorURL:               stringPointer(plan.EditorURL),
		ViewerURL:               stringPointer(plan.ViewerURL),
		ApplicationTypeSchemaID: stringPointer(plan.ApplicationTypeSchemaID),
		ForwardAuthToken:        boolPointer(plan.ForwardAuthToken),
		MaxRetryAttempts:        int64Pointer(plan.MaxRetryAttempts),
	}

	userRoles, err := stringSetToAPI(ctx, plan.UserRoles)
	if err != nil {
		return nil, fmt.Errorf("invalid user_roles: %w", err)
	}
	ic.UserRoles = userRoles

	interceptors, err := stringSetToAPI(ctx, plan.Interceptors)
	if err != nil {
		return nil, fmt.Errorf("invalid interceptors: %w", err)
	}
	ic.Interceptors = interceptors

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
	if ic.ApplicationProperties, err = freeFormObjectFromPlan("application_properties", plan.ApplicationProperties); err != nil {
		return nil, err
	}
	if ic.CatalogProperties, err = freeFormObjectFromPlan("catalog_properties", plan.CatalogProperties); err != nil {
		return nil, err
	}
	return ic, nil
}

// setApplicationState maps an API application into state. Existing values in
// state act as the prior value for free-form attributes.
func setApplicationState(ctx context.Context, state *applicationModel, ic *client.Application) error {
	state.APIName = types.StringValue(ic.Name)
	if state.Name.IsNull() || state.Name.IsUnknown() {
		state.Name = types.StringValue(ic.Name)
	}
	return setApplicationBodyState(ctx, &state.applicationBody, ic)
}

func setApplicationBodyState(ctx context.Context, state *applicationBody, ic *client.Application) error {
	state.Endpoint = stringValue(ic.Endpoint)
	state.Description = stringValue(ic.Description)
	state.DisplayName = stringValue(ic.DisplayName)
	state.DisplayVersion = stringValue(ic.DisplayVersion)
	state.IconURL = stringValue(ic.IconURL)
	state.CatalogSchemaID = stringValue(ic.CatalogSchemaID)
	state.EditorURL = stringValue(ic.EditorURL)
	state.ViewerURL = stringValue(ic.ViewerURL)
	state.ApplicationTypeSchemaID = stringValue(ic.ApplicationTypeSchemaID)
	state.ForwardAuthToken = boolValue(ic.ForwardAuthToken)
	state.MaxRetryAttempts = int64Value(ic.MaxRetryAttempts)

	userRoles, err := stringSetFromAPI(ctx, ic.UserRoles)
	if err != nil {
		return fmt.Errorf("failed to set user_roles: %w", err)
	}
	state.UserRoles = userRoles

	interceptors, err := stringSetFromAPI(ctx, ic.Interceptors)
	if err != nil {
		return fmt.Errorf("failed to set interceptors: %w", err)
	}
	state.Interceptors = interceptors

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
	if state.ApplicationProperties, err = freeFormObjectToState(state.ApplicationProperties, ic.ApplicationProperties); err != nil {
		return fmt.Errorf("failed to set application_properties: %w", err)
	}
	if state.CatalogProperties, err = freeFormObjectToState(state.CatalogProperties, ic.CatalogProperties); err != nil {
		return fmt.Errorf("failed to set catalog_properties: %w", err)
	}
	return nil
}

func applicationID(bucket, path types.String) types.String {
	return types.StringValue(bucket.ValueString() + "/" + path.ValueString())
}

// save PUTs the planned application and re-reads it, because PUT returns only
// the name.
func (r *applicationResource) save(ctx context.Context, plan *applicationModel, verb string) error {
	ic, err := applicationFromPlan(ctx, *plan)
	if err != nil {
		return fmt.Errorf("invalid application configuration: %w", err)
	}
	bucket, path := plan.Bucket.ValueString(), plan.Path.ValueString()
	if err := r.client.SaveApplication(ctx, bucket, path, ic); err != nil {
		return fmt.Errorf("failed to %s application: %w", verb, err)
	}
	saved, err := r.client.GetApplication(ctx, bucket, path)
	if err != nil {
		return fmt.Errorf("failed to read %sd application: %w", verb, err)
	}
	if saved == nil {
		return fmt.Errorf("application not found after %s: the API returned success but the application could not be read", verb)
	}
	plan.ID = applicationID(plan.Bucket, plan.Path)
	if err := setApplicationState(ctx, plan, saved); err != nil {
		return fmt.Errorf("failed to store application state: %w", err)
	}
	return nil
}

func (r *applicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan applicationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.save(ctx, &plan, "create"); err != nil {
		resp.Diagnostics.AddError("Failed to create application", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *applicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state applicationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ic, err := r.client.GetApplication(ctx, state.Bucket.ValueString(), state.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read application", err.Error())
		return
	}
	if ic == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	if err := setApplicationState(ctx, &state, ic); err != nil {
		resp.Diagnostics.AddError("Failed to store application state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *applicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan applicationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.save(ctx, &plan, "update"); err != nil {
		resp.Diagnostics.AddError("Failed to update application", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *applicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state applicationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteApplication(ctx, state.Bucket.ValueString(), state.Path.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete application", err.Error())
	}
}

func (r *applicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Use terraform import aidial_application.example <bucket>/<path>.",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bucket"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), parts[1])...)
}
