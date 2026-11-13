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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &translatorResource{}
	_ resource.ResourceWithImportState = &translatorResource{}
)

type translatorResource struct {
	client *client.Client
}

// translatorModel is the state of aidial_translator and of the
// aidial_translator data source.
type translatorModel struct {
	ID     types.String `tfsdk:"id"`
	Bucket types.String `tfsdk:"bucket"`
	Path   types.String `tfsdk:"path"`

	translatorBody
}

// translatorBody holds every attribute of a translator except its identity,
// shared by the resource, the data source and the file data source.
type translatorBody struct {
	BaseURL types.String `tfsdk:"base_url"`
	In      types.String `tfsdk:"in"`
	Out     types.String `tfsdk:"out"`
}

func (r *translatorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_translator"
}

func translatorInterfaceDescription(side string) string {
	return side + " interface type. One of `" + strings.Join(modelInterfaceTypes, "`, `") + "`."
}

func (r *translatorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	interfaceType := func(description string) schema.StringAttribute {
		return schema.StringAttribute{
			Required:    true,
			Validators:  []validator.String{stringvalidator.OneOf(modelInterfaceTypes...)},
			Description: description,
		}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a translator in DIAL Core (`PUT /v1/translators/{bucket}/{path}`): a service that converts requests from the `in` interface type to the `out` interface type.",
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
				Description:   "Translator path inside the bucket. DIAL Core accepts letters, digits and `._%:@[]()-`, so it cannot contain `/`.",
			},
			"base_url": schema.StringAttribute{Required: true, Description: "Base URL of the translator service."},
			"in":       interfaceType(translatorInterfaceDescription("Incoming")),
			"out":      interfaceType(translatorInterfaceDescription("Outgoing")),
		},
	}
}

func (r *translatorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func translatorFromPlan(plan translatorBody) *client.Translator {
	return &client.Translator{
		BaseURL: plan.BaseURL.ValueString(),
		In:      plan.In.ValueString(),
		Out:     plan.Out.ValueString(),
	}
}

// setTranslatorBodyState maps an API translator into state as Core returned it.
func setTranslatorBodyState(state *translatorBody, t *client.Translator) {
	state.BaseURL = types.StringValue(t.BaseURL)
	state.In = types.StringValue(t.In)
	state.Out = types.StringValue(t.Out)
}

func translatorID(bucket, path types.String) types.String {
	return types.StringValue(bucket.ValueString() + "/" + path.ValueString())
}

// save PUTs the planned translator and re-reads it, because PUT returns only
// the name.
func (r *translatorResource) save(ctx context.Context, plan *translatorModel, verb string) error {
	bucket, path := plan.Bucket.ValueString(), plan.Path.ValueString()
	if err := r.client.SaveTranslator(ctx, bucket, path, translatorFromPlan(plan.translatorBody)); err != nil {
		return fmt.Errorf("failed to %s translator: %w", verb, err)
	}
	saved, err := r.client.GetTranslator(ctx, bucket, path)
	if err != nil {
		return fmt.Errorf("failed to read %sd translator: %w", verb, err)
	}
	if saved == nil {
		return fmt.Errorf("translator not found after %s: the API returned success but the translator could not be read", verb)
	}
	plan.ID = translatorID(plan.Bucket, plan.Path)
	setTranslatorBodyState(&plan.translatorBody, saved)
	return nil
}

func (r *translatorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan translatorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.save(ctx, &plan, "create"); err != nil {
		resp.Diagnostics.AddError("Failed to create translator", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *translatorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state translatorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	t, err := r.client.GetTranslator(ctx, state.Bucket.ValueString(), state.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read translator", err.Error())
		return
	}
	if t == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	setTranslatorBodyState(&state.translatorBody, t)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *translatorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan translatorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.save(ctx, &plan, "update"); err != nil {
		resp.Diagnostics.AddError("Failed to update translator", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *translatorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state translatorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteTranslator(ctx, state.Bucket.ValueString(), state.Path.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete translator", err.Error())
	}
}

func (r *translatorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Use terraform import aidial_translator.example <bucket>/<path>.",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bucket"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), parts[1])...)
}
