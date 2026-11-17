package provider

import (
	"context"
	"fmt"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// This file holds the typed nested attributes of aidial_application: mcp,
// routes and external_services. The routes schema is the application's own
// copy and does not depend on aidial_route.

var applicationMcpDeliveries = []string{"HEADER", "META"}

var (
	appMcpAppsAttrTypes = map[string]attr.Type{"domain_override": types.StringType}

	appMcpAttrTypes = map[string]attr.Type{
		"endpoint":                types.StringType,
		"transport":               types.StringType,
		"allowed_tools":           types.SetType{ElemType: types.StringType},
		"config_delivery":         types.StringType,
		"forward_per_request_key": types.BoolType,
		"mcp_apps":                types.ObjectType{AttrTypes: appMcpAppsAttrTypes},
	}

	appRouteResponseAttrTypes = map[string]attr.Type{
		"body":   types.StringType,
		"status": types.Int64Type,
	}

	appRouteAttachmentAttrTypes = map[string]attr.Type{
		"request_body":  types.SetType{ElemType: types.StringType},
		"response_body": types.SetType{ElemType: types.StringType},
	}

	appRouteUpstreamAttrTypes = map[string]attr.Type{
		"endpoint":           types.StringType,
		"responses_endpoint": types.StringType,
		"id":                 types.StringType,
		"extra_data":         types.MapType{ElemType: types.StringType},
		"key":                types.StringType,
		"secret_extra_data":  types.MapType{ElemType: types.StringType},
		"tier":               types.Int64Type,
		"weight":             types.Int64Type,
		"base_url":           types.StringType,
		"auth_type":          types.StringType,
		"auth_header_name":   types.StringType,
	}

	appRouteAttrTypes = map[string]attr.Type{
		"order":              types.Int64Type,
		"methods":            types.SetType{ElemType: types.StringType},
		"paths":              types.SetType{ElemType: types.StringType},
		"permissions":        types.SetType{ElemType: types.StringType},
		"user_roles":         types.SetType{ElemType: types.StringType},
		"max_retry_attempts": types.Int64Type,
		"rewrite_path":       types.BoolType,
		"response":           types.ObjectType{AttrTypes: appRouteResponseAttrTypes},
		"attachment_paths":   types.ObjectType{AttrTypes: appRouteAttachmentAttrTypes},
		"upstreams":          types.ListType{ElemType: types.ObjectType{AttrTypes: appRouteUpstreamAttrTypes}},
	}

	appAuthAttrTypes = map[string]attr.Type{
		"authentication_type":    types.StringType,
		"client_id":              types.StringType,
		"client_secret":          types.StringType,
		"authorization_endpoint": types.StringType,
		"token_endpoint":         types.StringType,
		"scopes_supported":       types.SetType{ElemType: types.StringType},
		"api_key_header":         types.StringType,
		"code_verifier":          types.StringType,
	}

	appServiceAttrTypes = map[string]attr.Type{
		"display_name":  types.StringType,
		"description":   types.StringType,
		"auth_settings": types.ObjectType{AttrTypes: appAuthAttrTypes},
	}
)

type appMcpModel struct {
	Endpoint             types.String `tfsdk:"endpoint"`
	Transport            types.String `tfsdk:"transport"`
	AllowedTools         types.Set    `tfsdk:"allowed_tools"`
	ConfigDelivery       types.String `tfsdk:"config_delivery"`
	ForwardPerRequestKey types.Bool   `tfsdk:"forward_per_request_key"`
	McpApps              types.Object `tfsdk:"mcp_apps"`
}

type appMcpAppsModel struct {
	DomainOverride types.String `tfsdk:"domain_override"`
}

type appRouteModel struct {
	Order            types.Int64  `tfsdk:"order"`
	Methods          types.Set    `tfsdk:"methods"`
	Paths            types.Set    `tfsdk:"paths"`
	Permissions      types.Set    `tfsdk:"permissions"`
	UserRoles        types.Set    `tfsdk:"user_roles"`
	MaxRetryAttempts types.Int64  `tfsdk:"max_retry_attempts"`
	RewritePath      types.Bool   `tfsdk:"rewrite_path"`
	Response         types.Object `tfsdk:"response"`
	AttachmentPaths  types.Object `tfsdk:"attachment_paths"`
	Upstreams        types.List   `tfsdk:"upstreams"`
}

type appServiceModel struct {
	DisplayName  types.String `tfsdk:"display_name"`
	Description  types.String `tfsdk:"description"`
	AuthSettings types.Object `tfsdk:"auth_settings"`
}

type appAuthModel struct {
	AuthenticationType    types.String `tfsdk:"authentication_type"`
	ClientID              types.String `tfsdk:"client_id"`
	ClientSecret          types.String `tfsdk:"client_secret"`
	AuthorizationEndpoint types.String `tfsdk:"authorization_endpoint"`
	TokenEndpoint         types.String `tfsdk:"token_endpoint"`
	ScopesSupported       types.Set    `tfsdk:"scopes_supported"`
	APIKeyHeader          types.String `tfsdk:"api_key_header"`
	CodeVerifier          types.String `tfsdk:"code_verifier"`
}

// ---- resource schema ----

func applicationNestedResourceAttributes() map[string]schema.Attribute {
	str := func(description string) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Description: description}
	}
	defaultedStr := func(description string, validators ...validator.String) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, Validators: validators, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, Description: description}
	}
	defaultedSet := func(description string) schema.SetAttribute {
		return schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType, PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()}, Description: description}
	}
	defaultedInt := func(description string) schema.Int64Attribute {
		return schema.Int64Attribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}, Description: description}
	}

	upstream := map[string]schema.Attribute{
		"endpoint":           schema.StringAttribute{Required: true},
		"responses_endpoint": schema.StringAttribute{Optional: true, Computed: true},
		"id":                 schema.StringAttribute{Optional: true, Computed: true},
		"extra_data":         schema.MapAttribute{ElementType: types.StringType, Optional: true, Computed: true},
		"key":                schema.StringAttribute{Optional: true, Sensitive: true, Description: "Upstream key. DIAL Core never returns it, so state keeps the configured value."},
		"secret_extra_data":  schema.MapAttribute{ElementType: types.StringType, Optional: true, Sensitive: true},
		"tier":               schema.Int64Attribute{Required: true},
		"weight":             schema.Int64Attribute{Required: true},
		"base_url":           schema.StringAttribute{Optional: true, Computed: true},
		"auth_type": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "How the key is sent to the upstream: `API_KEY`, `BEARER` or `CUSTOM_HEADER`. `CUSTOM_HEADER` requires `auth_header_name` and needs DIAL Core 0.50 or newer.",
			Validators: []validator.String{
				stringvalidator.OneOf("API_KEY", "BEARER", "CUSTOM_HEADER"),
				authHeaderNameRequiredForCustomHeader{},
			},
		},
		"auth_header_name": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Name of the HTTP header that carries the key. Required when `auth_type` is `CUSTOM_HEADER`, not allowed otherwise.",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1), authHeaderNameOnlyForCustomHeader{}},
		},
	}

	return map[string]schema.Attribute{
		"mcp": schema.SingleNestedAttribute{
			Optional:    true,
			Description: "MCP settings of the application. Removing it removes MCP from the application.",
			Attributes: map[string]schema.Attribute{
				"endpoint":        str("MCP endpoint of the application."),
				"transport":       defaultedStr("MCP transport: `HTTP` or `SSE`. DIAL Core defaults to `HTTP`.", stringvalidator.OneOf(toolsetTransports...)),
				"allowed_tools":   defaultedSet("Names of the MCP tools exposed. An empty set, which DIAL Core defaults to, exposes all of them."),
				"config_delivery": defaultedStr("How the configuration is delivered: `HEADER` or `META`. DIAL Core defaults to `META`.", stringvalidator.OneOf(applicationMcpDeliveries...)),
				"forward_per_request_key": schema.BoolAttribute{
					Optional: true, Computed: true,
					PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
					Description:   "Forward the per-request key to the MCP endpoint. DIAL Core defaults to true.",
				},
				"mcp_apps": schema.SingleNestedAttribute{
					Optional:    true,
					Description: "MCP apps settings.",
					Attributes: map[string]schema.Attribute{
						"domain_override": str("Domain override for MCP apps."),
					},
				},
			},
		},
		"routes": schema.MapNestedAttribute{
			Optional:      true,
			Computed:      true,
			PlanModifiers: []planmodifier.Map{mapplanmodifier.UseStateForUnknown()},
			Description:   "Routes of the application, keyed by route name. DIAL Core fills `order`, `methods`, `permissions`, `max_retry_attempts` and similar fields in when they are not set. Set `routes = {}` to remove all routes.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"order":              defaultedInt("Route matching order."),
					"methods":            defaultedSet("HTTP methods the route matches."),
					"paths":              defaultedSet("Path patterns the route matches."),
					"permissions":        defaultedSet("Permissions of the route."),
					"user_roles":         defaultedSet("Roles allowed to use the route."),
					"max_retry_attempts": defaultedInt("Maximum number of retry attempts."),
					"rewrite_path": schema.BoolAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
					},
					"response": schema.SingleNestedAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
						Description:   "Static response returned by the route instead of forwarding upstream.",
						Attributes: map[string]schema.Attribute{
							"body":   schema.StringAttribute{Optional: true, Computed: true},
							"status": schema.Int64Attribute{Optional: true, Computed: true},
						},
					},
					"attachment_paths": schema.SingleNestedAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
						Attributes: map[string]schema.Attribute{
							"request_body":  defaultedSet("Request body paths that carry attachments."),
							"response_body": defaultedSet("Response body paths that carry attachments."),
						},
					},
					"upstreams": schema.ListNestedAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
						NestedObject:  schema.NestedAttributeObject{Attributes: upstream},
					},
				},
			},
		},
		"external_services": schema.MapNestedAttribute{
			Optional:    true,
			Description: "External services the application can use, keyed by service name.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"display_name": str("Human-readable service name."),
					"description":  str("Service description."),
					"auth_settings": schema.SingleNestedAttribute{
						Optional:    true,
						Description: "Authentication settings of the service.",
						Attributes: map[string]schema.Attribute{
							"authentication_type":    str("Authentication type: `OAUTH`, `API_KEY`, `NONE` or `DIAL_NATIVE`. DIAL Core validates which other fields it accepts."),
							"client_id":              str("OAuth client ID."),
							"authorization_endpoint": str("OAuth authorization endpoint."),
							"token_endpoint":         str("OAuth token endpoint."),
							"scopes_supported":       schema.SetAttribute{Optional: true, ElementType: types.StringType, Description: "OAuth scopes."},
							"api_key_header":         str("Header that carries the API key."),
							"client_secret": schema.StringAttribute{
								Optional:    true,
								Sensitive:   true,
								Validators:  []validator.String{stringvalidator.LengthAtLeast(8)},
								Description: "OAuth client secret, at least 8 characters. DIAL Core never returns it, so state keeps the configured value; after import it is null.",
							},
							"code_verifier": schema.StringAttribute{
								Optional:    true,
								Sensitive:   true,
								Description: "PKCE code verifier. DIAL Core never returns it and rejects it for some authentication types.",
							},
						},
					},
				},
			},
		},
	}
}

