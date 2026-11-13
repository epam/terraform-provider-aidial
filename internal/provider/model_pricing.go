package provider

import (
	"context"
	"fmt"
	"math/big"
	"reflect"
	"sort"
	"strings"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/dynamicvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// pricingRateModel is one entry of aidial_model.pricing (prompt, completion,
// cache_read, cache_write): a flat rate, or a condition with two prices.
//
// DIAL Core's PricingRate nests recursively and Terraform schemas cannot, so
// if_true and if_false are dynamically typed: a flat price string, or an object
// with the same attributes as the rate itself (rate, test, if_true, if_false),
// to any depth.
type pricingRateModel struct {
	Rate    types.String  `tfsdk:"rate"`
	Test    types.Object  `tfsdk:"test"`
	IfTrue  types.Dynamic `tfsdk:"if_true"`
	IfFalse types.Dynamic `tfsdk:"if_false"`
}

type pricingConditionModel struct {
	Field    types.String `tfsdk:"field"`
	Operator types.String `tfsdk:"operator"`
	Value    types.String `tfsdk:"value"`
}

var pricingConditionAttrTypes = map[string]attr.Type{
	"field":    types.StringType,
	"operator": types.StringType,
	"value":    types.StringType,
}

var pricingRateAttrTypes = map[string]attr.Type{
	"rate":     types.StringType,
	"test":     types.ObjectType{AttrTypes: pricingConditionAttrTypes},
	"if_true":  types.DynamicType,
	"if_false": types.DynamicType,
}

var pricingOperators = []string{"==", "!=", ">", "<", ">=", "<="}

const pricingRateDescription = "A flat `rate`, or a conditional price: `test` selects `if_true` or `if_false`. `rate` cannot be combined with the other attributes."

const pricingBranchDescription = "Price used when `test` holds (`if_false`: when it does not). Either a flat price string, or an object with the same attributes as this rate (`rate`, or `test` with `if_true` and `if_false`) for nested conditions."

// pricingRateResourceAttribute is the resource schema of one pricing entry.
func pricingRateResourceAttribute() schema.Attribute {
	sibling := func(name string) path.Expression {
		return path.MatchRelative().AtParent().AtName(name)
	}

	return schema.SingleNestedAttribute{
		Optional:    true,
		Computed:    true,
		Description: pricingRateDescription,
		Attributes: map[string]schema.Attribute{
			"rate": schema.StringAttribute{
				Optional:    true,
				Description: "Flat price per unit, as a decimal string, for example `0.000002`.",
				Validators: []validator.String{
					stringvalidator.ConflictsWith(sibling("test"), sibling("if_true"), sibling("if_false")),
				},
			},
			"test": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Condition that selects `if_true` or `if_false`. Requires both.",
				Attributes: map[string]schema.Attribute{
					"field":    schema.StringAttribute{Required: true, Description: "Request field to compare, for example `promptTokens`."},
					"operator": schema.StringAttribute{Required: true, Description: "One of `==`, `!=`, `>`, `<`, `>=`, `<=`.", Validators: []validator.String{stringvalidator.OneOf(pricingOperators...)}},
					"value":    schema.StringAttribute{Required: true, Description: "Value to compare with. A number is sent to DIAL Core as a number."},
				},
				Validators: []validator.Object{
					objectvalidator.AlsoRequires(sibling("if_true"), sibling("if_false")),
				},
			},
			"if_true": schema.DynamicAttribute{
				Optional:    true,
				Description: pricingBranchDescription,
				Validators: []validator.Dynamic{
					dynamicvalidator.AlsoRequires(sibling("test")),
					pricingBranchValidator{},
				},
			},
			"if_false": schema.DynamicAttribute{
				Optional:    true,
				Description: pricingBranchDescription,
				Validators: []validator.Dynamic{
					dynamicvalidator.AlsoRequires(sibling("test")),
					pricingBranchValidator{},
				},
			},
		},
	}
}

