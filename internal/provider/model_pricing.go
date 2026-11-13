package provider

import (
	"context"
	"fmt"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// pricingRateModel is one entry of aidial_model.pricing (prompt, completion,
// cache_read, cache_write): a flat rate, or a condition with two prices.
// DIAL Core's PricingRate nests recursively; Terraform schemas cannot, so
// if_true and if_false are flat prices.
type pricingRateModel struct {
	Rate    types.String `tfsdk:"rate"`
	Test    types.Object `tfsdk:"test"`
	IfTrue  types.String `tfsdk:"if_true"`
	IfFalse types.String `tfsdk:"if_false"`
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
	"if_true":  types.StringType,
	"if_false": types.StringType,
}

var pricingOperators = []string{"==", "!=", ">", "<", ">=", "<="}

const pricingRateDescription = "A flat `rate`, or a conditional price: `test` selects `if_true` or `if_false`. `rate` cannot be combined with the other attributes."

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
			"if_true": schema.StringAttribute{
				Optional:    true,
				Description: "Flat price used when `test` holds.",
				Validators:  []validator.String{stringvalidator.AlsoRequires(sibling("test"))},
			},
			"if_false": schema.StringAttribute{
				Optional:    true,
				Description: "Flat price used when `test` does not hold.",
				Validators:  []validator.String{stringvalidator.AlsoRequires(sibling("test"))},
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
			"if_true":  datasourceschema.StringAttribute{Computed: true},
			"if_false": datasourceschema.StringAttribute{Computed: true},
		},
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
	if ifTrue := stringPointer(m.IfTrue); ifTrue != nil {
		result.IfTrue = &client.PricingRate{Rate: ifTrue}
	}
	if ifFalse := stringPointer(m.IfFalse); ifFalse != nil {
		result.IfFalse = &client.PricingRate{Rate: ifFalse}
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

// pricingRateToState converts a rate returned by Core. Terraform cannot
// represent a condition nested inside if_true/if_false, so that is an error
// rather than a silent loss.
func pricingRateToState(name string, rate *client.PricingRate) (types.Object, error) {
	if rate == nil {
		return types.ObjectNull(pricingRateAttrTypes), nil
	}

	branch := func(label string, r *client.PricingRate) (types.String, error) {
		if r == nil {
			return types.StringNull(), nil
		}
		if r.Test != nil || r.IfTrue != nil || r.IfFalse != nil {
			return types.StringNull(), fmt.Errorf("pricing.%s.%s is a nested condition, which the provider does not support", name, label)
		}
		return stringValue(r.Rate), nil
	}

	ifTrue, err := branch("if_true", rate.IfTrue)
	if err != nil {
		return types.ObjectNull(pricingRateAttrTypes), err
	}
	ifFalse, err := branch("if_false", rate.IfFalse)
	if err != nil {
		return types.ObjectNull(pricingRateAttrTypes), err
	}

	test := types.ObjectNull(pricingConditionAttrTypes)
	if rate.Test != nil {
		obj, d := types.ObjectValue(pricingConditionAttrTypes, map[string]attr.Value{
			"field":    types.StringValue(rate.Test.Field),
			"operator": types.StringValue(rate.Test.Operator),
			"value":    types.StringValue(rate.Test.Value),
		})
		if d.HasError() {
			return types.ObjectNull(pricingRateAttrTypes), fmt.Errorf("failed to build pricing.%s.test object", name)
		}
		test = obj
	}

	obj, diags := types.ObjectValue(pricingRateAttrTypes, map[string]attr.Value{
		"rate":     stringValue(rate.Rate),
		"test":     test,
		"if_true":  ifTrue,
		"if_false": ifFalse,
	})
	if diags.HasError() {
		return types.ObjectNull(pricingRateAttrTypes), fmt.Errorf("failed to build pricing.%s object", name)
	}

	return obj, nil
}