// ---- data source schema ----

func applicationNestedDataSourceAttributes() map[string]dsschema.Attribute {
	str := func(description string) dsschema.StringAttribute {
		return dsschema.StringAttribute{Computed: true, Description: description}
	}
	strSet := func(description string) dsschema.SetAttribute {
		return dsschema.SetAttribute{Computed: true, ElementType: types.StringType, Description: description}
	}
	return map[string]dsschema.Attribute{
		"mcp": dsschema.SingleNestedAttribute{
			Computed:    true,
			Description: "MCP settings of the application.",
			Attributes: map[string]dsschema.Attribute{
				"endpoint":                str("MCP endpoint of the application."),
				"transport":               str("MCP transport."),
				"allowed_tools":           strSet("Names of the MCP tools exposed."),
				"config_delivery":         str("How the configuration is delivered."),
				"forward_per_request_key": dsschema.BoolAttribute{Computed: true},
				"mcp_apps": dsschema.SingleNestedAttribute{
					Computed:   true,
					Attributes: map[string]dsschema.Attribute{"domain_override": str("Domain override for MCP apps.")},
				},
			},
		},
		"routes": dsschema.MapNestedAttribute{
			Computed:    true,
			Description: "Routes of the application, keyed by route name.",
			NestedObject: dsschema.NestedAttributeObject{
				Attributes: map[string]dsschema.Attribute{
					"order":              dsschema.Int64Attribute{Computed: true},
					"methods":            strSet("HTTP methods the route matches."),
					"paths":              strSet("Path patterns the route matches."),
					"permissions":        strSet("Permissions of the route."),
					"user_roles":         strSet("Roles allowed to use the route."),
					"max_retry_attempts": dsschema.Int64Attribute{Computed: true},
					"rewrite_path":       dsschema.BoolAttribute{Computed: true},
					"response": dsschema.SingleNestedAttribute{
						Computed: true,
						Attributes: map[string]dsschema.Attribute{
							"body":   dsschema.StringAttribute{Computed: true},
							"status": dsschema.Int64Attribute{Computed: true},
						},
					},
					"attachment_paths": dsschema.SingleNestedAttribute{
						Computed: true,
						Attributes: map[string]dsschema.Attribute{
							"request_body":  strSet("Request body paths that carry attachments."),
							"response_body": strSet("Response body paths that carry attachments."),
						},
					},
					"upstreams": dsschema.ListNestedAttribute{
						Computed: true,
						NestedObject: dsschema.NestedAttributeObject{Attributes: map[string]dsschema.Attribute{
							"endpoint":           dsschema.StringAttribute{Computed: true},
							"responses_endpoint": dsschema.StringAttribute{Computed: true},
							"id":                 dsschema.StringAttribute{Computed: true},
							"extra_data":         dsschema.MapAttribute{ElementType: types.StringType, Computed: true},
							"key":                dsschema.StringAttribute{Computed: true, Sensitive: true},
							"secret_extra_data":  dsschema.MapAttribute{ElementType: types.StringType, Computed: true, Sensitive: true},
							"tier":               dsschema.Int64Attribute{Computed: true},
							"weight":             dsschema.Int64Attribute{Computed: true},
							"base_url":           dsschema.StringAttribute{Computed: true},
							"auth_type":          dsschema.StringAttribute{Computed: true},
							"auth_header_name":   dsschema.StringAttribute{Computed: true},
						}},
					},
				},
			},
		},
		"external_services": dsschema.MapNestedAttribute{
			Computed:    true,
			Description: "External services of the application, keyed by service name. Secrets are never returned by DIAL Core.",
			NestedObject: dsschema.NestedAttributeObject{
				Attributes: map[string]dsschema.Attribute{
					"display_name": str("Human-readable service name."),
					"description":  str("Service description."),
					"auth_settings": dsschema.SingleNestedAttribute{
						Computed: true,
						Attributes: map[string]dsschema.Attribute{
							"authentication_type":    str("Authentication type."),
							"client_id":              str("OAuth client ID."),
							"authorization_endpoint": str("OAuth authorization endpoint."),
							"token_endpoint":         str("OAuth token endpoint."),
							"scopes_supported":       strSet("OAuth scopes."),
							"api_key_header":         str("Header that carries the API key."),
							"client_secret":          dsschema.StringAttribute{Computed: true, Sensitive: true, Description: "Always null: DIAL Core never returns it."},
							"code_verifier":          dsschema.StringAttribute{Computed: true, Sensitive: true, Description: "Always null: DIAL Core never returns it."},
						},
					},
				},
			},
		},
	}
}

