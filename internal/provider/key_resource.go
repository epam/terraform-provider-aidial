package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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

	AllowedIPAddressRanges types.Set `tfsdk:"allowed_ip_address_ranges"`
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
			"allowed_ip_address_ranges": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Validators:  []validator.Set{setvalidator.ValueStringsAre(cidrValidator{})},
				Description: "Client IP ranges (CIDR, IPv4 or IPv6) allowed to use the key. Unset allows any client; an empty set allows no client. DIAL Core stores each range with its host bits cleared (`198.51.100.14/24` becomes `198.51.100.0/24`); the configured spelling is kept in state while it is equivalent.",
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

	if !plan.AllowedIPAddressRanges.IsNull() && !plan.AllowedIPAddressRanges.IsUnknown() {
		cidrs := []string{}
		if diags := plan.AllowedIPAddressRanges.ElementsAs(ctx, &cidrs, false); diags.HasError() {
			return nil, fmt.Errorf("invalid allowed_ip_address_ranges")
		}
		result.AllowedIPAddressRanges = &client.AllowedIPAddressRanges{CIDRs: cidrs}
	}

	return result, nil
}

// allowedIPRangesToState maps the ranges returned by DIAL Core to state. Core
// normalizes CIDRs, so the prior (planned or stored) set is kept whenever it
// is equivalent; that also covers ranges Core returns in an encoding the
// client cannot decode. Absent ranges are null.
func allowedIPRangesToState(ctx context.Context, prior types.Set, api *client.AllowedIPAddressRanges) (types.Set, error) {
	priorKnown := !prior.IsNull() && !prior.IsUnknown()
	if api == nil {
		return types.SetNull(types.StringType), nil
	}
	if api.Undecodable {
		if priorKnown {
			return prior, nil
		}
		return types.SetNull(types.StringType), nil
	}

	if priorKnown {
		var priorCIDRs []string
		if diags := prior.ElementsAs(ctx, &priorCIDRs, false); !diags.HasError() {
			normalized := make(map[string]struct{}, len(priorCIDRs))
			for _, cidr := range priorCIDRs {
				if n, err := client.NormalizeCIDR(cidr); err == nil {
					normalized[n] = struct{}{}
				}
			}
			same := len(normalized) == len(api.CIDRs)
			for _, cidr := range api.CIDRs {
				if _, ok := normalized[cidr]; !ok {
					same = false
				}
			}
			if same {
				return prior, nil
			}
		}
	}

	value, diags := types.SetValueFrom(ctx, types.StringType, api.CIDRs)
	if diags.HasError() {
		return types.SetNull(types.StringType), fmt.Errorf("failed to set allowed_ip_address_ranges: %s", diags.Errors()[0].Summary())
	}
	return value, nil
}

// optionalString maps an optional API string to state. DIAL Core omits empty
// values, so an empty api is stored as null, except when prior (the plan on
// create/update, the state on refresh) is exactly "": Core cannot echo that
// back, and storing null would contradict the plan or show a perpetual diff.
func optionalString(api string, prior types.String) types.String {
	if api != "" {
		return types.StringValue(api)
	}
	if !prior.IsNull() && !prior.IsUnknown() && prior.ValueString() == "" {
		return prior
	}

	return types.StringNull()
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
	state.Project = optionalString(key.Project, state.Project)
	state.Role = optionalString(key.Role, state.Role)
	state.Roles = roles
	state.Name = types.StringValue(key.Name)
	state.Status = types.StringValue(key.Status)

	ranges, err := allowedIPRangesToState(ctx, state.AllowedIPAddressRanges, key.AllowedIPAddressRanges)
	if err != nil {
		return err
	}
	state.AllowedIPAddressRanges = ranges

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

// cidrValidator accepts CIDR notation (IPv4 or IPv6), the only form DIAL Core
// accepts for allowedIpAddressRanges.
type cidrValidator struct{}

func (cidrValidator) Description(context.Context) string {
	return "value must be a CIDR such as 198.51.100.0/24 or 2001:db8::/32"
}

func (v cidrValidator) MarkdownDescription(ctx context.Context) string { return v.Description(ctx) }

func (v cidrValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if _, err := client.NormalizeCIDR(req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid CIDR", v.Description(ctx))
	}
}
