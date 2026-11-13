package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource                = &toolsetResource{}
	_ resource.ResourceWithImportState = &toolsetResource{}
)

// toolsetTransports and toolsetAuthTypes are the enums of the OpenAPI
// ToolSetTransport and ResourceAuthSettings.authentication_type.
var (
	toolsetTransports = []string{"HTTP", "SSE"}
	toolsetAuthTypes  = []string{"OAUTH", "API_KEY", "NONE", "DIAL_NATIVE"}
)

type toolsetResource struct {
	client *client.Client
}

// toolsetModel is the state of aidial_toolset and of the aidial_toolset data
// source. Terraform names are snake_case; `provider_name` is the API's
// `provider`, which Terraform reserves as a top-level attribute name.
type toolsetModel struct {
	ID      types.String `tfsdk:"id"`
	Bucket  types.String `tfsdk:"bucket"`
	Path    types.String `tfsdk:"path"`
	Name    types.String `tfsdk:"name"`
	APIName types.String `tfsdk:"api_name"`

	toolsetBody
}

// toolsetBody holds every attribute of a toolset except its identity. It is
// kept separate so that all three entry points (resource, data source, file
// data source) share one mapping.
type toolsetBody struct {
	DisplayName       types.String `tfsdk:"display_name"`
	Description       types.String `tfsdk:"description"`
	Author            types.String `tfsdk:"author"`
	Intro             types.String `tfsdk:"intro"`
	DisplayVersion    types.String `tfsdk:"display_version"`
	IconURL           types.String `tfsdk:"icon_url"`
	Reference         types.String `tfsdk:"reference"`
	ProviderName      types.String `tfsdk:"provider_name"`
	VendorWebsite     types.String `tfsdk:"vendor_website"`
	Endpoint          types.String `tfsdk:"endpoint"`
	ResponsesEndpoint types.String `tfsdk:"responses_endpoint"`
	BaseURL           types.String `tfsdk:"base_url"`
	OverrideName      types.String `tfsdk:"override_name"`
	CatalogSchemaID   types.String `tfsdk:"catalog_schema_id"`
	Transport         types.String `tfsdk:"transport"`

	ForwardAuthToken     types.Bool  `tfsdk:"forward_auth_token"`
	ForwardPerRequestKey types.Bool  `tfsdk:"forward_per_request_key"`
	MaxInputAttachments  types.Int64 `tfsdk:"max_input_attachments"`
	MaxRetryAttempts     types.Int64 `tfsdk:"max_retry_attempts"`

	AllowedTools         types.Set `tfsdk:"allowed_tools"`
	Dependencies         types.Set `tfsdk:"dependencies"`
	DescriptionKeywords  types.Set `tfsdk:"description_keywords"`
	InputAttachmentTypes types.Set `tfsdk:"input_attachment_types"`
	Interceptors         types.Set `tfsdk:"interceptors"`
	UserRoles            types.Set `tfsdk:"user_roles"`

	DefaultHeaders types.Map    `tfsdk:"default_headers"`
	Features       types.Object `tfsdk:"features"`
	Interfaces     types.Map    `tfsdk:"interfaces"`
	AuthSettings   types.Object `tfsdk:"auth_settings"`

	Defaults          types.Dynamic `tfsdk:"defaults"`
	ResponsesDefaults types.Dynamic `tfsdk:"responses_defaults"`
	CatalogProperties types.Dynamic `tfsdk:"catalog_properties"`
}

type toolsetAuthSettingsModel struct {
	AuthenticationType      types.String `tfsdk:"authentication_type"`
	APIKeyHeader            types.String `tfsdk:"api_key_header"`
	AuthorizationEndpoint   types.String `tfsdk:"authorization_endpoint"`
	TokenEndpoint           types.String `tfsdk:"token_endpoint"`
	TokenEndpointAuthMethod types.String `tfsdk:"token_endpoint_auth_method"`
	ClientID                types.String `tfsdk:"client_id"`
	ClientSecret            types.String `tfsdk:"client_secret"`
	RedirectURI             types.String `tfsdk:"redirect_uri"`
	ScopesSupported         types.Set    `tfsdk:"scopes_supported"`
	CodeChallengeMethod     types.String `tfsdk:"code_challenge_method"`
	Resource                types.String `tfsdk:"resource"`
	DynamicallyRegistered   types.Bool   `tfsdk:"dynamically_registered"`
}