// ---- mcp mapping ----

func mcpFromObject(ctx context.Context, value types.Object) (*client.ApplicationMcp, error) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	var model appMcpModel
	if diags := value.As(ctx, &model, basetypes.ObjectAsOptions{}); diags.HasError() {
		return nil, fmt.Errorf("invalid mcp: %s", diags.Errors()[0].Summary())
	}
	tools, err := stringSetToAPI(ctx, model.AllowedTools)
	if err != nil {
		return nil, fmt.Errorf("invalid mcp.allowed_tools: %w", err)
	}
	mcp := &client.ApplicationMcp{
		Endpoint:             stringPointer(model.Endpoint),
		Transport:            stringPointer(model.Transport),
		AllowedTools:         tools,
		ConfigDelivery:       stringPointer(model.ConfigDelivery),
		ForwardPerRequestKey: boolPointer(model.ForwardPerRequestKey),
	}
	if !model.McpApps.IsNull() && !model.McpApps.IsUnknown() {
		var apps appMcpAppsModel
		if diags := model.McpApps.As(ctx, &apps, basetypes.ObjectAsOptions{}); diags.HasError() {
			return nil, fmt.Errorf("invalid mcp.mcp_apps: %s", diags.Errors()[0].Summary())
		}
		mcp.McpApps = &client.ApplicationMcpApps{DomainOverride: stringPointer(apps.DomainOverride)}
	}
	return mcp, nil
}

