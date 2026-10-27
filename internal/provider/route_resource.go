package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type routeResource struct {
	client *client.Client
}

type routeResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Bucket  types.String `tfsdk:"bucket"`
	Path    types.String `tfsdk:"path"`
	Name    types.String `tfsdk:"name"`
	APIName types.String `tfsdk:"api_name"`

	Order            types.Int64 `tfsdk:"order"`
	Methods          types.Set   `tfsdk:"methods"`
	Paths            types.Set   `tfsdk:"paths"`
	Permissions      types.Set   `tfsdk:"permissions"`
	UserRoles        types.Set   `tfsdk:"user_roles"`
	MaxRetryAttempts types.Int64 `tfsdk:"max_retry_attempts"`
	RewritePath      types.Bool  `tfsdk:"rewrite_path"`

	Response        types.Object `tfsdk:"response"`
	AttachmentPaths types.Object `tfsdk:"attachment_paths"`
	Upstreams       types.List   `tfsdk:"upstreams"`

	Status types.String `tfsdk:"status"`
}

type routeResponseModel struct {
	Body   types.String `tfsdk:"body"`
	Status types.Int64  `tfsdk:"status"`
}

type routeAttachmentPathsModel struct {
	RequestBody  types.Set `tfsdk:"request_body"`
	ResponseBody types.Set `tfsdk:"response_body"`
}

var routeResponseAttrTypes = map[string]attr.Type{
	"body":   types.StringType,
	"status": types.Int64Type,
}

var routeAttachmentPathsAttrTypes = map[string]attr.Type{
	"request_body":  types.SetType{ElemType: types.StringType},
	"response_body": types.SetType{ElemType: types.StringType},
}

var routeUpstreamAttrTypes = map[string]attr.Type{
	"endpoint":           types.StringType,
	"responses_endpoint": types.StringType,
	"id":                 types.StringType,
	"extra_data":         types.MapType{ElemType: types.StringType},
	"key":                types.StringType,
	"secret_extra_data":  types.MapType{ElemType: types.StringType},
	"tier":               types.Int64Type,
	"weight":             types.Int64Type,
}

func (r *routeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_route"
}

func (r *routeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a routing rule in DIAL Core (`PUT /v1/routes/{bucket}/{path}`): methods, paths, permissions and upstreams or a static response.",
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
				PlanModifiers: []planmodifier.String{
					bucketDefaultsToPlatform{},
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Target DIAL bucket. Defaults to \"" + defaultBucket + "\" when not set.",
			},
			"path": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					pathDefaultsToName{},
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Route path inside the target bucket. Defaults to name when not set.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Route name stored in the DIAL API request body.",
			},
			"api_name": schema.StringAttribute{
				Computed:    true,
				Description: "Canonical route name returned by the DIAL API.",
			},
			"order": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Route matching order.",
			},
			"methods": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "HTTP methods the route matches.",
			},
			"paths": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Path patterns the route matches.",
			},
			"permissions": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Resource access types required to use the route, one of READ, WRITE, SHARE.",
			},
			"user_roles": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Roles allowed to use the route.",
			},
			"max_retry_attempts": schema.Int64Attribute{
				Optional: true,
				Computed: true,
			},
			"rewrite_path": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"response": schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"body":   schema.StringAttribute{Optional: true, Computed: true},
					"status": schema.Int64Attribute{Optional: true, Computed: true},
				},
				Description: "Static response returned by the route instead of forwarding upstream.",
			},
			"attachment_paths": schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"request_body": schema.SetAttribute{
						Optional:    true,
						Computed:    true,
						ElementType: types.StringType,
					},
					"response_body": schema.SetAttribute{
						Optional:    true,
						Computed:    true,
						ElementType: types.StringType,
					},
				},
			},
			"upstreams": schema.ListNestedAttribute{
				Optional: true,
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"endpoint":           schema.StringAttribute{Required: true},
						"responses_endpoint": schema.StringAttribute{Optional: true, Computed: true},
						"id":                 schema.StringAttribute{Optional: true, Computed: true},
						"extra_data":         schema.MapAttribute{ElementType: types.StringType, Optional: true, Computed: true},
						"key":                schema.StringAttribute{Optional: true, Sensitive: true},
						"secret_extra_data":  schema.MapAttribute{ElementType: types.StringType, Optional: true, Sensitive: true},
						"tier":               schema.Int64Attribute{Required: true},
						"weight":             schema.Int64Attribute{Required: true},
					},
				},
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Route status returned by the DIAL API.",
			},
		},
	}
}

