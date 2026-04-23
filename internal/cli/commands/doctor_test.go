package commands

import (
	"testing"

	"github.com/jakeschepis/sageo-cli/internal/auth"
	"github.com/jakeschepis/sageo-cli/internal/common/config"
	"github.com/jakeschepis/sageo-cli/internal/state"
)

func TestRunDoctorChecks_FreshProject_HasFails(t *testing.T) {
	in := doctorInputs{
		ProjectExists: false,
		Config:        &config.Config{},
		GSCStatus:     auth.Status{Authenticated: false},
	}
	checks := runDoctorChecks(in)
	sum := summariseChecks(checks)
	if sum["fail"] == 0 {
		t.Fatalf("expected at least one fail on fresh project; got %+v", sum)
	}
	if findCheck(checks, "project_initialised").Status != checkFail {
		t.Errorf("project_initialised should fail on fresh project")
	}
	if findCheck(checks, "gsc_auth").Status != checkFail {
		t.Errorf("gsc_auth should fail when unauthenticated")
	}
	if findCheck(checks, "gsc_property").Status != checkFail {
		t.Errorf("gsc_property should fail when empty")
	}
}

func TestRunDoctorChecks_FullyConfigured_AllPass(t *testing.T) {
	in := doctorInputs{
		ProjectExists: true,
		State: &state.State{
			Site:       "https://example.com",
			BrandTerms: []string{"Example", "example.com"},
		},
		Config: &config.Config{
			GSCProperty:        "https://example.com/",
			PSIAPIKey:          "key",
			LLMProvider:        "anthropic",
			AnthropicAPIKey:    "sk-ant-x",
			DataForSEOLogin:    "u@example.com",
			DataForSEOPassword: "pw",
		},
		GSCStatus: auth.Status{Authenticated: true},
	}
	checks := runDoctorChecks(in)
	sum := summariseChecks(checks)
	if sum["fail"] != 0 || sum["warn"] != 0 {
		t.Fatalf("expected all pass; got %+v\nchecks: %+v", sum, checks)
	}
}

func TestRunDoctorChecks_Partial_MixedStatuses(t *testing.T) {
	in := doctorInputs{
		ProjectExists: true,
		State:         &state.State{Site: "https://example.com"},
		Config: &config.Config{
			GSCProperty: "https://example.com/",
			LLMProvider: "anthropic",
		},
		GSCStatus: auth.Status{Authenticated: true},
	}
	checks := runDoctorChecks(in)

	if findCheck(checks, "project_initialised").Status != checkPass {
		t.Errorf("project_initialised should pass")
	}
	if findCheck(checks, "brand_terms").Status != checkWarn {
		t.Errorf("brand_terms should warn without terms")
	}
	if findCheck(checks, "llm_provider").Status != checkWarn {
		t.Errorf("llm_provider should warn without key")
	}
	if findCheck(checks, "dataforseo_creds").Status != checkWarn {
		t.Errorf("dataforseo_creds should warn without creds")
	}
	if findCheck(checks, "psi_api_key").Status != checkPass {
		t.Errorf("psi_api_key should pass via GSC fallback, got %s",
			findCheck(checks, "psi_api_key").Status)
	}
	if findCheck(checks, "gsc_auth").Status != checkPass {
		t.Errorf("gsc_auth should pass when authenticated")
	}
	if findCheck(checks, "gsc_property").Status != checkPass {
		t.Errorf("gsc_property should pass when set")
	}
}

func TestCheckLLMProvider_OpenAIVariant(t *testing.T) {
	in := doctorInputs{
		Config: &config.Config{LLMProvider: "openai"},
	}
	c := checkLLMProvider(in)
	if c.Status != checkWarn {
		t.Fatalf("expected warn without openai key, got %s", c.Status)
	}
	if c.Fix == "" || !contains(c.Fix, "openai_api_key") {
		t.Fatalf("fix should mention openai_api_key, got %q", c.Fix)
	}
	in.Config.OpenAIAPIKey = "sk-x"
	c = checkLLMProvider(in)
	if c.Status != checkPass {
		t.Fatalf("expected pass with openai key set, got %s", c.Status)
	}
}

func findCheck(checks []doctorCheck, name string) doctorCheck {
	for _, c := range checks {
		if c.Name == name {
			return c
		}
	}
	return doctorCheck{}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
