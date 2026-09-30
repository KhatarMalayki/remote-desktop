//go:build !windows

package agent

import "github.com/user/remote-desktop/internal/models"

func collectEndpointReport() models.EndpointReport {
	return models.EndpointReport{Status: "unsupported", Detail: "Inventaris endpoint ini hanya mendukung Windows"}
}

func runLockPolicy(request models.LockPolicyRequest) models.LockPolicyResult {
	return models.LockPolicyResult{ID: request.ID, Status: "unsupported", Detail: "Policy hanya mendukung Windows"}
}
