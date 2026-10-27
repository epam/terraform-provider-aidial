package provider

import (
	"context"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type roleDataSource struct{ client *client.Client }
type roleDataSourceModel struct {
	ID          types.String  `tfsdk:"id"`
	Bucket      types.String  `tfsdk:"bucket"`
	Path        types.String  `tfsdk:"path"`
	Name        types.String  `tfsdk:"name"`
	Description types.String  `tfsdk:"description"`
	DisplayName types.String  `tfsdk:"display_name"`
	Limits      types.Map     `tfsdk:"limits"`
	Share       types.Map     `tfsdk:"share"`
	CostDay     types.Float64 `tfsdk:"cost_limit_day"`
	CostMinute  types.Float64 `tfsdk:"cost_limit_minute"`
	CostMonth   types.Float64 `tfsdk:"cost_limit_month"`
	CostWeek    types.Float64 `tfsdk:"cost_limit_week"`
	Topics      types.Set     `tfsdk:"topics"`
	GrantedKeys types.Set     `tfsdk:"granted_keys"`
	CreatedAt   types.String  `tfsdk:"created_at"`
	UpdatedAt   types.String  `tfsdk:"updated_at"`
	Status      types.String  `tfsdk:"status"`
}

func (d *roleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}
func (d *roleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Looks up an existing DIAL Core role without managing it.", Attributes: map[string]schema.Attribute{
		"bucket": schema.StringAttribute{Required: true}, "path": schema.StringAttribute{Required: true}, "id": schema.StringAttribute{Computed: true}, "name": schema.StringAttribute{Computed: true}, "description": schema.StringAttribute{Computed: true}, "display_name": schema.StringAttribute{Computed: true}, "limits": schema.MapAttribute{Computed: true, ElementType: types.StringType}, "share": schema.MapAttribute{Computed: true, ElementType: types.StringType}, "cost_limit_day": schema.Float64Attribute{Computed: true}, "cost_limit_minute": schema.Float64Attribute{Computed: true}, "cost_limit_month": schema.Float64Attribute{Computed: true}, "cost_limit_week": schema.Float64Attribute{Computed: true}, "topics": schema.SetAttribute{Computed: true, ElementType: types.StringType}, "granted_keys": schema.SetAttribute{Computed: true, ElementType: types.StringType}, "created_at": schema.StringAttribute{Computed: true}, "updated_at": schema.StringAttribute{Computed: true}, "status": schema.StringAttribute{Computed: true}}}
}
func (d *roleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Invalid provider data", "Expected *client.Client")
		return
	}
	d.client = c
}
func (d *roleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state roleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	role, err := d.client.GetRole(ctx, state.Bucket.ValueString(), state.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read role", err.Error())
		return
	}
	if role == nil {
		resp.Diagnostics.AddError("Role not found", "The requested role does not exist.")
		return
	}
	state.ID = types.StringValue(state.Bucket.ValueString() + "/" + state.Path.ValueString())
	state.Name = types.StringValue(role.Name)
	state.Limits = limitsToState(role.Limits)
	state.Share = shareToState(role.Share)
	state.Status = types.StringValue(role.Status)
	state.CreatedAt = types.StringValue(role.CreatedAt)
	state.UpdatedAt = types.StringValue(role.UpdatedAt)
	if role.Description != nil {
		state.Description = types.StringValue(*role.Description)
	}
	if role.DisplayName != nil {
		state.DisplayName = types.StringValue(*role.DisplayName)
	}
	if len(role.Topics) > 0 {
		if topics, diags := types.SetValueFrom(ctx, types.StringType, role.Topics); !diags.HasError() {
			state.Topics = topics
		}
	} else {
		state.Topics = types.SetNull(types.StringType)
	}
	if len(role.GrantedKeys) > 0 {
		if grantedKeys, diags := types.SetValueFrom(ctx, types.StringType, role.GrantedKeys); !diags.HasError() {
			state.GrantedKeys = grantedKeys
		}
	} else {
		state.GrantedKeys = types.SetNull(types.StringType)
	}
	if role.CostLimit != nil {
		if role.CostLimit.Day != nil {
			state.CostDay = types.Float64Value(float64(*role.CostLimit.Day))
		}
		if role.CostLimit.Minute != nil {
			state.CostMinute = types.Float64Value(float64(*role.CostLimit.Minute))
		}
		if role.CostLimit.Month != nil {
			state.CostMonth = types.Float64Value(float64(*role.CostLimit.Month))
		}
		if role.CostLimit.Week != nil {
			state.CostWeek = types.Float64Value(float64(*role.CostLimit.Week))
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