func mcpToObject(ctx context.Context, api *client.ApplicationMcp) (types.Object, error) {
	null := types.ObjectNull(appMcpAttrTypes)
	if api == nil {
		return null, nil
	}
	tools, err := stringSetFromAPI(ctx, api.AllowedTools)
	if err != nil {
		return null, fmt.Errorf("failed to set mcp.allowed_tools: %w", err)
	}
	apps := types.ObjectNull(appMcpAppsAttrTypes)
	if api.McpApps != nil {
		obj, diags := types.ObjectValue(appMcpAppsAttrTypes, map[string]attr.Value{"domain_override": stringValue(api.McpApps.DomainOverride)})
		if diags.HasError() {
			return null, fmt.Errorf("failed to build mcp.mcp_apps: %s", diags.Errors()[0].Summary())
		}
		apps = obj
	}
	obj, diags := types.ObjectValue(appMcpAttrTypes, map[string]attr.Value{
		"endpoint":                stringValue(api.Endpoint),
		"transport":               stringValue(api.Transport),
		"allowed_tools":           tools,
		"config_delivery":         stringValue(api.ConfigDelivery),
		"forward_per_request_key": boolValue(api.ForwardPerRequestKey),
		"mcp_apps":                apps,
	})
	if diags.HasError() {
		return null, fmt.Errorf("failed to build mcp: %s", diags.Errors()[0].Summary())
	}
	return obj, nil
}

