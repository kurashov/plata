package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setRequiredEnv sets the minimum set of env vars so Load() does not fail on
// required validation. Individual tests override specific vars on top of this.
func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DB_URL", "postgres://x:x@localhost/x?sslmode=disable")
	t.Setenv("EXCHANGE_API_KEY", "key")
}

func TestLoad_AppliesDefaults(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "8080", cfg.HTTPPort)
	assert.Equal(t, "https://api.exchangeratesapi.io/v1", cfg.ExchangeBaseURL)
	assert.Equal(t, 4, cfg.WorkerCount)
	assert.Equal(t, 10*time.Second, cfg.UpdateTimeout)
}

func TestLoad_ReadsOverrides(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("HTTP_PORT", "9090")
	t.Setenv("WORKER_COUNT", "8")
	t.Setenv("UPDATE_TIMEOUT", "30s")
	t.Setenv("EXCHANGE_BASE_URL", "http://example.test/v1")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "9090", cfg.HTTPPort)
	assert.Equal(t, 8, cfg.WorkerCount)
	assert.Equal(t, 30*time.Second, cfg.UpdateTimeout)
	assert.Equal(t, "http://example.test/v1", cfg.ExchangeBaseURL)
}

func TestLoad_FailsWhenDBURLMissing(t *testing.T) {
	t.Setenv("EXCHANGE_API_KEY", "key")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DB_URL")
}

func TestLoad_FailsWhenAPIKeyMissing(t *testing.T) {
	t.Setenv("DB_URL", "postgres://x:x@localhost/x?sslmode=disable")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "EXCHANGE_API_KEY")
}

func TestLoad_FailsOnInvalidDuration(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("UPDATE_TIMEOUT", "not-a-duration")

	_, err := Load()
	require.Error(t, err)
}

func TestLoad_FailsOnInvalidInt(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("WORKER_COUNT", "abc")

	_, err := Load()
	require.Error(t, err)
}
