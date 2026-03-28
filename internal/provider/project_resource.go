package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"github.com/quantcdn/terraform-provider-quant/internal/client"
	"github.com/quantcdn/terraform-provider-quant/internal/mapper"
	"github.com/quantcdn/terraform-provider-quant/internal/resource_project"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
)

var (
	_ resource.Resource                = (*projectResource)(nil)
	_ resource.ResourceWithConfigure   = (*projectResource)(nil)
	_ resource.ResourceWithImportState = (*projectResource)(nil)
)

// APIError represents the structure of API error responses
type APIError struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

// extractAPIErrorMessage attempts to parse the API error response body
func extractAPIErrorMessage(resp *http.Response, fallbackErr error) string {
	if resp == nil || resp.Body == nil {
		return fallbackErr.Error()
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fallbackErr.Error()
	}

	// Try to parse as JSON error response
	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err != nil {
		// If JSON parsing fails, return the original error
		return fallbackErr.Error()
	}

	// If we successfully parsed the JSON and have a message, use it
	if apiErr.Message != "" {
		return apiErr.Message
	}

	// Fallback to original error
	return fallbackErr.Error()
}

func NewProjectResource() resource.Resource {
	return &projectResource{}
}

type projectResource struct {
	client *client.Client
}

func (r *projectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *projectResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := resource_project.ProjectResourceSchema(ctx)
	addUseStateForUnknown(s.Attributes)
	resp.Schema = s
}

func (r *projectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected resource configure type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
	}

	r.client = client
}