// ---- routes mapping ----

func stringSliceFromSet(ctx context.Context, name string, value types.Set) ([]string, error) {
	values, err := stringSetToAPI(ctx, value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", name, err)
	}
	if values == nil {
		return nil, nil
	}
	return *values, nil
}

func setOrNull(ctx context.Context, name string, values []string) (types.Set, error) {
	if values == nil {
		return types.SetNull(types.StringType), nil
	}
	set, diags := types.SetValueFrom(ctx, types.StringType, values)
	if diags.HasError() {
		return types.SetNull(types.StringType), fmt.Errorf("failed to set %s: %s", name, diags.Errors()[0].Summary())
	}
	return set, nil
}

func routesFromMap(ctx context.Context, value types.Map) (map[string]client.Route, error) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	var models map[string]appRouteModel
	if diags := value.ElementsAs(ctx, &models, false); diags.HasError() {
		return nil, fmt.Errorf("invalid routes: %s", diags.Errors()[0].Summary())
	}
	routes := make(map[string]client.Route, len(models))
	for name, model := range models {
		route, err := routeFromAppModel(ctx, model)
		if err != nil {
			return nil, fmt.Errorf("invalid routes[%q]: %w", name, err)
		}
		routes[name] = route
	}
	return routes, nil
}

func routeFromAppModel(ctx context.Context, model appRouteModel) (client.Route, error) {
	route := client.Route{
		Order:            int64Pointer(model.Order),
		MaxRetryAttempts: int64Pointer(model.MaxRetryAttempts),
		RewritePath:      boolPointer(model.RewritePath),
	}
	var err error
	if route.Methods, err = stringSliceFromSet(ctx, "methods", model.Methods); err != nil {
		return route, err
	}
	if route.Paths, err = stringSliceFromSet(ctx, "paths", model.Paths); err != nil {
		return route, err
	}
	if route.Permissions, err = stringSliceFromSet(ctx, "permissions", model.Permissions); err != nil {
		return route, err
	}
	if route.UserRoles, err = stringSliceFromSet(ctx, "user_roles", model.UserRoles); err != nil {
		return route, err
	}

	if !model.Response.IsNull() && !model.Response.IsUnknown() {
		var response routeResponseModel
		if diags := model.Response.As(ctx, &response, basetypes.ObjectAsOptions{}); diags.HasError() {
			return route, fmt.Errorf("invalid response: %s", diags.Errors()[0].Summary())
		}
		route.Response = &client.RouteResponse{Body: stringPointer(response.Body), Status: int64Pointer(response.Status)}
	}

	if !model.AttachmentPaths.IsNull() && !model.AttachmentPaths.IsUnknown() {
		var paths routeAttachmentPathsModel
		if diags := model.AttachmentPaths.As(ctx, &paths, basetypes.ObjectAsOptions{}); diags.HasError() {
			return route, fmt.Errorf("invalid attachment_paths: %s", diags.Errors()[0].Summary())
		}
		attachments := &client.RouteAttachmentPath{}
		if attachments.RequestBody, err = stringSliceFromSet(ctx, "attachment_paths.request_body", paths.RequestBody); err != nil {
			return route, err
		}
		if attachments.ResponseBody, err = stringSliceFromSet(ctx, "attachment_paths.response_body", paths.ResponseBody); err != nil {
			return route, err
		}
		route.AttachmentPaths = attachments
	}

	if !model.Upstreams.IsNull() && !model.Upstreams.IsUnknown() {
		var upstreams []modelUpstreamModel
		if diags := model.Upstreams.ElementsAs(ctx, &upstreams, false); diags.HasError() {
			return route, fmt.Errorf("invalid upstreams: %s", diags.Errors()[0].Summary())
		}
		for _, u := range upstreams {
			upstream := client.ModelUpstream{
				Endpoint:          stringPointer(u.Endpoint),
				ResponsesEndpoint: stringPointer(u.ResponsesEndpoint),
				ID:                stringPointer(u.ID),
				Key:               stringPointer(u.Key),
				BaseURL:           stringPointer(u.BaseURL),
				AuthType:          stringPointer(u.AuthType),
				AuthHeaderName:    stringPointer(u.AuthHeaderName),
			}
			if upstream.ExtraData, err = optionalStringMap(ctx, u.ExtraData, "extra_data"); err != nil {
				return route, err
			}
			if upstream.SecretExtraData, err = optionalStringMap(ctx, u.SecretExtraData, "secret_extra_data"); err != nil {
				return route, err
			}
			if !u.Tier.IsNull() && !u.Tier.IsUnknown() {
				tier := int(u.Tier.ValueInt64())
				upstream.Tier = &tier
			}
			if !u.Weight.IsNull() && !u.Weight.IsUnknown() {
				weight := int(u.Weight.ValueInt64())
				upstream.Weight = &weight
			}
			route.Upstreams = append(route.Upstreams, upstream)
		}
	}
	return route, nil
}