func (r *routeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func routeFromPlan(ctx context.Context, plan routeResourceModel) (*client.Route, error) {
	result := &client.Route{
		Name: plan.Name.ValueString(),
	}

	if !plan.Order.IsNull() && !plan.Order.IsUnknown() {
		result.Order = int64Pointer(plan.Order)
	}
	if !plan.MaxRetryAttempts.IsNull() && !plan.MaxRetryAttempts.IsUnknown() {
		result.MaxRetryAttempts = int64Pointer(plan.MaxRetryAttempts)
	}
	if !plan.RewritePath.IsNull() && !plan.RewritePath.IsUnknown() {
		result.RewritePath = boolPointer(plan.RewritePath)
	}

	if !plan.Methods.IsNull() && !plan.Methods.IsUnknown() {
		var methods []string
		diags := plan.Methods.ElementsAs(ctx, &methods, false)
		if diags.HasError() {
			return nil, fmt.Errorf("invalid methods")
		}
		result.Methods = methods
	}

	if !plan.Paths.IsNull() && !plan.Paths.IsUnknown() {
		var paths []string
		diags := plan.Paths.ElementsAs(ctx, &paths, false)
		if diags.HasError() {
			return nil, fmt.Errorf("invalid paths")
		}
		result.Paths = paths
	}

	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		var permissions []string
		diags := plan.Permissions.ElementsAs(ctx, &permissions, false)
		if diags.HasError() {
			return nil, fmt.Errorf("invalid permissions")
		}
		result.Permissions = permissions
	}

	if !plan.UserRoles.IsNull() && !plan.UserRoles.IsUnknown() {
		var userRoles []string
		diags := plan.UserRoles.ElementsAs(ctx, &userRoles, false)
		if diags.HasError() {
			return nil, fmt.Errorf("invalid user_roles")
		}
		result.UserRoles = userRoles
	}

	if !plan.Response.IsNull() && !plan.Response.IsUnknown() {
		var resp routeResponseModel
		diags := plan.Response.As(ctx, &resp, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("invalid response")
		}
		result.Response = &client.RouteResponse{
			Body:   stringPointer(resp.Body),
			Status: int64Pointer(resp.Status),
		}
	}

	if !plan.AttachmentPaths.IsNull() && !plan.AttachmentPaths.IsUnknown() {
		var ap routeAttachmentPathsModel
		diags := plan.AttachmentPaths.As(ctx, &ap, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("invalid attachment_paths")
		}
		attachmentPaths := &client.RouteAttachmentPath{}
		if !ap.RequestBody.IsNull() && !ap.RequestBody.IsUnknown() {
			var requestBody []string
			diags2 := ap.RequestBody.ElementsAs(ctx, &requestBody, false)
			if diags2.HasError() {
				return nil, fmt.Errorf("invalid attachment_paths.request_body")
			}
			attachmentPaths.RequestBody = requestBody
		}
		if !ap.ResponseBody.IsNull() && !ap.ResponseBody.IsUnknown() {
			var responseBody []string
			diags2 := ap.ResponseBody.ElementsAs(ctx, &responseBody, false)
			if diags2.HasError() {
				return nil, fmt.Errorf("invalid attachment_paths.response_body")
			}
			attachmentPaths.ResponseBody = responseBody
		}
		result.AttachmentPaths = attachmentPaths
	}

	if !plan.Upstreams.IsNull() && !plan.Upstreams.IsUnknown() {
		var upstreams []modelUpstreamModel
		diags := plan.Upstreams.ElementsAs(ctx, &upstreams, false)
		if diags.HasError() {
			return nil, fmt.Errorf("invalid upstreams")
		}
		for _, u := range upstreams {
			var responsesEndpoint *string
			if !u.ResponsesEndpoint.IsNull() && !u.ResponsesEndpoint.IsUnknown() {
				v := u.ResponsesEndpoint.ValueString()
				responsesEndpoint = &v
			}
			var id *string
			if !u.ID.IsNull() && !u.ID.IsUnknown() {
				v := u.ID.ValueString()
				id = &v
			}
			var extraData map[string]string
			if !u.ExtraData.IsNull() && !u.ExtraData.IsUnknown() {
				m := make(map[string]string)
				diags := u.ExtraData.ElementsAs(ctx, &m, false)
				if !diags.HasError() {
					extraData = m
				}
			}
			var key *string
			if !u.Key.IsNull() && !u.Key.IsUnknown() {
				v := u.Key.ValueString()
				key = &v
			}
			var secretExtraData map[string]string
			if !u.SecretExtraData.IsNull() && !u.SecretExtraData.IsUnknown() {
				m := make(map[string]string)
				diags2 := u.SecretExtraData.ElementsAs(ctx, &m, false)
				if !diags2.HasError() {
					secretExtraData = m
				}
			}
			var tier *int
			if !u.Tier.IsNull() && !u.Tier.IsUnknown() {
				v := int(u.Tier.ValueInt64())
				tier = &v
			}

			var weight *int
			if !u.Weight.IsNull() && !u.Weight.IsUnknown() {
				v := int(u.Weight.ValueInt64())
				weight = &v
			}
			result.Upstreams = append(result.Upstreams, client.ModelUpstream{
				Endpoint:          stringPointer(u.Endpoint),
				ResponsesEndpoint: responsesEndpoint,
				ID:                id,
				ExtraData:         extraData,
				Key:               key,
				SecretExtraData:   secretExtraData,
				Tier:              tier,
				Weight:            weight,
			})
		}
	}

	return result, nil
}

