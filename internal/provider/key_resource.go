package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type keyResource struct {
	client *client.Client
}

type keyResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Bucket  types.String `tfsdk:"bucket"`
	Path    types.String `tfsdk:"path"`
	Key     types.String `tfsdk:"key"`
	Project types.String `tfsdk:"project"`
	Role    types.String `tfsdk:"role"`
	Roles   types.Set    `tfsdk:"roles"`
	Secured types.Bool   `tfsdk:"secured"`
	Name    types.String `tfsdk:"name"`
	Status  types.String `tfsdk:"status"`
}

func (r *keyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_key"
}

func (r *keyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an API key bound to a role in DIAL Core (`PUT /v1/keys/{bucket}/{path}`). The secret `key` is never returned by the API; the value from configuration is kept in state.",
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
				Description: "Key path inside the target bucket.",
			},
			"key": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "DIAL API key value.",
			},
			"project": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Project associated with the key.",
			},
			"role": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Primary role associated with the key.",
			},
			"roles": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Roles associated with the key.",
			},
			"secured": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the key is secured.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Key name returned by the DIAL API.",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Key status returned by the DIAL API.",
			},
		},
	}
}

func (r *keyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func keyFromPlan(ctx context.Context, plan keyResourceModel) (*client.Key, error) {
	result := &client.Key{
		Key:     plan.Key.ValueString(),
		Project: plan.Project.ValueString(),
		Role:    plan.Role.ValueString(),
	}

	if !plan.Secured.IsNull() && !plan.Secured.IsUnknown() {
		secured := plan.Secured.ValueBool()
		result.Secured = &secured
	}

	if !plan.Roles.IsNull() && !plan.Roles.IsUnknown() {
		var roles []string

		diags := plan.Roles.ElementsAs(ctx, &roles, false)
		if diags.HasError() {
			return nil, fmt.Errorf("invalid roles")
		}

		result.Roles = roles
	}

	return result, nil
}

func setKeyState(ctx context.Context, state *keyResourceModel, key *client.Key) error {
	roles, diags := types.SetValueFrom(ctx, types.StringType, key.Roles)
	if diags.HasError() {
		return fmt.Errorf("failed to set roles: %s", diags.Errors()[0].Summary())
	}

	// key is writeOnly per the DIAL Core API contract: GET never echoes
	// the secret value back, so key.Key decodes as "" on refresh. state.Key
	// is Required (not Computed), so it must be left as whatever the plan
	// already has rather than overwritten here.
	state.Project = types.StringValue(key.Project)
	state.Role = types.StringValue(key.Role)
	state.Roles = roles
	state.Name = types.StringValue(key.Name)
	state.Status = types.StringValue(key.Status)

	if key.Secured == nil {
		state.Secured = types.BoolNull()
	} else {
		state.Secured = types.BoolValue(*key.Secured)
	}

	return nil
}

func (r *keyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan keyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, err := keyFromPlan(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid key configuration", err.Error())
		return
	}

	if err := r.client.SaveKey(ctx, plan.Bucket.ValueString(), plan.Path.ValueString(), key); err != nil {
		resp.Diagnostics.AddError("Failed to create key", err.Error())
		return
	}

	saved, err := r.client.GetKey(ctx, plan.Bucket.ValueString(), plan.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read created key", err.Error())
		return
	}
	if saved == nil {
		resp.Diagnostics.AddError(
			"Key not found after creation",
			"The API returned success but the key could not be read.",
		)
		return
	}

	if saved.Key == "" {
		saved.Key = plan.Key.ValueString()
	}

	plan.ID = types.StringValue(plan.Bucket.ValueString() + "/" + plan.Path.ValueString())

	if err := setKeyState(ctx, &plan, saved); err != nil {
		resp.Diagnostics.AddError("Failed to store key state", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *keyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state keyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, err := r.client.GetKey(ctx, state.Bucket.ValueString(), state.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read key", err.Error())
		return
	}
	if key == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	if key.Key == "" {
		key.Key = state.Key.ValueString()
	}

	if err := setKeyState(ctx, &state, key); err != nil {
		resp.Diagnostics.AddError("Failed to store key state", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *keyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan keyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, err := keyFromPlan(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid key configuration", err.Error())
		return
	}

	if err := r.client.SaveKey(ctx, plan.Bucket.ValueString(), plan.Path.ValueString(), key); err != nil {
		resp.Diagnostics.AddError("Failed to update key", err.Error())
		return
	}

	saved, err := r.client.GetKey(ctx, plan.Bucket.ValueString(), plan.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read updated key", err.Error())
		return
	}
	if saved == nil {
		resp.Diagnostics.AddError(
			"Key not found after update",
			"The API returned success but the key could not be read.",
		)
		return
	}

	if saved.Key == "" {
		saved.Key = plan.Key.ValueString()
	}

	plan.ID = types.StringValue(plan.Bucket.ValueString() + "/" + plan.Path.ValueString())

	if err := setKeyState(ctx, &plan, saved); err != nil {
		resp.Diagnostics.AddError("Failed to store key state", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *keyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state keyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteKey(ctx, state.Bucket.ValueString(), state.Path.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete key", err.Error())
	}
}

func (r *keyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Use terraform import aidial_key.example <bucket>/<path>.",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bucket"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), parts[1])...)
}