// routesToMap maps the API routes into state. The prior map supplies each
// upstream's key and secret_extra_data, which Core never returns.
func routesToMap(ctx context.Context, prior types.Map, api map[string]client.Route) (types.Map, error) {
	objType := types.ObjectType{AttrTypes: appRouteAttrTypes}
	elements := make(map[string]attr.Value, len(api))
	for name, route := range api {
		var priorUpstreams types.List
		if obj, ok := prior.Elements()[name].(types.Object); ok {
			if list, ok := obj.Attributes()["upstreams"].(types.List); ok {
				priorUpstreams = list
			}
		}
		obj, err := routeToObject(ctx, priorUpstreams, route)
		if err != nil {
			return types.MapNull(objType), fmt.Errorf("routes[%q]: %w", name, err)
		}
		elements[name] = obj
	}
	value, diags := types.MapValue(objType, elements)
	if diags.HasError() {
		return types.MapNull(objType), fmt.Errorf("failed to build routes: %s", diags.Errors()[0].Summary())
	}
	return value, nil
}

func routeToObject(ctx context.Context, priorUpstreams types.List, route client.Route) (types.Object, error) {
	null := types.ObjectNull(appRouteAttrTypes)
	values := map[string]attr.Value{
		"order":              int64Value(route.Order),
		"max_retry_attempts": int64Value(route.MaxRetryAttempts),
		"rewrite_path":       boolValue(route.RewritePath),
	}
	for name, list := range map[string][]string{
		"methods": route.Methods, "paths": route.Paths, "permissions": route.Permissions, "user_roles": route.UserRoles,
	} {
		set, err := setOrNull(ctx, name, list)
		if err != nil {
			return null, err
		}
		values[name] = set
	}

	values["response"] = types.ObjectNull(appRouteResponseAttrTypes)
	if route.Response != nil {
		obj, diags := types.ObjectValue(appRouteResponseAttrTypes, map[string]attr.Value{
			"body": stringValue(route.Response.Body), "status": int64Value(route.Response.Status),
		})
		if diags.HasError() {
			return null, fmt.Errorf("failed to build response: %s", diags.Errors()[0].Summary())
		}
		values["response"] = obj
	}

	values["attachment_paths"] = types.ObjectNull(appRouteAttachmentAttrTypes)
	if route.AttachmentPaths != nil {
		requestBody, err := setOrNull(ctx, "attachment_paths.request_body", route.AttachmentPaths.RequestBody)
		if err != nil {
			return null, err
		}
		responseBody, err := setOrNull(ctx, "attachment_paths.response_body", route.AttachmentPaths.ResponseBody)
		if err != nil {
			return null, err
		}
		obj, diags := types.ObjectValue(appRouteAttachmentAttrTypes, map[string]attr.Value{
			"request_body": requestBody, "response_body": responseBody,
		})
		if diags.HasError() {
			return null, fmt.Errorf("failed to build attachment_paths: %s", diags.Errors()[0].Summary())
		}
		values["attachment_paths"] = obj
	}

	upstreamType := types.ObjectType{AttrTypes: appRouteUpstreamAttrTypes}
	values["upstreams"] = types.ListNull(upstreamType)
	if route.Upstreams != nil {
		priorWriteOnly := upstreamWriteOnlyValues(priorUpstreams)
		items := make([]attr.Value, 0, len(route.Upstreams))
		for i, u := range route.Upstreams {
			extraData := types.MapNull(types.StringType)
			if len(u.ExtraData) > 0 {
				m, diags := types.MapValueFrom(ctx, types.StringType, u.ExtraData)
				if diags.HasError() {
					return null, fmt.Errorf("failed to set upstreams.extra_data: %s", diags.Errors()[0].Summary())
				}
				extraData = m
			}
			key, secret := priorWriteOnly.at(ctx, i, u.Key, u.SecretExtraData)
			tier, weight := types.Int64Null(), types.Int64Null()
			if u.Tier != nil {
				tier = types.Int64Value(int64(*u.Tier))
			}
			if u.Weight != nil {
				weight = types.Int64Value(int64(*u.Weight))
			}
			obj, diags := types.ObjectValue(appRouteUpstreamAttrTypes, map[string]attr.Value{
				"endpoint":           stringValue(u.Endpoint),
				"responses_endpoint": stringValue(u.ResponsesEndpoint),
				"id":                 stringValue(u.ID),
				"extra_data":         extraData,
				"key":                key,
				"secret_extra_data":  secret,
				"tier":               tier,
				"weight":             weight,
				"base_url":           stringValue(u.BaseURL),
				"auth_type":          stringValue(u.AuthType),
				"auth_header_name":   stringValue(u.AuthHeaderName),
			})
			if diags.HasError() {
				return null, fmt.Errorf("failed to build upstream: %s", diags.Errors()[0].Summary())
			}
			items = append(items, obj)
		}
		list, diags := types.ListValue(upstreamType, items)
		if diags.HasError() {
			return null, fmt.Errorf("failed to build upstreams: %s", diags.Errors()[0].Summary())
		}
		values["upstreams"] = list
	}

	obj, diags := types.ObjectValue(appRouteAttrTypes, values)
	if diags.HasError() {
		return null, fmt.Errorf("failed to build route: %s", diags.Errors()[0].Summary())
	}
	return obj, nil
}