// toolsetAuthSettingsAttrTypes are the attribute types of auth_settings on
// the resource. The data sources omit client_secret.
var toolsetAuthSettingsAttrTypes = map[string]attr.Type{
	"authentication_type":        types.StringType,
	"api_key_header":             types.StringType,
	"authorization_endpoint":     types.StringType,
	"token_endpoint":             types.StringType,
	"token_endpoint_auth_method": types.StringType,
	"client_id":                  types.StringType,
	"client_secret":              types.StringType,
	"redirect_uri":               types.StringType,
	"scopes_supported":           types.SetType{ElemType: types.StringType},
	"code_challenge_method":      types.StringType,
	"resource":                   types.StringType,
	"dynamically_registered":     types.BoolType,
}

// toolsetDataSourceAuthSettingsAttrTypes are the attribute types of
// auth_settings on the data sources.
var toolsetDataSourceAuthSettingsAttrTypes = func() map[string]attr.Type {
	types := make(map[string]attr.Type, len(toolsetAuthSettingsAttrTypes)-1)
	for name, t := range toolsetAuthSettingsAttrTypes {
		if name != "client_secret" {
			types[name] = t
		}
	}
	return types
}()

func (r *toolsetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_toolset"
}

func (r *toolsetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	str := func(description string) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Description: description}
	}
	set := func(description string) schema.SetAttribute {
		return schema.SetAttribute{Optional: true, ElementType: types.StringType, Description: description}
	}
	// DIAL Core fills these in itself when they are not sent (an empty list,
	// a default flag, a generated reference), so they are computed as well.
	// UseStateForUnknown keeps the value stable across updates, because Core
	// would otherwise regenerate it on every PUT.
	defaultedStr := func(description string) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, Description: description}
	}
	defaultedBool := func(description string) schema.BoolAttribute {
		return schema.BoolAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}, Description: description}
	}
	defaultedInt := func(description string) schema.Int64Attribute {
		return schema.Int64Attribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}, Description: description}
	}
	defaultedSet := func(description string) schema.SetAttribute {
		return schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType, PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()}, Description: description}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a toolset in DIAL Core (`PUT /v1/toolsets/{bucket}/{path}`).",
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
				Description:   "Target DIAL bucket, for example `platform` or `public`. DIAL Core decides which buckets it accepts for toolsets.",
			},
			"path": schema.StringAttribute{
				Required:      true,
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Toolset path inside the target bucket.",
			},
			"name": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "Toolset name sent in the DIAL API request body. DIAL Core derives the name from the bucket and path, so it can be omitted; then it is read back from the API.",
			},
			"api_name": schema.StringAttribute{
				Computed:    true,
				Description: "Canonical toolset name returned by the DIAL API.",
			},
			"display_name": str("Human-readable toolset name."),
			"description":  str("Toolset description."),
			"author": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "Author of the toolset, set by DIAL Core from the caller's identity.",
			},
			"intro":              str("Introductory text shown for the toolset."),
			"display_version":    str("Version shown for the toolset."),
			"icon_url":           str("URL of the toolset icon."),
			"reference":          defaultedStr("Reference identifier of the toolset. DIAL Core generates one when it is not set."),
			"provider_name":      str("Name of the toolset provider (`provider` in the DIAL API; Terraform reserves that attribute name)."),
			"vendor_website":     str("Website of the toolset vendor."),
			"endpoint":           str("Toolset MCP endpoint, for example `http://toolset/v1/mcp`."),
			"responses_endpoint": str("Endpoint used for Responses API requests."),
			"base_url":           str("Toolset-level base URL that completes `interfaces` entries without their own `base_url`."),
			"override_name":      str("Override of the toolset name presented to clients."),
			"catalog_schema_id":  str("URI (`$id`) of the catalog schema describing `catalog_properties`, for example the `schema_id` of an `aidial_catalog_schema`."),
			"transport": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators:    []validator.String{stringvalidator.OneOf(toolsetTransports...)},
				Description:   "MCP transport: `" + strings.Join(toolsetTransports, "` or `") + "`. DIAL Core defaults to `HTTP`.",
			},
			"forward_auth_token":      defaultedBool("Forward the caller's authorization token to the toolset. Defaults to false."),
			"forward_per_request_key": defaultedBool("Forward the per-request key to the toolset. Defaults to false."),
			"max_input_attachments":   schema.Int64Attribute{Optional: true, Description: "Maximum number of attachments in one request."},
			"max_retry_attempts":      defaultedInt("Maximum number of retry attempts. DIAL Core defaults to 1."),
			"allowed_tools":           defaultedSet("Names of the tools the toolset exposes. An empty set, which DIAL Core defaults to, exposes all of them."),
			"dependencies":            defaultedSet("Deployments the toolset depends on."),
			"description_keywords":    defaultedSet("Keywords describing the toolset."),
			"input_attachment_types":  set("MIME types of accepted input attachments."),
			"interceptors":            defaultedSet("Interceptors applied to requests."),
			"user_roles":              set("Roles allowed to use the toolset."),
			"default_headers": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Additional headers sent with every request to the toolset's endpoint.",
			},
			"features":   featuresResourceAttribute(),
			"interfaces": interfacesResourceAttribute(),
			"auth_settings": schema.SingleNestedAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
				Description:   "How DIAL Core authenticates to the toolset. Only configuration is exposed; session state such as sign-in status is not.",
				Attributes: map[string]schema.Attribute{
					"authentication_type": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Validators:  []validator.String{stringvalidator.OneOf(toolsetAuthTypes...)},
						Description: "One of `" + strings.Join(toolsetAuthTypes, "`, `") + "`. DIAL Core defaults to `NONE`.",
					},
					"api_key_header":             schema.StringAttribute{Optional: true, Description: "Header that carries the API key (for `API_KEY`)."},
					"authorization_endpoint":     schema.StringAttribute{Optional: true},
					"token_endpoint":             schema.StringAttribute{Optional: true},
					"token_endpoint_auth_method": schema.StringAttribute{Optional: true},
					"client_id":                  schema.StringAttribute{Optional: true},
					"client_secret": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "OAuth client secret. DIAL Core never returns it, so the configured value is kept in state; it is null after import.",
					},
					"redirect_uri":           schema.StringAttribute{Optional: true},
					"scopes_supported":       schema.SetAttribute{Optional: true, ElementType: types.StringType},
					"code_challenge_method":  schema.StringAttribute{Optional: true},
					"resource":               schema.StringAttribute{Optional: true},
					"dynamically_registered": schema.BoolAttribute{Optional: true},
				},
			},
			"defaults": schema.DynamicAttribute{
				Optional:    true,
				Description: "Default request parameters (free-form object, sent to DIAL as-is).",
			},
			"responses_defaults": schema.DynamicAttribute{
				Optional:    true,
				Description: "Default request parameters for Responses API requests (free-form object, sent as `responsesDefaults`).",
			},
			"catalog_properties": schema.DynamicAttribute{
				Optional:    true,
				Description: "Catalog metadata of the toolset (free-form object, sent as `catalogProperties`).",
			},
		},
	}
}

