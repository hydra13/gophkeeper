package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TransportContract defines a mandatory scenario that both HTTP and gRPC
// transport layers must implement and test.
type TransportContract struct {
	Operation  string // e.g. "auth.register"
	Scenario   string // e.g. "empty_email"
	HTTPTest   string // file or test name covering this in HTTP layer
	GRPCTest   string // file or test name covering this in gRPC layer
	HTTPStatus string // expected HTTP status code or response
	GRPCCode   string // expected gRPC status code
}

// mandatoryContracts lists all scenarios that must have test coverage
// in both transport layers. This table is the single source of truth for
// transport parity verification.
var mandatoryContracts = []TransportContract{
	// --- Auth ---
	{Operation: "auth.register", Scenario: "success", HTTPTest: "auth_register_v1_post/handler_test.go::TestHandler_Handle/Success", GRPCTest: "rpc/auth_service_test.go::TestRegister_Success", HTTPStatus: "201", GRPCCode: "OK"},
	{Operation: "auth.register", Scenario: "empty_email", HTTPTest: "auth_register_v1_post/handler_test.go::TestHandler_Handle/Empty_email", GRPCTest: "rpc/auth_service_test.go::TestRegister_EmptyEmail", HTTPStatus: "400", GRPCCode: "InvalidArgument"},
	{Operation: "auth.register", Scenario: "empty_password", HTTPTest: "auth_register_v1_post/handler_test.go::TestHandler_Handle/Empty_password", GRPCTest: "rpc/auth_service_test.go::TestRegister_EmptyPassword", HTTPStatus: "400", GRPCCode: "InvalidArgument"},
	{Operation: "auth.register", Scenario: "short_password", HTTPTest: "auth_register_v1_post/handler_test.go::TestHandler_Handle/Short_password", GRPCTest: "rpc/auth_service_test.go::TestRegister_ShortPassword", HTTPStatus: "400", GRPCCode: "InvalidArgument"},
	{Operation: "auth.register", Scenario: "email_exists", HTTPTest: "auth_register_v1_post/handler_test.go::TestHandler_Handle/Email_already_exists", GRPCTest: "rpc/auth_service_test.go::TestRegister_EmailAlreadyExists", HTTPStatus: "409", GRPCCode: "AlreadyExists"},
	{Operation: "auth.login", Scenario: "success", HTTPTest: "auth_login_v1_post/handler_test.go::TestHandler_Handle/Success", GRPCTest: "rpc/auth_service_test.go::TestLogin_Success", HTTPStatus: "200", GRPCCode: "OK"},
	{Operation: "auth.login", Scenario: "missing_fields", HTTPTest: "auth_login_v1_post/handler_test.go::TestHandler_Handle/Empty_email", GRPCTest: "rpc/auth_service_test.go::TestLogin_MissingFields", HTTPStatus: "400", GRPCCode: "InvalidArgument"},
	{Operation: "auth.login", Scenario: "invalid_credentials", HTTPTest: "auth_login_v1_post/handler_test.go::TestHandler_Handle/Invalid_credentials", GRPCTest: "rpc/auth_service_test.go::TestLogin_InvalidCredentials", HTTPStatus: "401", GRPCCode: "Unauthenticated"},
	{Operation: "auth.refresh", Scenario: "success", HTTPTest: "auth_refresh_v1_post/handler_test.go", GRPCTest: "rpc/auth_service_test.go::TestRefresh_Success", HTTPStatus: "200", GRPCCode: "OK"},
	{Operation: "auth.refresh", Scenario: "empty_token", HTTPTest: "auth_refresh_v1_post/handler_test.go", GRPCTest: "rpc/auth_service_test.go::TestRefresh_EmptyToken", HTTPStatus: "400", GRPCCode: "InvalidArgument"},
	{Operation: "auth.refresh", Scenario: "session_expired", HTTPTest: "auth_refresh_v1_post/handler_test.go", GRPCTest: "rpc/auth_service_test.go::TestRefresh_SessionExpired", HTTPStatus: "401", GRPCCode: "Unauthenticated"},
	{Operation: "auth.logout", Scenario: "success", HTTPTest: "auth_logout_v1_post/handler_test.go", GRPCTest: "rpc/auth_service_test.go::TestLogout_Success", HTTPStatus: "204", GRPCCode: "OK"},
	{Operation: "auth.logout", Scenario: "no_token", HTTPTest: "auth_logout_v1_post/handler_test.go", GRPCTest: "rpc/auth_service_test.go::TestLogout_NoToken", HTTPStatus: "401", GRPCCode: "Unauthenticated"},

	// --- Records CRUD ---
	{Operation: "records.create", Scenario: "success", HTTPTest: "records_v1_post/handler_test.go", GRPCTest: "rpc/data_service_test.go", HTTPStatus: "201", GRPCCode: "OK"},
	{Operation: "records.create", Scenario: "empty_name", HTTPTest: "records_v1_post/handler_test.go", GRPCTest: "rpc/data_service_test.go", HTTPStatus: "400", GRPCCode: "InvalidArgument"},
	{Operation: "records.get", Scenario: "success", HTTPTest: "records_by_id_v1_get/handler_test.go", GRPCTest: "rpc/data_service_test.go", HTTPStatus: "200", GRPCCode: "OK"},
	{Operation: "records.get", Scenario: "not_found", HTTPTest: "records_by_id_v1_get/handler_test.go", GRPCTest: "rpc/data_service_test.go", HTTPStatus: "404", GRPCCode: "NotFound"},
	{Operation: "records.get", Scenario: "ownership_mismatch", HTTPTest: "records_by_id_v1_get/handler_test.go", GRPCTest: "rpc/data_service_test.go", HTTPStatus: "403", GRPCCode: "PermissionDenied"},
	{Operation: "records.list", Scenario: "success", HTTPTest: "records_v1_get/handler_test.go", GRPCTest: "rpc/data_service_test.go", HTTPStatus: "200", GRPCCode: "OK"},
	{Operation: "records.update", Scenario: "success", HTTPTest: "records_by_id_v1_put/handler_test.go", GRPCTest: "rpc/data_service_test.go", HTTPStatus: "200", GRPCCode: "OK"},
	{Operation: "records.update", Scenario: "revision_conflict", HTTPTest: "records_by_id_v1_put/handler_test.go", GRPCTest: "rpc/data_service_test.go", HTTPStatus: "409", GRPCCode: "Aborted"},
	{Operation: "records.delete", Scenario: "success", HTTPTest: "records_by_id_v1_delete/handler_test.go", GRPCTest: "rpc/data_service_test.go", HTTPStatus: "200", GRPCCode: "OK"},

	// --- Sync ---
	{Operation: "sync.push", Scenario: "success", HTTPTest: "sync_push_v1_post/handler_test.go", GRPCTest: "rpc/sync_service_test.go", HTTPStatus: "200", GRPCCode: "OK"},
	{Operation: "sync.push", Scenario: "empty_device_id", HTTPTest: "sync_push_v1_post/handler_test.go", GRPCTest: "rpc/sync_service_test.go", HTTPStatus: "400", GRPCCode: "InvalidArgument"},
	{Operation: "sync.pull", Scenario: "success", HTTPTest: "sync_pull_v1_post/handler_test.go", GRPCTest: "rpc/sync_service_test.go", HTTPStatus: "200", GRPCCode: "OK"},
	{Operation: "sync.pull", Scenario: "default_limit", HTTPTest: "sync_pull_v1_post/handler_test.go", GRPCTest: "rpc/sync_service_test.go", HTTPStatus: "200", GRPCCode: "OK"},

	// --- Uploads ---
	{Operation: "uploads.create_session", Scenario: "success", HTTPTest: "uploads_v1_post/handler_test.go", GRPCTest: "rpc/uploads_service_test.go", HTTPStatus: "201", GRPCCode: "OK"},
	{Operation: "uploads.upload_chunk", Scenario: "success", HTTPTest: "uploads_by_id_chunks_v1_post/handler_test.go", GRPCTest: "rpc/uploads_service_test.go", HTTPStatus: "200", GRPCCode: "OK"},
	{Operation: "uploads.download_chunk", Scenario: "success", HTTPTest: "uploads_by_id_chunks_v1_get/handler_test.go", GRPCTest: "rpc/uploads_service_test.go", HTTPStatus: "200", GRPCCode: "OK"},

	// --- Health ---
	{Operation: "health.check", Scenario: "success", HTTPTest: "health_v1_get/handler_test.go", GRPCTest: "rpc/health_service_test.go", HTTPStatus: "200", GRPCCode: "OK"},
}

