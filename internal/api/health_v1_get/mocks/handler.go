package mocks

// HealthCheckerMock — мок HealthChecker для тестов.
type HealthCheckerMock struct {
	HealthFunc func() error
}

// Health вызывает мок-реализацию HealthFunc.
func (m *HealthCheckerMock) Health() error {
	return m.HealthFunc()
}