func setRouteState(ctx context.Context, state *routeResourceModel, route *client.Route) error {
	state.APIName = types.StringValue(route.Name)
	if state.Name.IsNull() || state.Name.IsUnknown() {
		state.Name = types.StringValue(route.Name)
	}
	state.MaxRetryAttempts = int64Value(route.MaxRetryAttempts)
	state.Order = int64Value(route.Order)
	state.RewritePath = boolValue(route.RewritePath)
	state.Status = types.StringValue(route.Status)

	methods, diags := types.SetValueFrom(ctx, types.StringType, route.Methods)
	if diags.HasError() {
		return fmt.Errorf("failed to set methods: %s", diags.Errors()[0].Summary())
	}
	state.Methods = methods

	paths, diags := types.SetValueFrom(ctx, types.StringType, route.Paths)
	if diags.HasError() {
		return fmt.Errorf("failed to set paths: %s", diags.Errors()[0].Summary())
	}
	state.Paths = paths

	permissions, diags := types.SetValueFrom(ctx, types.StringType, route.Permissions)
	if diags.HasError() {
		return fmt.Errorf("failed to set permissions: %s", diags.Errors()[0].Summary())
	}
	state.Permissions = permissions

	userRoles, diags := types.SetValueFrom(ctx, types.StringType, route.UserRoles)
	if diags.HasError() {
		return fmt.Errorf("failed to set user_roles: %s", diags.Errors()[0].Summary())
	}
	state.UserRoles = userRoles

	if route.Response != nil {
		obj, diags := types.ObjectValue(routeResponseAttrTypes, map[string]attr.Value{
			"body":   stringValue(route.Response.Body),
			"status": int64Value(route.Response.Status),
		})
		if diags.HasError() {
			return fmt.Errorf("failed to build response object")
		}
		state.Response = obj
	} else {
		state.Response = types.ObjectNull(routeResponseAttrTypes)
	}

	if route.AttachmentPaths != nil {
		requestBody, diags := types.SetValueFrom(ctx, types.StringType, route.AttachmentPaths.RequestBody)
		if diags.HasError() {
			return fmt.Errorf("failed to set attachment_paths.request_body")
		}
		responseBody, diags := types.SetValueFrom(ctx, types.StringType, route.AttachmentPaths.ResponseBody)
		if diags.HasError() {
			return fmt.Errorf("failed to set attachment_paths.response_body")
		}
		obj, diags := types.ObjectValue(routeAttachmentPathsAttrTypes, map[string]attr.Value{
			"request_body":  requestBody,
			"response_body": responseBody,
		})
		if diags.HasError() {
			return fmt.Errorf("failed to build attachment_paths object")
		}
		state.AttachmentPaths = obj
	} else {
		state.AttachmentPaths = types.ObjectNull(routeAttachmentPathsAttrTypes)
	}

	if len(route.Upstreams) > 0 {
		var upstreamObjs []attr.Value
		priorWriteOnly := upstreamWriteOnlyValues(state.Upstreams)
		for i, u := range route.Upstreams {
			responsesEndpointVal := types.StringNull()
			if u.ResponsesEndpoint != nil {
				responsesEndpointVal = types.StringValue(*u.ResponsesEndpoint)
			}
			idVal := types.StringNull()
			if u.ID != nil {
				idVal = types.StringValue(*u.ID)
			}
			extraDataVal := types.MapNull(types.StringType)
			if len(u.ExtraData) > 0 {
				m, mapDiags := types.MapValueFrom(ctx, types.StringType, u.ExtraData)
				if !mapDiags.HasError() {
					extraDataVal = m
				}
			}
			keyVal, secretMap := priorWriteOnly.at(ctx, i, u.Key, u.SecretExtraData)
			tierVal := types.Int64Null()
			if u.Tier != nil {
				tierVal = types.Int64Value(int64(*u.Tier))
			}

			weightVal := types.Int64Null()
			if u.Weight != nil {
				weightVal = types.Int64Value(int64(*u.Weight))
			}
			uValues := map[string]attr.Value{
				"endpoint":           stringValue(u.Endpoint),
				"responses_endpoint": responsesEndpointVal,
				"id":                 idVal,
				"extra_data":         extraDataVal,
				"key":                keyVal,
				"secret_extra_data":  secretMap,
				"tier":               tierVal,
				"weight":             weightVal,
			}
			uObj, diags := types.ObjectValue(routeUpstreamAttrTypes, uValues)
			if !diags.HasError() {
				upstreamObjs = append(upstreamObjs, uObj)
			}
		}
		list, diags := types.ListValue(types.ObjectType{AttrTypes: routeUpstreamAttrTypes}, upstreamObjs)
		if diags.HasError() {
			return fmt.Errorf("failed to build upstreams list")
		}
		state.Upstreams = list
	} else {
		state.Upstreams = types.ListNull(types.ObjectType{AttrTypes: routeUpstreamAttrTypes})
	}

	return nil
}

