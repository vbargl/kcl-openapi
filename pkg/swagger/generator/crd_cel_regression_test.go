package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	crd "kcl-lang.io/kcl-openapi/pkg/kube_resource/generator"
)

// The rules are reduced from CloudNativePG's synchronous replication schema.
func TestCRDScalarAndObjectCEL(t *testing.T) {
	specs, err := crd.GetSpecs(&crd.GenOpts{Spec: "testdata/cnpg_cel_crd.yaml"})
	if err != nil || len(specs) != 1 {
		t.Fatalf("get specs: %v (%d specs)", err, len(specs))
	}
	defer os.RemoveAll(filepath.Dir(specs[0]))
	dir := t.TempDir()
	opts := &GenOpts{Spec: specs[0], Target: dir, ModelPackage: "models", ValidateSpec: false}
	if err := opts.EnsureDefaults(); err != nil {
		t.Fatal(err)
	}
	if err := Generate(opts); err != nil {
		t.Fatal(err)
	}
	model := filepath.Join(dir, "models", "widget_v1_spec_synchronous.k")
	body, err := os.ReadFile(model)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"number > 0", "len(standbyNamesPre)", "not in [None, Undefined]"} {
		if !strings.Contains(string(body), fragment) {
			t.Errorf("missing %q in generated schema:\n%s", fragment, body)
		}
	}
	kcl := lookupKCLBinary(t)
	if kcl == "" {
		t.Skip("KCL executable unavailable; generated checks inspected above")
	}
	cases := []struct {
		name, fields string
		valid        bool
	}{
		{"valid", `method = "any", number = 1`, true},
		{"zero", `method = "any", number = 0`, false},
		{"preferred empty", `method = "any", number = 1, dataDurability = "preferred"`, true},
		{"preferred with pre", `method = "any", number = 1, dataDurability = "preferred", standbyNamesPre = ["replica"]`, false},
		{"preferred with post", `method = "any", number = 1, dataDurability = "preferred", standbyNamesPost = ["replica"]`, false},
		{"required with pre", `method = "any", number = 1, dataDurability = "required", standbyNamesPre = ["replica"]`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			main := filepath.Join(t.TempDir(), "main.k")
			if err := os.WriteFile(main, []byte("result = WidgetV1SpecSynchronous {"+tc.fields+"}\n"), 0600); err != nil {
				t.Fatal(err)
			}
			out, err := exec.Command(kcl, "run", model, main).CombinedOutput()
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v error=%v output=%s", tc.valid, err, out)
			}
			if !tc.valid && !strings.Contains(string(out), "Check failed") && !strings.Contains(string(out), "Instance check failed") {
				t.Fatalf("failed for reason other than validation: %s", out)
			}
		})
	}
}

func TestCRDOptionalScalarCEL(t *testing.T) {
	specs, err := crd.GetSpecs(&crd.GenOpts{Spec: "testdata/scalar_cel_crd.yaml"})
	if err != nil || len(specs) != 1 {
		t.Fatalf("get specs: %v (%d specs)", err, len(specs))
	}
	defer os.RemoveAll(filepath.Dir(specs[0]))
	dir := t.TempDir()
	opts := &GenOpts{Spec: specs[0], Target: dir, ModelPackage: "models", ValidateSpec: false}
	if err := opts.EnsureDefaults(); err != nil {
		t.Fatal(err)
	}
	if err := Generate(opts); err != nil {
		t.Fatal(err)
	}
	model := filepath.Join(dir, "models", "widget_v1_spec.k")
	body, err := os.ReadFile(model)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "number > 0") {
		t.Fatalf("missing scalar rule: %s", body)
	}
	kcl := lookupKCLBinary(t)
	if kcl == "" {
		t.Skip("KCL executable unavailable; generated checks inspected above")
	}
	for _, tc := range []struct {
		name, fields string
		valid        bool
	}{
		{"absent", "", true}, {"valid", "number = 1", true}, {"invalid", "number = 0", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			main := filepath.Join(t.TempDir(), "main.k")
			if err := os.WriteFile(main, []byte("result = WidgetV1Spec {"+tc.fields+"}\n"), 0600); err != nil {
				t.Fatal(err)
			}
			out, err := exec.Command(kcl, "run", model, main).CombinedOutput()
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v error=%v output=%s", tc.valid, err, out)
			}
			if !tc.valid && !strings.Contains(string(out), "Check failed") && !strings.Contains(string(out), "Instance check failed") {
				t.Fatalf("failed for reason other than validation: %s", out)
			}
		})
	}
}
