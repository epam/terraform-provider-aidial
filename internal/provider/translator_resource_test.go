package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// fakeTranslatorCore imitates the parts of DIAL Core 0.50.1 the translator
// resource depends on, as observed against a real Core: camelCase bodies and a
// GET that adds name and status.
type fakeTranslatorCore struct {
	mu       sync.Mutex
	stored   []byte
	received []byte
	paths    []string
}

func (f *fakeTranslatorCore) client(t *testing.T) *client.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.paths = append(f.paths, r.Method+" "+r.URL.EscapedPath())
		switch r.Method {
		case http.MethodPut:
			f.stored, _ = io.ReadAll(r.Body)
			f.received = append([]byte(nil), f.stored...)
			_, _ = w.Write([]byte(`{"name":"t"}`))
		case http.MethodGet:
			if f.stored == nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			var doc map[string]any
			_ = json.Unmarshal(f.stored, &doc)
			doc["name"] = "t"
			doc["status"] = "valid"
			_ = json.NewEncoder(w).Encode(doc)
		case http.MethodDelete:
			if f.stored == nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			f.stored = nil
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	t.Cleanup(srv.Close)
	c, err := client.NewClient(srv.URL, client.APIKeyCredentials("k"))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func (f *fakeTranslatorCore) body(t *testing.T) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(f.received, &doc); err != nil {
		t.Fatalf("received body is not JSON: %v", err)
	}
	return doc
}

func translatorResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	var resp resource.SchemaResponse
	(&translatorResource{}).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	return resp
}

func newTranslatorPlan() translatorModel {
	return translatorModel{
		ID:     types.StringUnknown(),
		Bucket: types.StringValue("platform"),
		Path:   types.StringValue("t"),
		translatorBody: translatorBody{
			BaseURL: types.StringValue("http://x:8000"),
			In:      types.StringValue("openaiChatCompletions"),
			Out:     types.StringValue("anthropicMessages"),
		},
	}
}

func newTranslatorState(t *testing.T, model translatorModel) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	schema := translatorResourceSchema(t).Schema
	state := tfsdk.State{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}
	if diags := state.Set(ctx, &model); diags.HasError() {
		t.Fatalf("state: %v", diags)
	}
	return state
}

func newTranslatorPlanRequest(t *testing.T, plan translatorModel) tfsdk.Plan {
	t.Helper()
	ctx := context.Background()
	schema := translatorResourceSchema(t).Schema
	p := tfsdk.Plan{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}
	if diags := p.Set(ctx, &plan); diags.HasError() {
		t.Fatalf("plan: %v", diags)
	}
	return p
}

func emptyTranslatorState(t *testing.T) tfsdk.State {
	ctx := context.Background()
	schema := translatorResourceSchema(t).Schema
	return tfsdk.State{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}
}

func runTranslatorCreate(t *testing.T, r *translatorResource, plan translatorModel) (*resource.CreateResponse, translatorModel) {
	t.Helper()
	ctx := context.Background()
	resp := &resource.CreateResponse{State: emptyTranslatorState(t)}
	r.Create(ctx, resource.CreateRequest{Plan: newTranslatorPlanRequest(t, plan)}, resp)
	var state translatorModel
	if !resp.Diagnostics.HasError() {
		if diags := resp.State.Get(ctx, &state); diags.HasError() {
			t.Fatalf("state: %v", diags)
		}
	}
	return resp, state
}

func runTranslatorRead(t *testing.T, r *translatorResource, prior translatorModel) (*resource.ReadResponse, translatorModel) {
	t.Helper()
	ctx := context.Background()
	state := newTranslatorState(t, prior)
	resp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, resp)
	var out translatorModel
	if !resp.Diagnostics.HasError() && !resp.State.Raw.IsNull() {
		if diags := resp.State.Get(ctx, &out); diags.HasError() {
			t.Fatalf("state: %v", diags)
		}
	}
	return resp, out
}

func TestTranslatorSchemasAreValid(t *testing.T) {
	ctx := context.Background()
	rs := translatorResourceSchema(t).Schema
	if diags := rs.ValidateImplementation(ctx); diags.HasError() {
		t.Fatalf("resource schema: %v", diags)
	}
	// The model struct, including the `in` attribute, must convert to and from the schema.
	if state := newTranslatorState(t, newTranslatorPlan()); state.Raw.IsNull() {
		t.Fatal("state is null")
	}
	for _, name := range []string{"name", "api_name"} {
		if _, ok := rs.Attributes[name]; ok {
			t.Errorf("%s must not be exposed", name)
		}
	}
}

func TestTranslatorAttributesAreRequiredAndValidated(t *testing.T) {
	rs := translatorResourceSchema(t).Schema
	for _, name := range []string{"base_url", "in", "out"} {
		a := identityFromResourceSchema(t, name, rs)
		if !a.required || a.optional || a.computed {
			t.Errorf("%s must be required only: %+v", name, a)
		}
	}
	for _, name := range []string{"in", "out"} {
		a := identityFromResourceSchema(t, name, rs)
		for _, ok := range modelInterfaceTypes {
			if !validate(t, a.validators, ok) {
				t.Errorf("%s must accept %q", name, ok)
			}
		}
		for _, bad := range []string{"bogus", "", "openaichatcompletions"} {
			if validate(t, a.validators, bad) {
				t.Errorf("%s must reject %q", name, bad)
			}
		}
	}
}

