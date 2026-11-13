package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &aliasResource{}
	_ resource.ResourceWithImportState    = &aliasResource{}
	_ resource.ResourceWithValidateConfig = &aliasResource{}
)

// aliasBuckets are the buckets DIAL Core accepts for aliases. The OpenAPI
// document says "public" only, but Core 0.50.1 accepts "platform" too and
// answers 404 for any other bucket.
var aliasBuckets = []string{"public", "platform"}

// aliasPresentations are the values of Core's AliasPresentation.
var aliasPresentations = []string{"model", "application", "toolset"}

// aliasTargetPathPattern is the target path form Core accepts: the kind
// followed by one name segment.
var aliasTargetPathPattern = regexp.MustCompile(`^(models|applications|toolsets)/[^/]+$`)

type aliasResource struct {
	client *client.Client
}

// aliasModel is the state of aidial_alias and of the aidial_alias data source.
type aliasModel struct {
	ID      types.String `tfsdk:"id"`
	Bucket  types.String `tfsdk:"bucket"`
	Path    types.String `tfsdk:"path"`
	Name    types.String `tfsdk:"name"`
	APIName types.String `tfsdk:"api_name"`

	aliasBody
}

// aliasBody holds every attribute of an alias except its identity. It is kept
// separate so that all three entry points (resource, data source, file data
// source) share one mapping.
type aliasBody struct {
	PresentAs       types.String `tfsdk:"present_as"`
	TargetPath      types.String `tfsdk:"target_path"`
	DisplayName     types.String `tfsdk:"display_name"`
	Description     types.String `tfsdk:"description"`
	DisplayVersion  types.String `tfsdk:"display_version"`
	IconURL         types.String `tfsdk:"icon_url"`
	Intro           types.String `tfsdk:"intro"`
	UserRoles       types.Set    `tfsdk:"user_roles"`
	DescriptionKeys types.Set    `tfsdk:"description_keywords"`
}

func (r *aliasResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alias"
}

func (r *aliasResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	str := func(description string) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Description: description}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an alias in DIAL Core (`PUT /v1/aliases/{bucket}/{path}`): an alternative invocation name that points at a model, application or toolset and can override how it is presented and who may use it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "Resource identifier in the form <bucket>/<path>.",
			},
			"bucket": schema.StringAttribute{
				Required:      true,
				Validators:    []validator.String{stringvalidator.OneOf(aliasBuckets...)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Target DIAL bucket. Must be \"public\" or \"platform\".",
			},
			"path": schema.StringAttribute{
				Required:      true,
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Alias path inside the bucket, which is also the name the alias is invoked by. DIAL Core accepts letters, digits and `._%:@[]()-`, so it cannot contain `/`.",
			},
			"name": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "Alias name sent in the DIAL API request body. It must equal the last segment of `path`, so it can be omitted; then it is read back from the API.",
			},
			"api_name": schema.StringAttribute{
				Computed:    true,
				Description: "Canonical alias name returned by the DIAL API.",
			},
			"present_as": schema.StringAttribute{
				Required:    true,
				Validators:  []validator.String{stringvalidator.OneOf(aliasPresentations...)},
				Description: "What the alias is shown as: `model`, `application` or `toolset`. An alias shown as a toolset must point at a toolset; one shown as a model or application cannot point at a toolset.",
			},
			"target_path": schema.StringAttribute{
				Required:    true,
				Validators:  []validator.String{stringvalidator.RegexMatches(aliasTargetPathPattern, "must be models/<name>, applications/<name> or toolsets/<name>")},
				Description: "The deployment the alias points at: `models/<name>`, `applications/<name>` or `toolsets/<name>`. DIAL Core does not check that the target exists.",
			},
			"display_name":    str("Human-readable alias name."),
			"description":     str("Alias description."),
			"display_version": str("Version shown for the alias."),
			"icon_url":        str("URL of the alias icon."),
			"intro":           str("Introductory text shown for the alias."),
			"user_roles":      schema.SetAttribute{Optional: true, ElementType: types.StringType, Description: "Roles allowed to use the alias."},
			"description_keywords": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Keywords that help find the alias by description.",
			},
		},
	}
}

func (r *aliasResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// aliasTargetKindError returns a message when present_as and the kind of
// target_path cannot go together, as DIAL Core would reject them. Every other
// combination is left to Core.
func aliasTargetKindError(presentAs, targetPath string) string {
	isToolset := strings.HasPrefix(targetPath, "toolsets/")
	switch {
	case presentAs == "toolset" && !isToolset:
		return fmt.Sprintf("An alias shown as a toolset must point at a toolset (toolsets/<name>), got %q.", targetPath)
	case (presentAs == "model" || presentAs == "application") && isToolset:
		return fmt.Sprintf("An alias shown as %s cannot point at a toolset, got %q.", presentAs, targetPath)
	}
	return ""
}

func (r *aliasResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var presentAs, targetPath types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("present_as"), &presentAs)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("target_path"), &targetPath)...)
	if resp.Diagnostics.HasError() || presentAs.IsNull() || presentAs.IsUnknown() || targetPath.IsNull() || targetPath.IsUnknown() {
		return
	}
	if msg := aliasTargetKindError(presentAs.ValueString(), targetPath.ValueString()); msg != "" {
		resp.Diagnostics.AddAttributeError(path.Root("target_path"), "Invalid alias target", msg)
	}
}

