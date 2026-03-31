package provider_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
)

func mockAiSkillServer(t *testing.T, org string, skillId string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io"

	skillDeleted := false
	currentName := ""
	currentDescription := ""
	currentContent := ""
	currentTriggerCondition := ""
	var currentTags []string
	var currentSource map[string]interface{}
	synced := false

	skillsURL := fmt.Sprintf("%s/api/v3/organisations/%s/ai/skills", baseUrl, org)
	skillURL := fmt.Sprintf("%s/%s", skillsURL, skillId)
	importURL := fmt.Sprintf("%s/import", skillsURL)
	syncURL := fmt.Sprintf("%s/sync", skillURL)

	buildSkillResponse := func() map[string]interface{} {
		sk := map[string]interface{}{
			"id":          skillId,
			"name":        currentName,
			"namespace":   fmt.Sprintf("%s/%s", org, currentName),
			"installedAt": "2026-03-30T10:00:00Z",
			"updatedAt":   "2026-03-30T10:00:00Z",
		}
		if currentDescription != "" {
			sk["description"] = currentDescription
		}
		if currentContent != "" {
			sk["content"] = currentContent
		}
		if currentTriggerCondition != "" {
			sk["triggerCondition"] = currentTriggerCondition
		}
		if currentTags != nil {
			sk["tags"] = currentTags
		}
		if currentSource != nil {
			sk["source"] = currentSource
		}
		if synced {
			sk["updatedAt"] = "2026-03-30T12:00:00Z"
		}
		return map[string]interface{}{
			"skill": sk,
		}
	}

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	// POST create inline skill
	httpmock.RegisterResponder("POST", skillsURL,
		func(req *http.Request) (*http.Response, error) {
			skillDeleted = false
			if req.Body != nil {
				body, _ := io.ReadAll(req.Body)
				var requestBody map[string]interface{}
				if err := json.Unmarshal(body, &requestBody); err == nil {
					if name, ok := requestBody["name"].(string); ok {
						currentName = name
					}
					if desc, ok := requestBody["description"].(string); ok {
						currentDescription = desc
					}
					if content, ok := requestBody["content"].(string); ok {
						currentContent = content
					}
					if tc, ok := requestBody["triggerCondition"].(string); ok {
						currentTriggerCondition = tc
					}
					if tags, ok := requestBody["tags"].([]interface{}); ok {
						currentTags = nil
						for _, tag := range tags {
							if s, ok := tag.(string); ok {
								currentTags = append(currentTags, s)
							}
						}
					}
					currentSource = nil
				}
			}
			return httpmock.NewJsonResponse(200, buildSkillResponse())
		})

	// POST import skill
	httpmock.RegisterResponder("POST", importURL,
		func(req *http.Request) (*http.Response, error) {
			skillDeleted = false
			if req.Body != nil {
				body, _ := io.ReadAll(req.Body)
				var requestBody struct {
					Source map[string]interface{} `json:"source"`
				}
				if err := json.Unmarshal(body, &requestBody); err == nil {
					currentSource = requestBody.Source
					// Set name from source for initial response
					currentName = "imported-skill"
					currentContent = ""
				}
			}
			return httpmock.NewJsonResponse(200, buildSkillResponse())
		})

	// POST sync skill
	httpmock.RegisterResponder("POST", syncURL,
		func(req *http.Request) (*http.Response, error) {
			synced = true
			return httpmock.NewJsonResponse(200, buildSkillResponse())
		})

	// GET skill by ID
	httpmock.RegisterResponder("GET", skillURL,
		func(req *http.Request) (*http.Response, error) {
			if skillDeleted {
				return httpmock.NewStringResponse(404, "Not Found"), nil
			}
			return httpmock.NewJsonResponse(200, buildSkillResponse())
		})

	// PUT update skill
	httpmock.RegisterResponder("PUT", skillURL,
		func(req *http.Request) (*http.Response, error) {
			if req.Body != nil {
				body, _ := io.ReadAll(req.Body)
				var requestBody map[string]interface{}
				if err := json.Unmarshal(body, &requestBody); err == nil {
					if name, ok := requestBody["name"].(string); ok {
						currentName = name
					}
					if desc, ok := requestBody["description"].(string); ok {
						currentDescription = desc
					}
					if content, ok := requestBody["content"].(string); ok {
						currentContent = content
					}
					if tc, ok := requestBody["triggerCondition"].(string); ok {
						currentTriggerCondition = tc
					}
					if tags, ok := requestBody["tags"].([]interface{}); ok {
						currentTags = nil
						for _, tag := range tags {
							if s, ok := tag.(string); ok {
								currentTags = append(currentTags, s)
							}
						}
					}
				}
			}
			return httpmock.NewJsonResponse(200, buildSkillResponse())
		})

	// DELETE skill
	httpmock.RegisterResponder("DELETE", skillURL,
		func(req *http.Request) (*http.Response, error) {
			if skillDeleted {
				return httpmock.NewStringResponse(404, "Not Found"), nil
			}
			skillDeleted = true
			return httpmock.NewStringResponse(200, ""), nil
		})
}