func TestTranslatorCreateSendsCamelCaseAndReadsBack(t *testing.T) {
	core := &fakeTranslatorCore{}
	r := &translatorResource{client: core.client(t)}

	resp, state := runTranslatorCreate(t, r, newTranslatorPlan())
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	body := core.body(t)
	if len(body) != 3 || body["baseUrl"] != "http://x:8000" || body["in"] != "openaiChatCompletions" || body["out"] != "anthropicMessages" {
		t.Fatalf("body = %v", body)
	}
	if state.ID.ValueString() != "platform/t" || state.BaseURL.ValueString() != "http://x:8000" ||
		state.In.ValueString() != "openaiChatCompletions" || state.Out.ValueString() != "anthropicMessages" {
		t.Fatalf("unexpected state %+v", state)
	}
	if got := strings.Join(core.paths, ","); got != "PUT /v1/translators/platform/t,GET /v1/translators/platform/t" {
		t.Fatalf("create must PUT then GET, got %s", got)
	}
}

func TestTranslatorUpdateIsInPlaceAndOutOfBandDeletionRemovesState(t *testing.T) {
	core := &fakeTranslatorCore{}
	r := &translatorResource{client: core.client(t)}
	ctx := context.Background()

	resp, created := runTranslatorCreate(t, r, newTranslatorPlan())
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}

	next := created
	next.BaseURL = types.StringValue("http://y:9000")
	next.Out = types.StringValue("openaiResponses")
	upd := &resource.UpdateResponse{State: emptyTranslatorState(t)}
	r.Update(ctx, resource.UpdateRequest{Plan: newTranslatorPlanRequest(t, next)}, upd)
	if upd.Diagnostics.HasError() {
		t.Fatalf("update: %v", upd.Diagnostics)
	}
	var after translatorModel
	upd.State.Get(ctx, &after)
	if after.BaseURL.ValueString() != "http://y:9000" || after.Out.ValueString() != "openaiResponses" || core.body(t)["baseUrl"] != "http://y:9000" {
		t.Fatalf("not updated: state=%+v body=%v", after, core.body(t))
	}
	if after.ID.ValueString() != "platform/t" {
		t.Fatalf("id changed: %v", after.ID)
	}

	core.mu.Lock()
	core.stored = nil
	core.mu.Unlock()
	rr, _ := runTranslatorRead(t, r, after)
	if rr.Diagnostics.HasError() || !rr.State.Raw.IsNull() {
		t.Fatalf("a missing translator must be removed from state, diags: %v", rr.Diagnostics)
	}
}

func TestTranslatorOutOfBandChangeShowsAsDrift(t *testing.T) {
	core := &fakeTranslatorCore{}
	r := &translatorResource{client: core.client(t)}

	_, created := runTranslatorCreate(t, r, newTranslatorPlan())
	core.mu.Lock()
	core.stored = []byte(`{"baseUrl":"http://other","in":"openaiChatCompletions","out":"anthropicMessages"}`)
	core.mu.Unlock()

	rr, state := runTranslatorRead(t, r, created)
	if rr.Diagnostics.HasError() {
		t.Fatalf("read: %v", rr.Diagnostics)
	}
	if state.BaseURL.ValueString() != "http://other" {
		t.Fatalf("read must keep Core's value so the plan shows drift, got %v", state.BaseURL)
	}
}

func TestTranslatorDeleteIsIdempotent(t *testing.T) {
	core := &fakeTranslatorCore{}
	r := &translatorResource{client: core.client(t)}
	ctx := context.Background()

	_, created := runTranslatorCreate(t, r, newTranslatorPlan())
	for i := 0; i < 2; i++ { // the second delete gets a 404
		state := newTranslatorState(t, created)
		resp := &resource.DeleteResponse{State: state}
		r.Delete(ctx, resource.DeleteRequest{State: state}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("delete #%d: %v", i+1, resp.Diagnostics)
		}
	}
}

func TestTranslatorCreateReportsAPIError(t *testing.T) {
	c := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad translator"))
	})
	resp, _ := runTranslatorCreate(t, &translatorResource{client: c}, newTranslatorPlan())
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "400") ||
		!strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "bad translator") {
		t.Fatalf("expected an error with status and body, got %v", resp.Diagnostics)
	}
}

func TestTranslatorImportState(t *testing.T) {
	ctx := context.Background()
	r := &translatorResource{}

	resp := &resource.ImportStateResponse{State: emptyTranslatorState(t)}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "platform/openai-to-claude"}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("import: %v", resp.Diagnostics)
	}
	var bucket, path, id types.String
	resp.State.GetAttribute(ctx, pathRoot("bucket"), &bucket)
	resp.State.GetAttribute(ctx, pathRoot("path"), &path)
	resp.State.GetAttribute(ctx, pathRoot("id"), &id)
	if bucket.ValueString() != "platform" || path.ValueString() != "openai-to-claude" || id.ValueString() != "platform/openai-to-claude" {
		t.Fatalf("bucket=%v path=%v id=%v", bucket, path, id)
	}

	for _, bad := range []string{"nope", "/path", "bucket/", ""} {
		resp := &resource.ImportStateResponse{State: emptyTranslatorState(t)}
		r.ImportState(ctx, resource.ImportStateRequest{ID: bad}, resp)
		if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "<bucket>/<path>") {
			t.Errorf("import id %q must fail naming the expected format, got %v", bad, resp.Diagnostics)
		}
	}
}
