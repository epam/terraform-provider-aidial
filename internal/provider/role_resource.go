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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type roleResource struct {
	client *client.Client
}

type roleCostLimitModel struct {
	Day    types.Float64 `tfsdk:"day"`
	Minute types.Float64 `tfsdk:"minute"`
	Month  types.Float64 `tfsdk:"month"`
	Week   types.Float64 `tfsdk:"week"`
}

type roleResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Bucket      types.String `tfsdk:"bucket"`
	Path        types.String `tfsdk:"path"`
	Name        types.String `tfsdk:"name"`
	APIName     types.String `tfsdk:"api_name"`
	Description types.String `tfsdk:"description"`
	DisplayName types.String `tfsdk:"display_name"`
	Limits      types.Map    `tfsdk:"limits"`
	Share       types.Map    `tfsdk:"share"`
	CostLimit   types.Object `tfsdk:"cost_limit"`
	Topics      types.Set    `tfsdk:"topics"`
	GrantedKeys types.Set    `tfsdk:"granted_keys"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
	Status      types.String `tfsdk:"status"`
}

func (r *roleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *roleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a role in DIAL Core (`PUT /v1/roles/{bucket}/{path}`): limits, sharing, cost limits and topics.",
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
				Description: "Role path inside the target bucket. Defaults to name when not set.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Role name stored in the DIAL API request body.",
			},
			"api_name": schema.StringAttribute{
				Computed:    true,
				Description: "Canonical role name returned by the DIAL API.",
			},
			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Role description.",
			},
			"display_name": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Human-readable role name.",
			},
			"limits": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Map of limit names to JSON objects encoded as strings.",
			},
			"share": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Map of share names to JSON objects encoded as strings.",
			},
			"cost_limit": schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"day": schema.Float64Attribute{
						Optional:    true,
						Computed:    true,
						Description: "Cost limit per day.",
					},
					"minute": schema.Float64Attribute{
						Optional:    true,
						Computed:    true,
						Description: "Cost limit per minute.",
					},
					"month": schema.Float64Attribute{
						Optional:    true,
						Computed:    true,
						Description: "Cost limit per month.",
					},
					"week": schema.Float64Attribute{
						Optional:    true,
						Computed:    true,
						Description: "Cost limit per week.",
					},
				},
				Description: "Cost limits for this role.",
			},
			"topics": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
				Description: "Topics associated with the role.",
			},
			"granted_keys": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "API keys granted this role, managed outside this resource.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Role creation timestamp.",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Role last-updated timestamp.",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Role status returned by the DIAL API.",
			},
		},
	}
}

func (r *roleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func roleFromPlan(ctx context.Context, plan roleResourceModel) (*client.Role, error) {
	limits, limitDiags := limitsFromState(plan.Limits)
	if limitDiags.HasError() {
		return nil, fmt.Errorf("invalid limits")
	}

	share, shareDiags := shareFromState(plan.Share)
	if shareDiags.HasError() {
		return nil, fmt.Errorf("invalid share")
	}

	role := &client.Role{
		Name:   plan.Name.ValueString(),
		Limits: limits,
		Share:  share,
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		value := plan.Description.ValueString()
		role.Description = &value
	}
	if !plan.DisplayName.IsNull() && !plan.DisplayName.IsUnknown() {
		value := plan.DisplayName.ValueString()
		role.DisplayName = &value
	}

	if !plan.Topics.IsNull() && !plan.Topics.IsUnknown() {
		var topics []string
		diags := plan.Topics.ElementsAs(ctx, &topics, false)
		if diags.HasError() {
			return nil, fmt.Errorf("invalid topics")
		}
		role.Topics = topics
	}

	if !plan.CostLimit.IsNull() && !plan.CostLimit.IsUnknown() {
		var costLimit roleCostLimitModel
		if diags := plan.CostLimit.As(ctx, &costLimit, basetypes.ObjectAsOptions{}); !diags.HasError() {
			role.CostLimit = &client.CostLimit{}

			if !costLimit.Day.IsNull() {
				value := client.FlexibleFloat64(costLimit.Day.ValueFloat64())
				role.CostLimit.Day = &value
			}
			if !costLimit.Minute.IsNull() {
				value := client.FlexibleFloat64(costLimit.Minute.ValueFloat64())
				role.CostLimit.Minute = &value
			}
			if !costLimit.Month.IsNull() {
				value := client.FlexibleFloat64(costLimit.Month.ValueFloat64())
				role.CostLimit.Month = &value
			}
			if !costLimit.Week.IsNull() {
				value := client.FlexibleFloat64(costLimit.Week.ValueFloat64())
				role.CostLimit.Week = &value
			}
		}
	}

	return role, nil
}

func costLimitFieldToValue(val *client.FlexibleFloat64) attr.Value {
	if val == nil {
		return types.Float64Null()
	}
	return types.Float64Value(float64(*val))
}

func setRoleState(ctx context.Context, state *roleResourceModel, role *client.Role) error {
	state.APIName = types.StringValue(role.Name)
	if state.Name.IsNull() || state.Name.IsUnknown() {
		state.Name = types.StringValue(role.Name)
	}
	limits := dropServerDefaultLimits(role.Limits, state.Limits)
	state.Limits = preserveEquivalentJSON(state.Limits, mapOrPlannedEmpty(state.Limits, limitsToState(limits), len(limits)))
	state.Share = preserveEquivalentJSON(state.Share, mapOrPlannedEmpty(state.Share, shareToState(role.Share), len(role.Share)))

	status, createdAt, updatedAt := role.Status, role.CreatedAt, role.UpdatedAt
	if role.Metadata != nil {
		if status == "" {
			status = role.Metadata.Status
		}
		if createdAt == "" {
			createdAt = string(role.Metadata.CreatedAt)
		}
		if updatedAt == "" {
			updatedAt = string(role.Metadata.UpdatedAt)
		}
	}
	state.Status = types.StringValue(status)
	state.CreatedAt = types.StringValue(createdAt)
	state.UpdatedAt = types.StringValue(updatedAt)

	if role.Description != nil {
		state.Description = types.StringValue(*role.Description)
	} else {
		state.Description = types.StringNull()
	}
	if role.DisplayName != nil {
		state.DisplayName = types.StringValue(*role.DisplayName)
	} else {
		state.DisplayName = types.StringNull()
	}

	if len(role.Topics) > 0 {
		topics, diags := types.SetValueFrom(ctx, types.StringType, role.Topics)
		if diags.HasError() {
			return fmt.Errorf("failed to set topics: %s", diags.Errors()[0].Summary())
		}
		state.Topics = topics
	} else {
		state.Topics = types.SetNull(types.StringType)
	}

	if len(role.GrantedKeys) > 0 {
		grantedKeys, diags := types.SetValueFrom(ctx, types.StringType, role.GrantedKeys)
		if diags.HasError() {
			return fmt.Errorf("failed to set granted_keys: %s", diags.Errors()[0].Summary())
		}
		state.GrantedKeys = grantedKeys
	} else {
		state.GrantedKeys = types.SetNull(types.StringType)
	}

	if role.CostLimit == nil {
		state.CostLimit = types.ObjectNull(map[string]attr.Type{
			"day": types.Float64Type, "minute": types.Float64Type,
			"month": types.Float64Type, "week": types.Float64Type,
		})
		return nil
	}

	costLimitObj := map[string]attr.Value{
		"day":    costLimitFieldToValue(role.CostLimit.Day),
		"minute": costLimitFieldToValue(role.CostLimit.Minute),
		"month":  costLimitFieldToValue(role.CostLimit.Month),
		"week":   costLimitFieldToValue(role.CostLimit.Week),
	}
	obj, objDiags := types.ObjectValue(map[string]attr.Type{
		"day": types.Float64Type, "minute": types.Float64Type,
		"month": types.Float64Type, "week": types.Float64Type,
	}, costLimitObj)
	if objDiags.HasError() {
		return fmt.Errorf("failed to build cost_limit object")
	}
	state.CostLimit = obj

	return nil
}

func (r *roleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan roleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := roleFromPlan(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid role configuration", err.Error())
		return
	}

	if err := r.client.SaveRole(ctx, plan.Bucket.ValueString(), plan.Path.ValueString(), role); err != nil {
		resp.Diagnostics.AddError("Failed to create role", err.Error())
		return
	}

	saved, err := r.client.GetRole(ctx, plan.Bucket.ValueString(), plan.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read created role", err.Error())
		return
	}
	if saved == nil {
		resp.Diagnostics.AddError("Role not found after creation", "The API returned success but the role could not be read.")
		return
	}

	plan.ID = types.StringValue(plan.Bucket.ValueString() + "/" + plan.Path.ValueString())
	if err := setRoleState(ctx, &plan, saved); err != nil {
		resp.Diagnostics.AddError("Failed to store role state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *roleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state roleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := r.client.GetRole(ctx, state.Bucket.ValueString(), state.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read role", err.Error())
		return
	}
	if role == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	if err := setRoleState(ctx, &state, role); err != nil {
		resp.Diagnostics.AddError("Failed to store role state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *roleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan roleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := roleFromPlan(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid role configuration", err.Error())
		return
	}

	if err := r.client.SaveRole(ctx, plan.Bucket.ValueString(), plan.Path.ValueString(), role); err != nil {
		resp.Diagnostics.AddError("Failed to update role", err.Error())
		return
	}

	saved, err := r.client.GetRole(ctx, plan.Bucket.ValueString(), plan.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read updated role", err.Error())
		return
	}
	if saved == nil {
		resp.Diagnostics.AddError("Role not found after update", "The API returned success but the role could not be read.")
		return
	}

	plan.ID = types.StringValue(plan.Bucket.ValueString() + "/" + plan.Path.ValueString())
	if err := setRoleState(ctx, &plan, saved); err != nil {
		resp.Diagnostics.AddError("Failed to store role state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *roleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state roleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteRole(ctx, state.Bucket.ValueString(), state.Path.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete role", err.Error())
	}
}

func (r *roleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Use terraform import aidial_role.example <bucket>/<path>.",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bucket"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), parts[1])...)
}
