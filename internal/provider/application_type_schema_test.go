package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const applicationTypeSchemaForTest = `{"$id":"https://dial.epam.com/application_type_schemas/quick","type":"object","properties":{"prompt":{"type":"string"}}}`

func TestApplicationTypeSchemaTypesRegistered(t *testing.T) {
	ctx := context.Background()
	resources := map[string]bool{}
	for _, factory := range (&AidialProvider{}).Resources(ctx) {
		resp := &resource.MetadataResponse{}
		factory().Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "aidial"}, resp)
		resources[resp.TypeName] = true
	}
	if !resources["aidial_application_type_schema"] {
		t.Error("aidial_application_type_schema resource is not registered")
	}
	dataSources := map[string]bool{}
	for _, factory := range (&AidialProvider{}).DataSources(ctx) {
		resp := &datasource.MetadataResponse{}
		factory().Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "aidial"}, resp)
		dataSources[resp.TypeName] = true
	}
	for _, name := range []string{"aidial_application_type_schema", "aidial_file_application_type_schema"} {
		if !dataSources[name] {
			t.Errorf("%s data source is not registered", name)
		}
	}
}

func TestApplicationTypeSchemaValidator(t *testing.T) {
	for name, tc := range map[string]struct {
		value   types.String
		wantErr bool
	}{
		"valid":      {types.StringValue(applicationTypeSchemaForTest), false},
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
		applicationTypeSchemaValidator{}.ValidateString(context.Background(), validator.StringRequest{ConfigValue: tc.value}, resp)
		if resp.Diagnostics.HasError() != tc.wantErr {
			t.Fatalf("%s: error=%v, want %v", name, resp.Diagnostics.HasError(), tc.wantErr)
		}
	}
}

func TestApplicationTypeSchemaResourceSchemaIsValid(t *testing.T) {
	resp := &resource.SchemaResponse{}
	(&applicationTypeSchemaResource{}).Schema(context.Background(), resource.SchemaRequest{}, resp)
	if diags := resp.Schema.ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatalf("invalid schema: %v", diags)
	}
}

func TestSetApplicationTypeSchemaStateKeepsConfiguredText(t *testing.T) {
	prior := types.StringValue("{\n  \"type\": \"object\",\n  \"$id\": \"https://x/y\"\n}")
	var state applicationTypeSchemaResourceModel

	if err := setApplicationTypeSchemaState(&state, prior, client.ApplicationTypeSchema(`{"$id":"https://x/y","type":"object"}`)); err != nil {
		t.Fatalf("setApplicationTypeSchemaState: %v", err)
	}
	if !state.Schema.Equal(prior) {
		t.Fatalf("configured schema text must be kept, got %v", state.Schema)
	}
	if state.SchemaID.ValueString() != "https://x/y" {
		t.Fatalf("unexpected schema_id %v", state.SchemaID)
	}
}

func TestSetApplicationTypeSchemaStateTakesDriftFromAPI(t *testing.T) {
	var state applicationTypeSchemaResourceModel
	if err := setApplicationTypeSchemaState(&state, types.StringValue(`{"$id":"a"}`), client.ApplicationTypeSchema(`{"$id":"b"}`)); err != nil {
		t.Fatalf("setApplicationTypeSchemaState: %v", err)
	}
	if state.Schema.ValueString() != `{"$id":"b"}` || state.SchemaID.ValueString() != "b" {
		t.Fatalf("API value must win on drift, got %v / %v", state.Schema, state.SchemaID)
	}
}

func TestSetApplicationTypeSchemaStateAfterImport(t *testing.T) {
	var state applicationTypeSchemaResourceModel
	if err := setApplicationTypeSchemaState(&state, types.StringNull(), client.ApplicationTypeSchema(`{"$id":"u"}`)); err != nil {
		t.Fatalf("setApplicationTypeSchemaState: %v", err)
	}
	if state.Schema.ValueString() != `{"$id":"u"}` {
		t.Fatalf("unexpected schema %v", state.Schema)
	}
}

func TestApplicationTypeSchemaResourceSaveAndRead(t *testing.T) {
	var stored []byte
	var putPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			putPath = r.URL.Path
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
	c, err := client.NewClient(srv.URL, client.APIKeyCredentials("k"))
	if err != nil {
		t.Fatal(err)
	}
	r := &applicationTypeSchemaResource{client: c}

	plan := applicationTypeSchemaResourceModel{
		Bucket: types.StringValue("platform"),
		Path:   types.StringValue("quick"),
		Schema: types.StringValue(applicationTypeSchemaForTest),
	}
	var diags diagCollector
	if !r.save(context.Background(), &plan, "create", &diags.Diagnostics) {
		t.Fatalf("save failed: %v", diags.Diagnostics)
	}
	if putPath != "/v1/schemas/platform/quick" {
		t.Fatalf("unexpected PUT path %q", putPath)
	}
	if plan.ID.ValueString() != "platform/quick" {
		t.Fatalf("unexpected id %v", plan.ID)
	}
	if plan.SchemaID.ValueString() != "https://dial.epam.com/application_type_schemas/quick" {
		t.Fatalf("unexpected schema_id %v", plan.SchemaID)
	}
	if plan.Schema.ValueString() != applicationTypeSchemaForTest {
		t.Fatalf("configured schema must be kept, got %v", plan.Schema)
	}
	assertJSONEqual(t, stored, applicationTypeSchemaForTest)
}