func (r *toolsetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// toolsetFromPlan converts planned values to the API shape. Null and unknown
// attributes are not sent; a configured empty set is sent as [].
func toolsetFromPlan(ctx context.Context, plan toolsetModel) (*client.ToolSet, error) {
	ts := &client.ToolSet{
		Name:                 plan.Name.ValueString(),
		DisplayName:          stringPointer(plan.DisplayName),
		Description:          stringPointer(plan.Description),
		Intro:                stringPointer(plan.Intro),
		DisplayVersion:       stringPointer(plan.DisplayVersion),
		IconURL:              stringPointer(plan.IconURL),
		Reference:            stringPointer(plan.Reference),
		Provider:             stringPointer(plan.ProviderName),
		VendorWebsite:        stringPointer(plan.VendorWebsite),
		Endpoint:             stringPointer(plan.Endpoint),
		ResponsesEndpoint:    stringPointer(plan.ResponsesEndpoint),
		BaseURL:              stringPointer(plan.BaseURL),
		OverrideName:         stringPointer(plan.OverrideName),
		CatalogSchemaID:      stringPointer(plan.CatalogSchemaID),
		Transport:            stringPointer(plan.Transport),
		ForwardAuthToken:     boolPointer(plan.ForwardAuthToken),
		ForwardPerRequestKey: boolPointer(plan.ForwardPerRequestKey),
		MaxInputAttachments:  int64Pointer(plan.MaxInputAttachments),
		MaxRetryAttempts:     int64Pointer(plan.MaxRetryAttempts),
	}

	for _, list := range []struct {
		name   string
		value  types.Set
		target **[]string
	}{
		{"allowed_tools", plan.AllowedTools, &ts.AllowedTools},
		{"dependencies", plan.Dependencies, &ts.Dependencies},
		{"description_keywords", plan.DescriptionKeywords, &ts.DescriptionKeywords},
		{"input_attachment_types", plan.InputAttachmentTypes, &ts.InputAttachmentTypes},
		{"interceptors", plan.Interceptors, &ts.Interceptors},
		{"user_roles", plan.UserRoles, &ts.UserRoles},
	} {
		value, err := stringSetToAPI(ctx, list.value)
		if err != nil {
			return nil, fmt.Errorf("invalid %s: %w", list.name, err)
		}
		*list.target = value
	}

	headers, err := optionalStringMap(ctx, plan.DefaultHeaders, "default_headers")
	if err != nil {
		return nil, err
	}
	ts.DefaultHeaders = headers

	ts.Features = featuresFromObject(ctx, plan.Features)

	interfaces, err := interfacesFromMap(ctx, plan.Interfaces)
	if err != nil {
		return nil, err
	}
	ts.Interfaces = interfaces

	auth, err := toolsetAuthFromObject(ctx, plan.AuthSettings)
	if err != nil {
		return nil, err
	}
	ts.AuthSettings = auth

	if ts.Defaults, err = freeFormObjectFromPlan("defaults", plan.Defaults); err != nil {
		return nil, err
	}
	if ts.ResponsesDefaults, err = freeFormObjectFromPlan("responses_defaults", plan.ResponsesDefaults); err != nil {
		return nil, err
	}
	if ts.CatalogProperties, err = freeFormObjectFromPlan("catalog_properties", plan.CatalogProperties); err != nil {
		return nil, err
	}
	return ts, nil
}

func toolsetAuthFromObject(ctx context.Context, value types.Object) (*client.ToolSetAuthSettings, error) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	var a toolsetAuthSettingsModel
	if diags := value.As(ctx, &a, basetypes.ObjectAsOptions{}); diags.HasError() {
		return nil, fmt.Errorf("invalid auth_settings: %s", diags.Errors()[0].Summary())
	}
	scopes, err := stringSetToAPI(ctx, a.ScopesSupported)
	if err != nil {
		return nil, fmt.Errorf("invalid auth_settings.scopes_supported: %w", err)
	}
	return &client.ToolSetAuthSettings{
		AuthenticationType:      stringPointer(a.AuthenticationType),
		APIKeyHeader:            stringPointer(a.APIKeyHeader),
		AuthorizationEndpoint:   stringPointer(a.AuthorizationEndpoint),
		TokenEndpoint:           stringPointer(a.TokenEndpoint),
		TokenEndpointAuthMethod: stringPointer(a.TokenEndpointAuthMethod),
		ClientID:                stringPointer(a.ClientID),
		ClientSecret:            stringPointer(a.ClientSecret),
		RedirectURI:             stringPointer(a.RedirectURI),
		ScopesSupported:         scopes,
		CodeChallengeMethod:     stringPointer(a.CodeChallengeMethod),
		Resource:                stringPointer(a.Resource),
		DynamicallyRegistered:   boolPointer(a.DynamicallyRegistered),
	}, nil
}