func TestAccAiSkillInlineResource(t *testing.T) {
	org := "test-org"
	skillId := "skill-uuid-001"
	mockAiSkillServer(t, org, skillId)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create inline skill
			{
				Config: testAccAiSkillInlineConfig(org, "my-skill", "A test skill", "You are a helpful assistant."),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_ai_skill.test", "name", "my-skill"),
					resource.TestCheckResourceAttr("quant_ai_skill.test", "description", "A test skill"),
					resource.TestCheckResourceAttr("quant_ai_skill.test", "content", "You are a helpful assistant."),
					resource.TestCheckResourceAttr("quant_ai_skill.test", "id", skillId),
					resource.TestCheckResourceAttr("quant_ai_skill.test", "organization", org),
					resource.TestCheckResourceAttrSet("quant_ai_skill.test", "namespace"),
					resource.TestCheckResourceAttrSet("quant_ai_skill.test", "installed_at"),
					resource.TestCheckResourceAttrSet("quant_ai_skill.test", "updated_at"),
				),
			},
			// Step 2: Update content
			{
				Config: testAccAiSkillInlineConfig(org, "my-skill", "Updated description", "You are a very helpful assistant."),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_ai_skill.test", "description", "Updated description"),
					resource.TestCheckResourceAttr("quant_ai_skill.test", "content", "You are a very helpful assistant."),
				),
			},
			// Step 3: Destroy (implicitly tested by framework)
		},
	})
}

func TestAccAiSkillImportResource(t *testing.T) {
	org := "test-org"
	skillId := "skill-uuid-002"
	mockAiSkillServer(t, org, skillId)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create from import source
			{
				Config: testAccAiSkillImportConfig(org, "imported-skill", "github", "myorg/myrepo", "skills/helper.md", "v1.0.0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_ai_skill.test", "name", "imported-skill"),
					resource.TestCheckResourceAttr("quant_ai_skill.test", "id", skillId),
					resource.TestCheckResourceAttr("quant_ai_skill.test", "organization", org),
					resource.TestCheckResourceAttr("quant_ai_skill.test", "source.type", "github"),
					resource.TestCheckResourceAttr("quant_ai_skill.test", "source.repo", "myorg/myrepo"),
					resource.TestCheckResourceAttr("quant_ai_skill.test", "source.path", "skills/helper.md"),
					resource.TestCheckResourceAttr("quant_ai_skill.test", "source.version", "v1.0.0"),
					resource.TestCheckResourceAttrSet("quant_ai_skill.test", "namespace"),
				),
			},
			// Step 2: Destroy (implicitly tested by framework)
		},
	})
}

func testAccAiSkillInlineConfig(org string, name string, description string, content string) string {
	return fmt.Sprintf(`
provider "quant" {
  organization = %[1]q
  bearer       = "testtoken"
}

resource "quant_ai_skill" "test" {
  organization = %[1]q
  name         = %[2]q
  description  = %[3]q
  content      = %[4]q
}
`, org, name, description, content)
}

func testAccAiSkillImportConfig(org string, name string, sourceType string, repo string, path string, version string) string {
	return fmt.Sprintf(`
provider "quant" {
  organization = %[1]q
  bearer       = "testtoken"
}

resource "quant_ai_skill" "test" {
  organization = %[1]q
  name         = %[2]q

  source = {
    type    = %[3]q
    repo    = %[4]q
    path    = %[5]q
    version = %[6]q
  }
}
`, org, name, sourceType, repo, path, version)
}