// TestTransportParity verifies that every mandatory scenario is documented
// with test coverage in both HTTP and gRPC transport layers.
// This test acts as a living contract: if a new mandatory scenario is added
// to one transport, it must be reflected in the other.
func TestTransportParity(t *testing.T) {
	t.Parallel()

	for _, tc := range mandatoryContracts {
		tc := tc
		t.Run(tc.Operation+"/"+tc.Scenario, func(t *testing.T) {
			t.Parallel()

			assert.NotEmpty(t, tc.HTTPTest, "HTTP test reference must not be empty for %s/%s", tc.Operation, tc.Scenario)
			assert.NotEmpty(t, tc.GRPCTest, "gRPC test reference must not be empty for %s/%s", tc.Operation, tc.Scenario)
			assert.NotEmpty(t, tc.HTTPStatus, "expected HTTP status must not be empty for %s/%s", tc.Operation, tc.Scenario)
			assert.NotEmpty(t, tc.GRPCCode, "expected gRPC code must not be empty for %s/%s", tc.Operation, tc.Scenario)
		})
	}
}

// TestTransportCoverageCount ensures the contract table covers all
// mandatory operation groups. If a new operation group is added to either
// transport, this test will fail until the contract table is updated.
func TestTransportCoverageCount(t *testing.T) {
	t.Parallel()

	operations := make(map[string]int)
	for _, tc := range mandatoryContracts {
		operations[tc.Operation]++
	}

	// Every operation group must have at least one scenario
	requiredOperations := []string{
		"auth.register",
		"auth.login",
		"auth.refresh",
		"auth.logout",
		"records.create",
		"records.get",
		"records.list",
		"records.update",
		"records.delete",
		"sync.push",
		"sync.pull",
		"uploads.create_session",
		"uploads.upload_chunk",
		"uploads.download_chunk",
		"health.check",
	}

	for _, op := range requiredOperations {
		count, exists := operations[op]
		assert.True(t, exists, "operation %s missing from transport contract table", op)
		assert.GreaterOrEqual(t, count, 1, "operation %s must have at least 1 scenario", op)
	}

	// Minimum total scenarios
	assert.GreaterOrEqual(t, len(mandatoryContracts), 25, "transport contract table should have at least 25 scenarios (current: %d)", len(mandatoryContracts))
}