// ---- external services mapping ----

func externalServicesFromMap(ctx context.Context, value types.Map) (map[string]client.ApplicationExternalService, error) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	var models map[string]appServiceModel
	if diags := value.ElementsAs(ctx, &models, false); diags.HasError() {
		return nil, fmt.Errorf("invalid external_services: %s", diags.Errors()[0].Summary())
	}
	services := make(map[string]client.ApplicationExternalService, len(models))
	for name, model := range models {
		service := client.ApplicationExternalService{
			DisplayName: stringPointer(model.DisplayName),
			Description: stringPointer(model.Description),
		}
		if !model.AuthSettings.IsNull() && !model.AuthSettings.IsUnknown() {
			var auth appAuthModel
			if diags := model.AuthSettings.As(ctx, &auth, basetypes.ObjectAsOptions{}); diags.HasError() {
				return nil, fmt.Errorf("invalid external_services[%q].auth_settings: %s", name, diags.Errors()[0].Summary())
			}
			scopes, err := stringSetToAPI(ctx, auth.ScopesSupported)
			if err != nil {
				return nil, fmt.Errorf("invalid external_services[%q].auth_settings.scopes_supported: %w", name, err)
			}
			service.AuthSettings = &client.ApplicationExternalServiceAuth{
				AuthenticationType:    stringPointer(auth.AuthenticationType),
				ClientID:              stringPointer(auth.ClientID),
				ClientSecret:          stringPointer(auth.ClientSecret),
				AuthorizationEndpoint: stringPointer(auth.AuthorizationEndpoint),
				TokenEndpoint:         stringPointer(auth.TokenEndpoint),
				ScopesSupported:       scopes,
				APIKeyHeader:          stringPointer(auth.APIKeyHeader),
				CodeVerifier:          stringPointer(auth.CodeVerifier),
			}
		}
		services[name] = service
	}
	return services, nil
}

