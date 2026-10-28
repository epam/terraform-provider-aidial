package provider

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Acceptance tests run against a real DIAL Core. They are skipped unless TF_ACC is set,
// so `go test ./...` needs no Docker. `make testacc` starts a throwaway Core and sets
// DIAL_BASE_URL and DIAL_API_KEY.

var accProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"aidial": providerserver.NewProtocol6WithError(New()),
}

func accPreCheck(t *testing.T) {
	t.Helper()
	for _, v := range []string{"DIAL_BASE_URL", "DIAL_API_KEY"} {
		if os.Getenv(v) == "" {
			t.Fatalf("%s must be set for acceptance tests (see `make testacc`)", v)
		}
	}
}

// accPath returns a path unique per test and run, so tests can share one Core.
func accPath(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("tf-acc-%s-%d", strings.ToLower(strings.ReplaceAll(t.Name(), "_", "-")), time.Now().UnixNano()%1_000_000)
}

func accProviderConfig() string {
	return fmt.Sprintf("provider \"aidial\" {\n  base_url = %q\n  api_key  = %q\n}\n",
		os.Getenv("DIAL_BASE_URL"), os.Getenv("DIAL_API_KEY"))
}

func accImportStep(address string) resource.TestStep {
	return resource.TestStep{ResourceName: address, ImportState: true, ImportStateVerify: true}
}

// accDelete removes a resource behind Terraform's back, as another tool or the DIAL UI would.
func accDelete(t *testing.T, url string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, strings.TrimRight(os.Getenv("DIAL_BASE_URL"), "/")+url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Api-Key", os.Getenv("DIAL_API_KEY"))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("DELETE %s: %s", url, resp.Status)
	}
}