func (r *routeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan routeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	route, err := routeFromPlan(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid route configuration", err.Error())
		return
	}

	if err := r.client.SaveRoute(ctx, plan.Bucket.ValueString(), plan.Path.ValueString(), route); err != nil {
		resp.Diagnostics.AddError("Failed to create route", err.Error())
		return
	}

	saved, err := r.client.GetRoute(ctx, plan.Bucket.ValueString(), plan.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read created route", err.Error())
		return
	}
	if saved == nil {
		resp.Diagnostics.AddError("Route not found after creation", "The API returned success but the route could not be read.")
		return
	}

	plan.ID = types.StringValue(plan.Bucket.ValueString() + "/" + plan.Path.ValueString())
	if err := setRouteState(ctx, &plan, saved); err != nil {
		resp.Diagnostics.AddError("Failed to store route state", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *routeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state routeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	route, err := r.client.GetRoute(ctx, state.Bucket.ValueString(), state.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read route", err.Error())
		return
	}
	if route == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	if err := setRouteState(ctx, &state, route); err != nil {
		resp.Diagnostics.AddError("Failed to store route state", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *routeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan routeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	route, err := routeFromPlan(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid route configuration", err.Error())
		return
	}

	if err := r.client.SaveRoute(ctx, plan.Bucket.ValueString(), plan.Path.ValueString(), route); err != nil {
		resp.Diagnostics.AddError("Failed to update route", err.Error())
		return
	}

	saved, err := r.client.GetRoute(ctx, plan.Bucket.ValueString(), plan.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read updated route", err.Error())
		return
	}
	if saved == nil {
		resp.Diagnostics.AddError("Route not found after update", "The API returned success but the route could not be read.")
		return
	}

	plan.ID = types.StringValue(plan.Bucket.ValueString() + "/" + plan.Path.ValueString())
	if err := setRouteState(ctx, &plan, saved); err != nil {
		resp.Diagnostics.AddError("Failed to store route state", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *routeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state routeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteRoute(ctx, state.Bucket.ValueString(), state.Path.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete route", err.Error())
	}
}

func (r *routeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Use terraform import aidial_route.example <bucket>/<path>.",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bucket"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), parts[1])...)
}