func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_project.ProjectModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Create the project and wait for it to be ready (includes polling)
	resp.Diagnostics.Append(callProjectCreateAPI(ctx, r, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read the final API results back into the model for Terraform state
	// This ensures we have the most up-to-date information after creation is complete
	resp.Diagnostics.Append(callProjectReadAPI(ctx, r, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_project.ProjectModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read API call logic
	resp.Diagnostics.Append(callProjectReadAPI(ctx, r, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_project.ProjectModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var stateData resource_project.ProjectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	data.MachineName = stateData.MachineName

	// Update API call logic
	resp.Diagnostics.Append(callProjectUpdateAPI(ctx, r, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callProjectReadAPI(ctx, r, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_project.ProjectModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete API call logic
	resp.Diagnostics.Append(callProjectDeleteAPI(ctx, r, &data)...)
}

// Import state for a given machine name.
func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data resource_project.ProjectModel
	data.MachineName = types.StringValue(req.ID)

	// Read API call logic
	resp.Diagnostics.Append(callProjectReadAPI(ctx, r, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Create project request.
func callProjectCreateAPI(ctx context.Context, r *projectResource, project *resource_project.ProjectModel) (diags diag.Diagnostics) {
	if project.Name.IsNull() || project.Name.IsUnknown() {
		diags.AddAttributeError(
			path.Root("name"),
			"Missing project.name attribute",
			"Cannot create a project without a name.",
		)
		return
	}

	if project.BasicAuthUsername.IsNull() && !project.BasicAuthPassword.IsNull() {
		diags.AddError(
			"Missing basic authentication username",
			"To enable basic authentication both username and password must be provided.",
		)
		return
	}

	if !project.BasicAuthUsername.IsNull() && project.BasicAuthPassword.IsNull() {
		diags.AddError(
			"Missing basic authentication password",
			"To enable basic authentication both username and password must be provided.",
		)
		return
	}

	// Default region to "au" if not set.
	if project.Region.IsNull() || project.Region.IsUnknown() {
		project.Region = types.StringValue("au")
	}

	// Map TF model fields to SDK request via reflection.
	req := quantadmingo.NewV2ProjectRequestWithDefaults()
	diags.Append(mapper.ToSDK(ctx, project, req)...)
	if diags.HasError() {
		return
	}

	res, resp, err := r.client.Instance.ProjectsAPI.ProjectsCreate(r.client.AuthContext, r.client.Organization).V2ProjectRequest(*req).Execute()

	if err != nil {
		if resp != nil {
			// Check for authentication/authorization errors (401/403)
			if resp.StatusCode == 401 {
				diags.AddError(
					"Authentication Failed",
					"Failed to authenticate with the Quant API. Please check your API token is valid.",
				)
				return
			}
			if resp.StatusCode == 403 {
				diags.AddError(
					"Authorization Failed",
					"Your API token does not have sufficient permissions to create projects.",
				)
				return
			}
			// Handle bad request (400) - usually validation errors
			if resp.StatusCode == 400 {
				apiErrorMsg := extractAPIErrorMessage(resp, err)
				diags.AddError(
					"Invalid Project Configuration",
					apiErrorMsg,
				)
				return
			}
			// Handle conflict (409) which likely means project already exists
			if resp.StatusCode == 409 {
				apiErrorMsg := extractAPIErrorMessage(resp, err)
				diags.AddError(
					"Project Already Exists",
					apiErrorMsg,
				)
				return
			}
		}
		// Generic error fallback
		diags.AddError(
			"Unable to Create Project",
			fmt.Sprintf("An unexpected error occurred: %s", err.Error()),
		)
		return
	}

	project.MachineName = types.StringValue(res.GetMachineName())

	// The API returns 200 with project data for successful creation request,
	// but the actual project provisioning is asynchronous in the backend.
	// We need to poll until the project is fully available.
	createStateConf := retry.StateChangeConf{
		Pending: []string{"creating", "pending", "not_found"},
		Target:  []string{"ready", "active"},
		Refresh: func() (interface{}, string, error) {
			withToken := false
			if !project.WithToken.IsNull() {
				withToken = project.WithToken.ValueBool()
			}

			projectResult, resp, err := r.client.Instance.ProjectsAPI.ProjectsRead(r.client.AuthContext, r.client.Organization, project.MachineName.ValueString()).WithToken(withToken).Execute()
			if err != nil {
				// If we get a 404, the project is still being created
				if resp != nil && resp.StatusCode == 404 {
					// Project not found yet, still creating
					return nil, "not_found", nil
				}
				// For other errors, return the error to stop polling
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				return nil, "", fmt.Errorf("error checking project status (HTTP %d): %v", statusCode, err)
			}

			// Project exists and can be read successfully
			return projectResult, "ready", nil
		},
		Timeout:      10 * time.Minute,
		Delay:        5 * time.Second,
		MinTimeout:   3 * time.Second,
		PollInterval: 5 * time.Second,
	}

	_, err = createStateConf.WaitForStateContext(ctx)
	if err != nil {
		diags.AddError(
			"Project creation timeout",
			fmt.Sprintf("Project was created but did not become ready within the timeout period. This may indicate the project is still being provisioned. Error: %s", err.Error()),
		)
	}

	return
}

func callProjectUpdateAPI(ctx context.Context, r *projectResource, project *resource_project.ProjectModel) (diags diag.Diagnostics) {
	if project.MachineName.IsNull() || project.MachineName.IsUnknown() {
		diags.AddAttributeError(
			path.Root("machine_name"),
			"Missing project.machine_name attribute",
			"To read project information the project machine name needs to be known, please import the terraform state.",
		)
		return diags
	}

	if project.BasicAuthUsername.IsNull() && !project.BasicAuthPassword.IsNull() {
		diags.AddError(
			"Missing basic authentication username",
			"To enable basic authentication both username and password must be provided.",
		)
		return diags
	}

	if !project.BasicAuthUsername.IsNull() && project.BasicAuthPassword.IsNull() {
		diags.AddError(
			"Missing basic authentication password",
			"To enable basic authentication both username and password must be provided.",
		)
		return diags
	}

	// Map TF model fields to SDK request via reflection.
	req := quantadmingo.NewV2ProjectRequestWithDefaults()
	diags.Append(mapper.ToSDK(ctx, project, req)...)
	if diags.HasError() {
		return diags
	}

	api := r.client.Instance.ProjectsAPI.ProjectsUpdate(r.client.AuthContext, r.client.Organization, project.MachineName.ValueString())
	_, _, err := api.V2ProjectRequest(*req).Execute()

	if err != nil {
		diags.AddError("Unable to update project", fmt.Sprintf("Error: %s", err.Error()))
	}

	return
}

func callProjectReadAPI(ctx context.Context, r *projectResource, project *resource_project.ProjectModel) (diags diag.Diagnostics) {
	if project.MachineName.IsNull() || project.MachineName.IsUnknown() {
		diags.AddError(
			"Unable to read project",
			"The project machine name is unknown, this may indicate an issue with your local state.",
		)
		return
	}

	withToken := false
	if !project.WithToken.IsNull() {
		withToken = project.WithToken.ValueBool()
	}
	api, _, err := r.client.Instance.ProjectsAPI.ProjectsRead(r.client.AuthContext, r.client.Organization, project.MachineName.ValueString()).WithToken(withToken).Execute()

	if err != nil {
		diags.Append(diag.NewErrorDiagnostic(
			"Unable to read project data from API",
			fmt.Sprintf("There was an issue with the request when requesting project information from the API, please check the error and update your configuration.\nError: %s", err.Error()),
		))
		return diags
	}

	// Map SDK response fields to TF model via reflection.
	// This covers: Id, Uuid, Name, MachineName, WriteToken.
	diags.Append(mapper.FromSDK(ctx, api, project)...)

	// WriteToken is optional in the API response — set to null when not returned.
	if api.WriteToken == nil {
		project.WriteToken = types.StringNull()
	}

	// Resolve any unknown optional/computed fields to null so Terraform state is consistent.
	if project.AllowQueryParams.IsUnknown() {
		project.AllowQueryParams = types.BoolNull()
	}
	if project.BasicAuthPassword.IsUnknown() {
		project.BasicAuthPassword = types.StringNull()
	}
	if project.BasicAuthUsername.IsUnknown() {
		project.BasicAuthUsername = types.StringNull()
	}
	if project.DisableRevisions.IsUnknown() {
		project.DisableRevisions = types.BoolNull()
	}
	if project.Project.IsUnknown() {
		project.Project = types.StringNull()
	}
	if project.Organization.IsNull() || project.Organization.IsUnknown() {
		project.Organization = types.StringNull()
	}

	return diags
}

func callProjectDeleteAPI(ctx context.Context, r *projectResource, project *resource_project.ProjectModel) (diags diag.Diagnostics) {
	if project.MachineName.IsNull() || project.MachineName.IsUnknown() {
		diags.AddAttributeError(
			path.Root("machine_name"),
			"Missing project.machine_name attribute",
			"To delete project information the project machine name needs to be known, please import the terraform state.",
		)
		return
	}

	_, err := r.client.Instance.ProjectsAPI.ProjectsDelete(r.client.AuthContext, r.client.Organization, project.MachineName.ValueString()).Execute()

	if err != nil {
		diags.AddError(
			"Unable to delete project",
			fmt.Sprintf("Error: %s", err.Error()),
		)
		return
	}

	// API returns 200 for successful deletion, so no polling needed
	return
}