// aliasFromPlan converts planned values to the API shape. Null and unknown
// attributes are not sent; a configured empty set is sent as [].
func aliasFromPlan(ctx context.Context, plan aliasModel) (*client.Alias, error) {
	a := &client.Alias{
		Name:           plan.Name.ValueString(),
		PresentAs:      plan.PresentAs.ValueString(),
		Target:         &client.AliasTarget{Path: plan.TargetPath.ValueString()},
		DisplayName:    stringPointer(plan.DisplayName),
		Description:    stringPointer(plan.Description),
		DisplayVersion: stringPointer(plan.DisplayVersion),
		IconURL:        stringPointer(plan.IconURL),
		Intro:          stringPointer(plan.Intro),
	}

	userRoles, err := stringSetToAPI(ctx, plan.UserRoles)
	if err != nil {
		return nil, fmt.Errorf("invalid user_roles: %w", err)
	}
	a.UserRoles = userRoles

	keywords, err := stringSetToAPI(ctx, plan.DescriptionKeys)
	if err != nil {
		return nil, fmt.Errorf("invalid description_keywords: %w", err)
	}
	a.DescriptionWords = keywords
	return a, nil
}

// setAliasState maps an API alias into state.
func setAliasState(ctx context.Context, state *aliasModel, a *client.Alias) error {
	state.APIName = types.StringValue(a.Name)
	if state.Name.IsNull() || state.Name.IsUnknown() {
		state.Name = types.StringValue(a.Name)
	}
	return setAliasBodyState(ctx, &state.aliasBody, a)
}

func setAliasBodyState(ctx context.Context, state *aliasBody, a *client.Alias) error {
	state.PresentAs = types.StringValue(a.PresentAs)
	state.TargetPath = types.StringNull()
	if a.Target != nil {
		state.TargetPath = types.StringValue(a.Target.Path)
	}
	state.DisplayName = stringValue(a.DisplayName)
	state.Description = stringValue(a.Description)
	state.DisplayVersion = stringValue(a.DisplayVersion)
	state.IconURL = stringValue(a.IconURL)
	state.Intro = stringValue(a.Intro)

	userRoles, err := stringSetFromAPI(ctx, a.UserRoles)
	if err != nil {
		return fmt.Errorf("failed to set user_roles: %w", err)
	}
	state.UserRoles = userRoles

	keywords, err := stringSetFromAPI(ctx, a.DescriptionWords)
	if err != nil {
		return fmt.Errorf("failed to set description_keywords: %w", err)
	}
	state.DescriptionKeys = keywords
	return nil
}

func aliasID(bucket, path types.String) types.String {
	return types.StringValue(bucket.ValueString() + "/" + path.ValueString())
}

// save PUTs the planned alias and re-reads it, because PUT returns only the
// name.
func (r *aliasResource) save(ctx context.Context, plan *aliasModel, verb string) error {
	a, err := aliasFromPlan(ctx, *plan)
	if err != nil {
		return fmt.Errorf("invalid alias configuration: %w", err)
	}
	bucket, path := plan.Bucket.ValueString(), plan.Path.ValueString()
	if err := r.client.SaveAlias(ctx, bucket, path, a); err != nil {
		return fmt.Errorf("failed to %s alias: %w", verb, err)
	}
	saved, err := r.client.GetAlias(ctx, bucket, path)
	if err != nil {
		return fmt.Errorf("failed to read %sd alias: %w", verb, err)
	}
	if saved == nil {
		return fmt.Errorf("alias not found after %s: the API returned success but the alias could not be read", verb)
	}
	plan.ID = aliasID(plan.Bucket, plan.Path)
	if err := setAliasState(ctx, plan, saved); err != nil {
		return fmt.Errorf("failed to store alias state: %w", err)
	}
	return nil
}

func (r *aliasResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan aliasModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.save(ctx, &plan, "create"); err != nil {
		resp.Diagnostics.AddError("Failed to create alias", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *aliasResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state aliasModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	a, err := r.client.GetAlias(ctx, state.Bucket.ValueString(), state.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read alias", err.Error())
		return
	}
	if a == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	if err := setAliasState(ctx, &state, a); err != nil {
		resp.Diagnostics.AddError("Failed to store alias state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *aliasResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan aliasModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.save(ctx, &plan, "update"); err != nil {
		resp.Diagnostics.AddError("Failed to update alias", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *aliasResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state aliasModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteAlias(ctx, state.Bucket.ValueString(), state.Path.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete alias", err.Error())
	}
}

func (r *aliasResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Use terraform import aidial_alias.example <bucket>/<path>.",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bucket"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), parts[1])...)
}