// toolsetAuthToObject maps the API auth settings to state. client_secret is
// never returned by Core, so it is carried over from prior state when
// withSecret is set, and omitted from the object otherwise (data sources).
func toolsetAuthToObject(ctx context.Context, prior types.Object, api *client.ToolSetAuthSettings, withSecret bool) (types.Object, error) {
	attrTypes := toolsetDataSourceAuthSettingsAttrTypes
	if withSecret {
		attrTypes = toolsetAuthSettingsAttrTypes
	}
	if api == nil {
		return types.ObjectNull(attrTypes), nil
	}

	scopes, err := stringSetFromAPI(ctx, api.ScopesSupported)
	if err != nil {
		return types.ObjectNull(attrTypes), fmt.Errorf("failed to set auth_settings.scopes_supported: %w", err)
	}
	values := map[string]attr.Value{
		"authentication_type":        stringValue(api.AuthenticationType),
		"api_key_header":             stringValue(api.APIKeyHeader),
		"authorization_endpoint":     stringValue(api.AuthorizationEndpoint),
		"token_endpoint":             stringValue(api.TokenEndpoint),
		"token_endpoint_auth_method": stringValue(api.TokenEndpointAuthMethod),
		"client_id":                  stringValue(api.ClientID),
		"redirect_uri":               stringValue(api.RedirectURI),
		"scopes_supported":           scopes,
		"code_challenge_method":      stringValue(api.CodeChallengeMethod),
		"resource":                   stringValue(api.Resource),
		"dynamically_registered":     boolValue(api.DynamicallyRegistered),
	}
	if withSecret {
		secret := types.StringNull()
		if !prior.IsNull() && !prior.IsUnknown() {
			var p toolsetAuthSettingsModel
			if diags := prior.As(ctx, &p, basetypes.ObjectAsOptions{}); !diags.HasError() {
				secret = p.ClientSecret
			}
		}
		// A secret echoed by Core is not authoritative: the configured value wins.
		if secret.IsNull() || secret.IsUnknown() {
			secret = stringValue(api.ClientSecret)
		}
		values["client_secret"] = secret
	}

	obj, diags := types.ObjectValue(attrTypes, values)
	if diags.HasError() {
		return types.ObjectNull(attrTypes), fmt.Errorf("failed to build auth_settings object: %s", diags.Errors()[0].Summary())
	}
	return obj, nil
}

