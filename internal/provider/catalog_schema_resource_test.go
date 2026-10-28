package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const catalogSchemaForTest = `{"$id":"https://dial.epam.com/catalog-schemas/model","type":"object","properties":{"vendor":{"type":"string"}}}`

func TestCatalogSchemaValidator(t *testing.T) {
	for name, tc := range map[string]struct {
		value   types.String
		wantErr bool
	}{
		"valid":      {types.StringValue(catalogSchemaForTest), false},
		"null":       {types.StringNull(), false},
		"unknown":    {types.StringUnknown(), false},
		"not json":   {types.StringValue(`nope`), true},
		"not object": {types.StringValue(`["a"]`), true},
		"null json":  {types.StringValue(`null`), true},
		"no id":      {types.StringValue(`{"type":"object"}`), true},
		"blank id":   {types.StringValue(`{"$id":"  "}`), true},
		"numeric id": {types.StringValue(`{"$id":1}`), true},
	} {
		resp := &validator.StringResponse{}
		catalogSchemaValidator{}.ValidateString(context.Background(), validator.StringRequest{ConfigValue: tc.value}, resp)
		if resp.Diagnostics.HasError() != tc.wantErr {
			t.Fatalf("%s: error=%v, want %v", name, resp.Diagnostics.HasError(), tc.wantErr)
		}
	}
}

func TestCatalogSchemaResourceSchemaIsValid(t *testing.T) {
	r := &catalogSchemaResource{}
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if diags := resp.Schema.ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatalf("invalid schema: %v", diags)
	}
}

func TestSetCatalogSchemaStateKeepsConfiguredText(t *testing.T) {
	prior := types.StringValue("{\n  \"type\": \"object\",\n  \"$id\": \"https://x/y\"\n}")
	var state catalogSchemaResourceModel

	if err := setCatalogSchemaState(&state, prior, client.CatalogSchema(`{"$id":"https://x/y","type":"object"}`)); err != nil {
		t.Fatalf("setCatalogSchemaState: %v", err)
	}
	if !state.Schema.Equal(prior) {
		t.Fatalf("configured schema text must be kept, got %v", state.Schema)
	}
	if state.SchemaID.ValueString() != "https://x/y" {
		t.Fatalf("unexpected schema_id %v", state.SchemaID)
	}
}

func TestSetCatalogSchemaStateTakesDriftFromAPI(t *testing.T) {
	var state catalogSchemaResourceModel
	if err := setCatalogSchemaState(&state, types.StringValue(`{"$id":"a"}`), client.CatalogSchema(`{"$id":"b"}`)); err != nil {
		t.Fatalf("setCatalogSchemaState: %v", err)
	}
	if state.Schema.ValueString() != `{"$id":"b"}` || state.SchemaID.ValueString() != "b" {
		t.Fatalf("API value must win on drift, got %v / %v", state.Schema, state.SchemaID)
	}
}

func TestSetCatalogSchemaStateAfterImport(t *testing.T) {
	var state catalogSchemaResourceModel
	if err := setCatalogSchemaState(&state, types.StringNull(), client.CatalogSchema(`{"$id":"u"}`)); err != nil {
		t.Fatalf("setCatalogSchemaState: %v", err)
	}
	if state.Schema.ValueString() != `{"$id":"u"}` {
		t.Fatalf("unexpected schema %v", state.Schema)
	}
}

func TestCatalogSchemaResourceSaveAndRead(t *testing.T) {
	var stored []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			stored, _ = io.ReadAll(r.Body)
			w.Write([]byte(`{"name":"s"}`))
		case http.MethodGet:
			if stored == nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			var doc map[string]any
			_ = json.Unmarshal(stored, &doc)
			doc["name"], doc["status"] = "s", "APPLIED"
			_ = json.NewEncoder(w).Encode(doc)
		}
	}))
	defer srv.Close()
	c, err := client.NewClient(srv.URL, "k")
	if err != nil {
		t.Fatal(err)
	}
	r := &catalogSchemaResource{client: c}

	plan := catalogSchemaResourceModel{
		Bucket: types.StringValue("platform"),
		Path:   types.StringValue("model"),
		Schema: types.StringValue(catalogSchemaForTest),
	}
	var diags diagCollector
	if !r.save(context.Background(), &plan, "create", &diags.Diagnostics) {
		t.Fatalf("save failed: %v", diags.Diagnostics)
	}
	if plan.ID.ValueString() != "platform/model" {
		t.Fatalf("unexpected id %v", plan.ID)
	}
	if plan.SchemaID.ValueString() != "https://dial.epam.com/catalog-schemas/model" {
		t.Fatalf("unexpected schema_id %v", plan.SchemaID)
	}
	if plan.Schema.ValueString() != catalogSchemaForTest {
		t.Fatalf("configured schema must be kept, got %v", plan.Schema)
	}
	assertJSONEqual(t, stored, catalogSchemaForTest)
}

func TestParseCatalogSchemaReturnsID(t *testing.T) {
	_, id, err := parseCatalogSchema(catalogSchemaForTest)
	if err != nil || id != "https://dial.epam.com/catalog-schemas/model" {
		t.Fatalf("unexpected result %q, %v", id, err)
	}
}

type diagCollector struct{ Diagnostics diag.Diagnostics }
