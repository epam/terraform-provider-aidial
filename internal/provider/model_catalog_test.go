package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	catalogSchemaIDForTest   = "https://dial.epam.com/catalog-schemas/model"
	catalogPropertiesForTest = `{"provider":"Provider","vendor":"Vendor","license":"License","knowledgeCutoffDate":"2026-08-17","parameters":"100B"}`
)

func TestModelFromPlanSendsCatalogFields(t *testing.T) {
	plan := newModelPlan(t)
	plan.CatalogSchemaID = types.StringValue(catalogSchemaIDForTest)
	plan.CatalogProperties = dynamicFromJSON(t, catalogPropertiesForTest)

	model, err := modelFromPlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("modelFromPlan: %v", err)
	}
	if model.CatalogSchemaID == nil || *model.CatalogSchemaID != catalogSchemaIDForTest {
		t.Fatalf("unexpected catalogSchemaId: %v", model.CatalogSchemaID)
	}
	assertJSONEqual(t, model.CatalogProperties, catalogPropertiesForTest)
}

func TestModelFromPlanCatalogNonStringValues(t *testing.T) {
	const raw = `{"contextWindow":128000,"multimodal":true,"tags":["a","b"],"nested":{"x":1}}`
	plan := newModelPlan(t)
	plan.CatalogProperties = dynamicFromJSON(t, raw)

	model, err := modelFromPlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("modelFromPlan: %v", err)
	}
	assertJSONEqual(t, model.CatalogProperties, raw)
}

func TestModelFromPlanRejectsNonObjectCatalogProperties(t *testing.T) {
	plan := newModelPlan(t)
	plan.CatalogProperties = types.DynamicValue(types.StringValue("nope"))
	if _, err := modelFromPlan(context.Background(), plan); err == nil {
		t.Fatal("expected an error for a non-object catalog_properties value")
	}
}

func TestModelFromPlanOmitsUnsetCatalogFields(t *testing.T) {
	plan := newModelPlan(t)
	plan.CatalogSchemaID = types.StringNull()
	plan.CatalogProperties = types.DynamicNull()

	model, err := modelFromPlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("modelFromPlan: %v", err)
	}
	body, _ := json.Marshal(model)
	if bytes.Contains(body, []byte("catalogSchemaId")) || bytes.Contains(body, []byte("catalogProperties")) {
		t.Fatalf("unset catalog fields must not be sent: %s", body)
	}
}

func TestSetModelStateReadsCatalogFields(t *testing.T) {
	state := modelResourceModel{Name: types.StringValue("m")}
	id := catalogSchemaIDForTest
	api := &client.Model{Name: "m", Type: "CHAT", CatalogSchemaID: &id, CatalogProperties: json.RawMessage(catalogPropertiesForTest)}

	if err := setModelState(context.Background(), &state, api); err != nil {
		t.Fatalf("setModelState: %v", err)
	}
	if state.CatalogSchemaID.ValueString() != catalogSchemaIDForTest {
		t.Fatalf("unexpected catalog_schema_id: %v", state.CatalogSchemaID)
	}
	got, _ := freeFormObjectFromPlan("catalog_properties", state.CatalogProperties)
	assertJSONEqual(t, got, catalogPropertiesForTest)
}

func TestSetModelStateKeepsConfiguredCatalogProperties(t *testing.T) {
	prior := dynamicFromJSON(t, catalogPropertiesForTest)
	state := modelResourceModel{Name: types.StringValue("m"), CatalogProperties: prior}
	api := json.RawMessage(`{"parameters":"100B","license":"License","vendor":"Vendor","provider":"Provider","knowledgeCutoffDate":"2026-08-17"}`)

	if err := setModelState(context.Background(), &state, &client.Model{Name: "m", Type: "CHAT", CatalogProperties: api}); err != nil {
		t.Fatalf("setModelState: %v", err)
	}
	if !state.CatalogProperties.Equal(prior) {
		t.Fatalf("configured catalog_properties must be kept, got %v", state.CatalogProperties)
	}
}

func TestSetModelStateCatalogDrift(t *testing.T) {
	state := modelResourceModel{Name: types.StringValue("m"), CatalogProperties: dynamicFromJSON(t, `{"vendor":"Old"}`)}
	if err := setModelState(context.Background(), &state, &client.Model{Name: "m", Type: "CHAT", CatalogProperties: json.RawMessage(`{"vendor":"New"}`)}); err != nil {
		t.Fatalf("setModelState: %v", err)
	}
	got, _ := freeFormObjectFromPlan("catalog_properties", state.CatalogProperties)
	assertJSONEqual(t, got, `{"vendor":"New"}`)
}

func TestSetModelStateEmptyCatalogProperties(t *testing.T) {
	for name, tc := range map[string]struct {
		prior    types.Dynamic
		api      json.RawMessage
		wantNull bool
	}{
		"unset, API omits":     {types.DynamicNull(), nil, true},
		"unset, API {}":        {types.DynamicNull(), json.RawMessage(`{}`), true},
		"configured {}, omits": {dynamicFromJSON(t, `{}`), nil, false},
		"configured {}, {}":    {dynamicFromJSON(t, `{}`), json.RawMessage(`{}`), false},
	} {
		state := modelResourceModel{Name: types.StringValue("m"), CatalogProperties: tc.prior}
		if err := setModelState(context.Background(), &state, &client.Model{Name: "m", Type: "CHAT", CatalogProperties: tc.api}); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if state.CatalogProperties.IsNull() != tc.wantNull {
			t.Fatalf("%s: null=%v, want %v", name, state.CatalogProperties.IsNull(), tc.wantNull)
		}
		if !state.CatalogSchemaID.IsNull() {
			t.Fatalf("%s: catalog_schema_id must be null when the API omits it", name)
		}
	}
}