// pricingRateDataSourceAttribute is the computed-only schema of one pricing entry.
func pricingRateDataSourceAttribute() datasourceschema.Attribute {
	return datasourceschema.SingleNestedAttribute{
		Computed:    true,
		Description: pricingRateDescription,
		Attributes: map[string]datasourceschema.Attribute{
			"rate": datasourceschema.StringAttribute{Computed: true},
			"test": datasourceschema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]datasourceschema.Attribute{
					"field":    datasourceschema.StringAttribute{Computed: true},
					"operator": datasourceschema.StringAttribute{Computed: true},
					"value":    datasourceschema.StringAttribute{Computed: true},
				},
			},
			"if_true":  datasourceschema.DynamicAttribute{Computed: true, Description: pricingBranchDescription},
			"if_false": datasourceschema.DynamicAttribute{Computed: true, Description: pricingBranchDescription},
		},
	}
}

// pricingBranchValidator checks the shape of a nested if_true/if_false value.
type pricingBranchValidator struct{}

func (pricingBranchValidator) Description(context.Context) string {
	return "must be a price string or a nested rate object"
}

func (v pricingBranchValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (pricingBranchValidator) ValidateDynamic(_ context.Context, req validator.DynamicRequest, resp *validator.DynamicResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || req.ConfigValue.IsUnderlyingValueNull() || req.ConfigValue.IsUnderlyingValueUnknown() {
		return
	}
	if _, err := pricingBranchToClient(req.ConfigValue.UnderlyingValue()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid pricing rate", err.Error())
	}
}

// pricingRateToClient converts a planned pricing entry; a null or unknown
// object means "not configured".
func pricingRateToClient(ctx context.Context, obj types.Object) (*client.PricingRate, error) {
	if obj.IsNull() || obj.IsUnknown() {
		return nil, nil
	}

	var m pricingRateModel
	if diags := obj.As(ctx, &m, basetypes.ObjectAsOptions{}); diags.HasError() {
		return nil, fmt.Errorf("invalid pricing rate: %v", diags)
	}

	result := &client.PricingRate{Rate: stringPointer(m.Rate)}
	for _, b := range []struct {
		in  types.Dynamic
		out **client.PricingRate
	}{{m.IfTrue, &result.IfTrue}, {m.IfFalse, &result.IfFalse}} {
		if b.in.IsNull() || b.in.IsUnknown() || b.in.IsUnderlyingValueNull() || b.in.IsUnderlyingValueUnknown() {
			continue
		}
		converted, err := pricingBranchToClient(b.in.UnderlyingValue())
		if err != nil {
			return nil, err
		}
		*b.out = converted
	}

	if !m.Test.IsNull() && !m.Test.IsUnknown() {
		var c pricingConditionModel
		if diags := m.Test.As(ctx, &c, basetypes.ObjectAsOptions{}); diags.HasError() {
			return nil, fmt.Errorf("invalid pricing condition: %v", diags)
		}
		result.Test = &client.PricingCondition{
			Field:    c.Field.ValueString(),
			Operator: c.Operator.ValueString(),
			Value:    c.Value.ValueString(),
		}
	}

	return result, nil
}

// pricingBranchToClient converts the value of an if_true/if_false attribute:
// a price string (or number), or an object with rate/test/if_true/if_false.
func pricingBranchToClient(value attr.Value) (*client.PricingRate, error) {
	switch v := value.(type) {
	case basetypes.DynamicValue:
		return pricingBranchToClient(v.UnderlyingValue())
	case basetypes.StringValue, basetypes.NumberValue, basetypes.Int64Value, basetypes.Float64Value:
		text, ok := scalarAttrText(v)
		if !ok {
			return nil, nil
		}
		return &client.PricingRate{Rate: &text}, nil
	case basetypes.ObjectValue:
		return pricingObjectToClient(v)
	default:
		return nil, fmt.Errorf("a pricing branch must be a price string or an object, got %s", value.Type(context.Background()))
	}
}

func pricingObjectToClient(obj basetypes.ObjectValue) (*client.PricingRate, error) {
	attrs := obj.Attributes()
	for name := range attrs {
		switch name {
		case "rate", "test", "if_true", "if_false":
		default:
			return nil, fmt.Errorf("unsupported attribute %q in a nested pricing rate (allowed: rate, test, if_true, if_false)", name)
		}
	}

	result := &client.PricingRate{}
	if rate, ok := attrs["rate"]; ok {
		if text, ok := scalarAttrText(rate); ok {
			result.Rate = &text
		}
	}
	if test, ok := attrs["test"]; ok && !test.IsNull() && !test.IsUnknown() {
		cond, err := pricingConditionFromValue(test)
		if err != nil {
			return nil, err
		}
		result.Test = cond
	}
	for name, dest := range map[string]**client.PricingRate{"if_true": &result.IfTrue, "if_false": &result.IfFalse} {
		branch, ok := attrs[name]
		if !ok || branch.IsNull() || branch.IsUnknown() {
			continue
		}
		converted, err := pricingBranchToClient(branch)
		if err != nil {
			return nil, err
		}
		*dest = converted
	}

	if result.Rate != nil && (result.Test != nil || result.IfTrue != nil || result.IfFalse != nil) {
		return nil, fmt.Errorf("a nested pricing rate cannot combine rate with test, if_true or if_false")
	}
	if (result.Test != nil) != (result.IfTrue != nil) || (result.Test != nil) != (result.IfFalse != nil) {
		return nil, fmt.Errorf("a nested pricing rate needs test, if_true and if_false together")
	}
	if result.Rate == nil && result.Test == nil {
		return nil, fmt.Errorf("a nested pricing rate needs rate, or test with if_true and if_false")
	}

	return result, nil
}

func pricingConditionFromValue(value attr.Value) (*client.PricingCondition, error) {
	obj, ok := value.(basetypes.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("test must be an object with field, operator and value")
	}

	cond := &client.PricingCondition{}
	for name, v := range obj.Attributes() {
		text, ok := scalarAttrText(v)
		if !ok {
			return nil, fmt.Errorf("test.%s must be a string", name)
		}
		switch name {
		case "field":
			cond.Field = text
		case "operator":
			cond.Operator = text
		case "value":
			cond.Value = text
		default:
			return nil, fmt.Errorf("unsupported attribute %q in test (allowed: field, operator, value)", name)
		}
	}

	if cond.Field == "" || cond.Operator == "" {
		return nil, fmt.Errorf("test needs field, operator and value")
	}
	if _, found := obj.Attributes()["value"]; !found {
		return nil, fmt.Errorf("test needs field, operator and value")
	}
	valid := false
	for _, op := range pricingOperators {
		valid = valid || op == cond.Operator
	}
	if !valid {
		return nil, fmt.Errorf("test.operator must be one of %s, got %q", strings.Join(pricingOperators, ", "), cond.Operator)
	}

	return cond, nil
}

// scalarAttrText returns the text of a string or number attribute value.
func scalarAttrText(value attr.Value) (string, bool) {
	switch v := value.(type) {
	case basetypes.StringValue:
		if v.IsNull() || v.IsUnknown() {
			return "", false
		}
		return v.ValueString(), true
	case basetypes.NumberValue:
		if v.IsNull() || v.IsUnknown() {
			return "", false
		}
		return v.ValueBigFloat().Text('f', -1), true
	case basetypes.Int64Value:
		if v.IsNull() || v.IsUnknown() {
			return "", false
		}
		return new(big.Float).SetInt64(v.ValueInt64()).Text('f', -1), true
	case basetypes.Float64Value:
		if v.IsNull() || v.IsUnknown() {
			return "", false
		}
		return new(big.Float).SetFloat64(v.ValueFloat64()).Text('f', -1), true
	}

	return "", false
}

// pricingRateToState converts a rate returned by Core. prior is the configured
// or previous value: where it describes the same price as Core's answer it is
// kept, so a branch written as { rate = "1" } is not rewritten to "1" on read.
func pricingRateToState(name string, rate *client.PricingRate, prior types.Object) (types.Object, error) {
	null := types.ObjectNull(pricingRateAttrTypes)
	if rate == nil {
		return null, nil
	}

	var priorModel *pricingRateModel
	if !prior.IsNull() && !prior.IsUnknown() {
		var m pricingRateModel
		if diags := prior.As(context.Background(), &m, basetypes.ObjectAsOptions{}); !diags.HasError() {
			priorModel = &m
		}
	}
	priorBranch := func(pick func(pricingRateModel) types.Dynamic) types.Dynamic {
		if priorModel == nil {
			return types.DynamicNull()
		}
		return pick(*priorModel)
	}

	test := types.ObjectNull(pricingConditionAttrTypes)
	if rate.Test != nil {
		obj, diags := types.ObjectValue(pricingConditionAttrTypes, map[string]attr.Value{
			"field":    types.StringValue(rate.Test.Field),
			"operator": types.StringValue(rate.Test.Operator),
			"value":    types.StringValue(rate.Test.Value),
		})
		if diags.HasError() {
			return null, fmt.Errorf("failed to build pricing.%s.test object", name)
		}
		test = obj
	}

	obj, diags := types.ObjectValue(pricingRateAttrTypes, map[string]attr.Value{
		"rate":     stringValue(rate.Rate),
		"test":     test,
		"if_true":  pricingBranchToState(rate.IfTrue, priorBranch(func(m pricingRateModel) types.Dynamic { return m.IfTrue })),
		"if_false": pricingBranchToState(rate.IfFalse, priorBranch(func(m pricingRateModel) types.Dynamic { return m.IfFalse })),
	})
	if diags.HasError() {
		return null, fmt.Errorf("failed to build pricing.%s object: %s", name, diagSummary(diags))
	}

	return obj, nil
}

func diagSummary(diags diag.Diagnostics) string {
	var parts []string
	for _, d := range diags.Errors() {
		parts = append(parts, d.Summary()+": "+d.Detail())
	}
	sort.Strings(parts)

	return strings.Join(parts, "; ")
}

// pricingBranchToState builds the dynamic value of if_true/if_false from Core's
// answer. A prior value that converts to the same rate wins; otherwise the
// canonical form is used: a string for a flat price, else an object holding
// only the attributes that are set.
func pricingBranchToState(rate *client.PricingRate, prior types.Dynamic) types.Dynamic {
	if rate == nil {
		return types.DynamicNull()
	}

	if !prior.IsNull() && !prior.IsUnknown() && !prior.IsUnderlyingValueNull() && !prior.IsUnderlyingValueUnknown() {
		if converted, err := pricingBranchToClient(prior.UnderlyingValue()); err == nil && reflect.DeepEqual(converted, rate) {
			return prior
		}
	}

	return types.DynamicValue(pricingBranchValue(rate))
}

func pricingBranchValue(rate *client.PricingRate) attr.Value {
	if rate.Test == nil && rate.IfTrue == nil && rate.IfFalse == nil && rate.Rate != nil {
		return types.StringValue(*rate.Rate)
	}

	attrs := map[string]attr.Value{}
	attrTypes := map[string]attr.Type{}
	put := func(key string, v attr.Value) {
		attrs[key] = v
		attrTypes[key] = v.Type(context.Background())
	}
	if rate.Rate != nil {
		put("rate", types.StringValue(*rate.Rate))
	}
	if rate.Test != nil {
		put("test", types.ObjectValueMust(pricingConditionAttrTypes, map[string]attr.Value{
			"field":    types.StringValue(rate.Test.Field),
			"operator": types.StringValue(rate.Test.Operator),
			"value":    types.StringValue(rate.Test.Value),
		}))
	}
	if rate.IfTrue != nil {
		put("if_true", pricingBranchValue(rate.IfTrue))
	}
	if rate.IfFalse != nil {
		put("if_false", pricingBranchValue(rate.IfFalse))
	}

	return types.ObjectValueMust(attrTypes, attrs)
}