// externalServicesToMap maps the API services into state. Core never returns
// client_secret or code_verifier, so they come from the prior map (null when
// there is none, as after import or in a data source). A configured empty map
// stays empty; otherwise no services read back as null.
func externalServicesToMap(ctx context.Context, prior types.Map, api map[string]client.ApplicationExternalService) (types.Map, error) {
	objType := types.ObjectType{AttrTypes: appServiceAttrTypes}
	if len(api) == 0 {
		if !prior.IsNull() && !prior.IsUnknown() && len(prior.Elements()) == 0 {
			return prior, nil
		}
		return types.MapNull(objType), nil
	}
	elements := make(map[string]attr.Value, len(api))
	for name, service := range api {
		secret, verifier := types.StringNull(), types.StringNull()
		if obj, ok := prior.Elements()[name].(types.Object); ok {
			if auth, ok := obj.Attributes()["auth_settings"].(types.Object); ok && !auth.IsNull() && !auth.IsUnknown() {
				if v, ok := auth.Attributes()["client_secret"].(types.String); ok && !v.IsUnknown() {
					secret = v
				}
				if v, ok := auth.Attributes()["code_verifier"].(types.String); ok && !v.IsUnknown() {
					verifier = v
				}
			}
		}

		auth := types.ObjectNull(appAuthAttrTypes)
		if service.AuthSettings != nil {
			settings := service.AuthSettings
			scopes, err := stringSetFromAPI(ctx, settings.ScopesSupported)
			if err != nil {
				return types.MapNull(objType), fmt.Errorf("failed to set external_services[%q].scopes_supported: %w", name, err)
			}
			obj, diags := types.ObjectValue(appAuthAttrTypes, map[string]attr.Value{
				"authentication_type":    stringValue(settings.AuthenticationType),
				"client_id":              stringValue(settings.ClientID),
				"client_secret":          secret,
				"authorization_endpoint": stringValue(settings.AuthorizationEndpoint),
				"token_endpoint":         stringValue(settings.TokenEndpoint),
				"scopes_supported":       scopes,
				"api_key_header":         stringValue(settings.APIKeyHeader),
				"code_verifier":          verifier,
			})
			if diags.HasError() {
				return types.MapNull(objType), fmt.Errorf("failed to build external_services[%q].auth_settings: %s", name, diags.Errors()[0].Summary())
			}
			auth = obj
		}
		obj, diags := types.ObjectValue(appServiceAttrTypes, map[string]attr.Value{
			"display_name":  stringValue(service.DisplayName),
			"description":   stringValue(service.Description),
			"auth_settings": auth,
		})
		if diags.HasError() {
			return types.MapNull(objType), fmt.Errorf("failed to build external_services[%q]: %s", name, diags.Errors()[0].Summary())
		}
		elements[name] = obj
	}
	value, diags := types.MapValue(objType, elements)
	if diags.HasError() {
		return types.MapNull(objType), fmt.Errorf("failed to build external_services: %s", diags.Errors()[0].Summary())
	}
	return value, nil
}