// setToolsetState maps an API toolset into state. Existing values in state
// act as the prior value for write-only and free-form attributes. withSecret
// is false for the data sources, which do not expose client_secret.
func setToolsetState(ctx context.Context, state *toolsetModel, ts *client.ToolSet, withSecret bool) error {
	state.APIName = types.StringValue(ts.Name)
	if state.Name.IsNull() || state.Name.IsUnknown() {
		state.Name = types.StringValue(ts.Name)
	}
	return setToolsetBodyState(ctx, &state.toolsetBody, ts, withSecret)
}

func setToolsetBodyState(ctx context.Context, state *toolsetBody, ts *client.ToolSet, withSecret bool) error {
	state.DisplayName = stringValue(ts.DisplayName)
	state.Description = stringValue(ts.Description)
	state.Author = stringValue(ts.Author)
	state.Intro = stringValue(ts.Intro)
	state.DisplayVersion = stringValue(ts.DisplayVersion)
	state.IconURL = stringValue(ts.IconURL)
	state.Reference = stringValue(ts.Reference)
	state.ProviderName = stringValue(ts.Provider)
	state.VendorWebsite = stringValue(ts.VendorWebsite)
	state.Endpoint = stringValue(ts.Endpoint)
	state.ResponsesEndpoint = stringValue(ts.ResponsesEndpoint)
	state.BaseURL = stringValue(ts.BaseURL)
	state.OverrideName = stringValue(ts.OverrideName)
	state.CatalogSchemaID = stringValue(ts.CatalogSchemaID)
	state.Transport = stringValue(ts.Transport)
	state.ForwardAuthToken = boolValue(ts.ForwardAuthToken)
	state.ForwardPerRequestKey = boolValue(ts.ForwardPerRequestKey)
	state.MaxInputAttachments = int64Value(ts.MaxInputAttachments)
	state.MaxRetryAttempts = int64Value(ts.MaxRetryAttempts)

	for _, list := range []struct {
		name   string
		value  *[]string
		target *types.Set
	}{
		{"allowed_tools", ts.AllowedTools, &state.AllowedTools},
		{"dependencies", ts.Dependencies, &state.Dependencies},
		{"description_keywords", ts.DescriptionKeywords, &state.DescriptionKeywords},
		{"input_attachment_types", ts.InputAttachmentTypes, &state.InputAttachmentTypes},
		{"interceptors", ts.Interceptors, &state.Interceptors},
		{"user_roles", ts.UserRoles, &state.UserRoles},
	} {
		value, err := stringSetFromAPI(ctx, list.value)
		if err != nil {
			return fmt.Errorf("failed to set %s: %w", list.name, err)
		}
		*list.target = value
	}

	headers := ts.DefaultHeaders
	if headers == nil {
		headers = map[string]string{}
	}
	headersValue, diags := types.MapValueFrom(ctx, types.StringType, headers)
	if diags.HasError() {
		return fmt.Errorf("failed to set default_headers: %s", diags.Errors()[0].Summary())
	}
	state.DefaultHeaders = headersValue

	features, err := featuresToObject(ctx, ts.Features)
	if err != nil {
		return err
	}
	state.Features = features

	interfaces, err := interfacesToMap(ctx, state.Interfaces, ts.Interfaces)
	if err != nil {
		return err
	}
	state.Interfaces = interfaces

	auth, err := toolsetAuthToObject(ctx, state.AuthSettings, ts.AuthSettings, withSecret)
	if err != nil {
		return err
	}
	state.AuthSettings = auth

	if state.Defaults, err = freeFormObjectToState(state.Defaults, ts.Defaults); err != nil {
		return fmt.Errorf("failed to set defaults: %w", err)
	}
	if state.ResponsesDefaults, err = freeFormObjectToState(state.ResponsesDefaults, ts.ResponsesDefaults); err != nil {
		return fmt.Errorf("failed to set responses_defaults: %w", err)
	}
	if state.CatalogProperties, err = freeFormObjectToState(state.CatalogProperties, ts.CatalogProperties); err != nil {
		return fmt.Errorf("failed to set catalog_properties: %w", err)
	}
	return nil
}

