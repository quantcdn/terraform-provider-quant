package provider_test

import (
	"fmt"
	"terraform-provider-quant/internal/client"

	quantadmin "github.com/quantcdn/quant-admin-go"

	"testing"
)

func TestClient(t *testing.T) {
	token := "ycZoeUU5LArm5nQU10xSBWQcNM69qcKOUuDBqzj0PJwMo4tk1cqjmimAMc8V"
	org := "quant"

	c := client.New(token, org)
	api := c.Admin.ProjectsAPI

	project := *quantadmin.NewProjectRequest()

	project.SetName("tf-project2")
	project.SetRegion("apsoutheast-2")
	project.SetAllowQueryParams(true)
	project.SetBasicAuthPreviewOnly("false")

	res, _, err := api.OrganizationsOrganizationProjectsPost(c.Auth, org).ProjectRequest(project).Execute()

	if err != nil {
		fmt.Printf("Unexpected error: %v", err)
		// fmt.Printf("A: %v", a)
	} else {
		fmt.Println("Result:", res)
	}

}