func TestApplicationTypeSchemaResourceSaveFailsWhenMissingAfterPut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.Write([]byte(`{"name":"s"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	c, _ := client.NewClient(srv.URL, client.APIKeyCredentials("k"))
	r := &applicationTypeSchemaResource{client: c}

	plan := applicationTypeSchemaResourceModel{
		Bucket: types.StringValue("platform"),
		Path:   types.StringValue("quick"),
		Schema: types.StringValue(applicationTypeSchemaForTest),
	}
	var diags diagCollector
	if r.save(context.Background(), &plan, "create", &diags.Diagnostics) || !diags.Diagnostics.HasError() {
		t.Fatal("expected an error when the schema is missing after PUT")
	}
}

func TestApplicationTypeSchemaImportRejectsBadID(t *testing.T) {
	r := &applicationTypeSchemaResource{}
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	objType := resp.Schema.Type().TerraformType(context.Background())
	for _, id := range []string{"nobucket", "/path", "bucket/"} {
		importResp := &resource.ImportStateResponse{State: tfsdk.State{Schema: resp.Schema, Raw: tftypes.NewValue(objType, nil)}}
		r.ImportState(context.Background(), resource.ImportStateRequest{ID: id}, importResp)
		if !importResp.Diagnostics.HasError() {
			t.Errorf("import ID %q must be rejected", id)
		}
	}
}

// readSchemaDataSource runs Read of ds for a config holding the given string
// arguments (all other attributes null).
func readSchemaDataSource(t *testing.T, ds datasource.DataSource, args map[string]string) *datasource.ReadResponse {
	t.Helper()
	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	values := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, typ := range objType.AttributeTypes {
		values[name] = tftypes.NewValue(typ, nil)
	}
	for name, value := range args {
		values[name] = tftypes.NewValue(tftypes.String, value)
	}
	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, values)}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)}}
	ds.(datasource.DataSourceWithConfigure).Read(ctx, req, resp)
	return resp
}

func TestApplicationTypeSchemaDataSourcePopulatesState(t *testing.T) {
	var gotPath string
	c := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte(`{"$id":"https://x/y","type":"object","name":"quick","status":"APPLIED"}`))
	})

	resp := readSchemaDataSource(t, &applicationTypeSchemaDataSource{client: c}, map[string]string{"bucket": "platform", "path": "quick"})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if gotPath != "/v1/schemas/platform/quick" {
		t.Fatalf("unexpected request path %q", gotPath)
	}
	if got := stateString(t, resp, "id"); got.ValueString() != "platform/quick" {
		t.Errorf("id = %v", got)
	}
	if got := stateString(t, resp, "schema_id"); got.ValueString() != "https://x/y" {
		t.Errorf("schema_id = %v", got)
	}
	if got := stateString(t, resp, "schema"); strings.Contains(got.ValueString(), "status") || strings.Contains(got.ValueString(), `"name"`) {
		t.Errorf("schema must not contain metadata: %v", got)
	}
}

func TestApplicationTypeSchemaDataSourceNotFound(t *testing.T) {
	c := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })

	resp := readSchemaDataSource(t, &applicationTypeSchemaDataSource{client: c}, map[string]string{"bucket": "platform", "path": "missing"})
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics[0].Summary(), "not found") {
		t.Fatalf("expected a not-found diagnostic, got %v", resp.Diagnostics)
	}
}

func TestFileApplicationTypeSchemaDataSourcePopulatesState(t *testing.T) {
	var gotPath string
	c := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.Write([]byte(`{"$id":"https://x/y","type":"object","name":"a/b","status":"APPLIED"}`))
	})

	resp := readSchemaDataSource(t, &fileApplicationTypeSchemaDataSource{client: c}, map[string]string{"name": "a/b"})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if gotPath != "/v1/admin/config/file/schemas/a%2Fb" {
		t.Fatalf("unexpected request path %q", gotPath)
	}
	for name, want := range map[string]string{"name": "a/b", "schema_id": "https://x/y", "status": "APPLIED"} {
		if got := stateString(t, resp, name); got.ValueString() != want {
			t.Errorf("%s = %v, want %q", name, got, want)
		}
	}
	assertJSONEqual(t, []byte(stateString(t, resp, "schema").ValueString()), `{"$id":"https://x/y","type":"object"}`)
}

func TestFileApplicationTypeSchemaDataSourceStatusNullWhenAbsent(t *testing.T) {
	c := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"$id":"u"}`)) })

	resp := readSchemaDataSource(t, &fileApplicationTypeSchemaDataSource{client: c}, map[string]string{"name": "n"})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if got := stateString(t, resp, "status"); !got.IsNull() {
		t.Errorf("status must be null when Core sends none, got %v", got)
	}
}

func TestFileApplicationTypeSchemaDataSourceNotFound(t *testing.T) {
	c := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })

	resp := readSchemaDataSource(t, &fileApplicationTypeSchemaDataSource{client: c}, map[string]string{"name": "missing"})
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics[0].Summary(), "not found") {
		t.Fatalf("expected a not-found diagnostic, got %v", resp.Diagnostics)
	}
}

func TestFileApplicationTypeSchemaDataSourceNameRequired(t *testing.T) {
	resp := &datasource.SchemaResponse{}
	(&fileApplicationTypeSchemaDataSource{}).Schema(context.Background(), datasource.SchemaRequest{}, resp)
	name, ok := resp.Schema.Attributes["name"]
	if !ok || !name.IsRequired() {
		t.Fatal("name must be a required attribute")
	}
	for _, absent := range []string{"bucket", "path", "id"} {
		if _, ok := resp.Schema.Attributes[absent]; ok {
			t.Errorf("file data source must not have %q", absent)
		}
	}
}