func toolsetID(bucket, path types.String) types.String {
	return types.StringValue(bucket.ValueString() + "/" + path.ValueString())
}

// save PUTs the planned toolset and re-reads it, because PUT returns only
// the name.
func (r *toolsetResource) save(ctx context.Context, plan *toolsetModel, verb string) error {
	ts, err := toolsetFromPlan(ctx, *plan)
	if err != nil {
		return fmt.Errorf("invalid toolset configuration: %w", err)
	}
	bucket, path := plan.Bucket.ValueString(), plan.Path.ValueString()
	if err := r.client.SaveToolset(ctx, bucket, path, ts); err != nil {
		return fmt.Errorf("failed to %s toolset: %w", verb, err)
	}
	saved, err := r.client.GetToolset(ctx, bucket, path)
	if err != nil {
		return fmt.Errorf("failed to read %sd toolset: %w", verb, err)
	}
	if saved == nil {
		return fmt.Errorf("toolset not found after %s: the API returned success but the toolset could not be read", verb)
	}
	plan.ID = toolsetID(plan.Bucket, plan.Path)
	if err := setToolsetState(ctx, plan, saved, true); err != nil {
		return fmt.Errorf("failed to store toolset state: %w", err)
	}
	return nil
}

func (r *toolsetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan toolsetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.save(ctx, &plan, "create"); err != nil {
		resp.Diagnostics.AddError("Failed to create toolset", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *toolsetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state toolsetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ts, err := r.client.GetToolset(ctx, state.Bucket.ValueString(), state.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read toolset", err.Error())
		return
	}
	if ts == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	if err := setToolsetState(ctx, &state, ts, true); err != nil {
		resp.Diagnostics.AddError("Failed to store toolset state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *toolsetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan toolsetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.save(ctx, &plan, "update"); err != nil {
		resp.Diagnostics.AddError("Failed to update toolset", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *toolsetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state toolsetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteToolset(ctx, state.Bucket.ValueString(), state.Path.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete toolset", err.Error())
	}
}

func (r *toolsetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Use terraform import aidial_toolset.example <bucket>/<path>.",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bucket"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), parts[1])...)
}
