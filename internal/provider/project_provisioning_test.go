package provider

import "testing"

// TestProjectProvisioningState guards against the 45-minute create timeout on
// Fastly projects. platform_provisioning_status is null for them, so polling
// for "deployed" can never succeed and every create fails after the full
// timeout — which is what TestE2E_Project hit against staging.
func TestProjectProvisioningState(t *testing.T) {
	cases := []struct {
		name  string
		props map[string]interface{}
		want  string
	}{
		{
			name:  "fastly project with null provisioning status is ready",
			props: map[string]interface{}{"platform_mode": "fastly", "platform_provisioning_status": nil},
			want:  "deployed",
		},
		{
			name:  "fastly project with no status key at all is ready",
			props: map[string]interface{}{"platform_mode": "fastly"},
			want:  "deployed",
		},
		{
			name:  "aws project still provisioning keeps polling",
			props: map[string]interface{}{"platform_mode": "aws", "platform_provisioning_status": "provisioning"},
			want:  "provisioning",
		},
		{
			name:  "aws project with a null status keeps polling",
			props: map[string]interface{}{"platform_mode": "aws", "platform_provisioning_status": nil},
			want:  "provisioning",
		},
		{
			name:  "aws project reporting deployed is ready",
			props: map[string]interface{}{"platform_mode": "aws", "platform_provisioning_status": "deployed"},
			want:  "deployed",
		},
		{
			name:  "AWS casing is not significant",
			props: map[string]interface{}{"platform_mode": "AWS", "platform_provisioning_status": "provisioning"},
			want:  "provisioning",
		},
		{
			name:  "deployed wins even without a platform_mode",
			props: map[string]interface{}{"platform_provisioning_status": "deployed"},
			want:  "deployed",
		},
		{
			name:  "an absent platform_mode must not hang the create",
			props: map[string]interface{}{},
			want:  "deployed",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := projectProvisioningState(tc.props); got != tc.want {
				t.Errorf("projectProvisioningState(%v) = %q, want %q", tc.props, got, tc.want)
			}
		})
	}
}
