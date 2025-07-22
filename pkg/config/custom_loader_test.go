package config_test

import (
	"os"
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test configuration structs for custom env loading
type CustomEnvConfig struct {
	TestString    string   `env:"TEST_CUSTOM_STRING"`
	TestInt       int      `env:"TEST_CUSTOM_INT"`
	TestBool      bool     `env:"TEST_CUSTOM_BOOL"`
	TestArray     []string `env:"TEST_CUSTOM_ARRAY" envSeparator:","`
	TestWithQuote string   `env:"TEST_CUSTOM_WITH_QUOTES"`
	TestEmpty     string   `env:"TEST_CUSTOM_EMPTY"`
	TestPriority  string   `env:"TEST_PRIORITY"`
}

type OverrideConfig struct {
	TestUnique    string `env:"TEST_OVERRIDE_UNIQUE"`
	TestMultiEnv  string `env:"TEST_MULTIENV_FEATURE"`
	TestOverriden string `env:"TEST_CUSTOM_STRING"`
}

type RequiredEnvConfig struct {
	Required string `env:"OVERRIDDEN_REQUIRED,required"`
}

func TestLoadEnv_CustomPath(t *testing.T) {
	// Unset environment variables to ensure test clarity
	os.Unsetenv("TEST_CUSTOM_STRING")
	os.Unsetenv("TEST_CUSTOM_INT")
	os.Unsetenv("TEST_CUSTOM_BOOL")
	os.Unsetenv("TEST_CUSTOM_ARRAY")
	os.Unsetenv("TEST_CUSTOM_WITH_QUOTES")
	os.Unsetenv("TEST_CUSTOM_EMPTY")
	os.Unsetenv("TEST_PRIORITY")
	config.ResetCache()

	// Load environment from custom path
	err := config.LoadEnv("testdata/.env.custom")
	require.NoError(t, err, "LoadEnv should not return error with valid file")

	// Verify environment variables were loaded
	var cfg CustomEnvConfig
	err = config.Load(&cfg)
	require.NoError(t, err, "Load should successfully parse config after LoadEnv")

	// Assert values from custom env file
	assert.Equal(t, "custom_value", cfg.TestString)
	assert.Equal(t, 1234, cfg.TestInt)
	assert.Equal(t, true, cfg.TestBool)
	assert.Equal(t, []string{"item1", "item2", "item3"}, cfg.TestArray)
	assert.Equal(t, "quoted value", cfg.TestWithQuote)
	assert.Equal(t, "", cfg.TestEmpty)
	assert.Equal(t, "custom_file_value", cfg.TestPriority)
}

func TestLoadEnv_MultiplePaths(t *testing.T) {
	// Clear all environment variables and cache for a clean test
	os.Unsetenv("TEST_CUSTOM_STRING")
	os.Unsetenv("TEST_CUSTOM_INT")
	os.Unsetenv("TEST_CUSTOM_BOOL")
	os.Unsetenv("TEST_CUSTOM_ARRAY")
	os.Unsetenv("TEST_CUSTOM_WITH_QUOTES")
	os.Unsetenv("TEST_CUSTOM_EMPTY")
	os.Unsetenv("TEST_PRIORITY")
	os.Unsetenv("TEST_OVERRIDE_UNIQUE")
	os.Unsetenv("TEST_MULTIENV_FEATURE")
	os.Unsetenv("OVERRIDDEN_REQUIRED")
	config.ResetCache()

	// Load multiple files in one call (order matters for precedence)
	err := config.LoadEnv("testdata/.env.custom", "testdata/.env.override")
	require.NoError(t, err, "LoadEnv should not return error with valid files")

	// Load custom config
	var customCfg CustomEnvConfig
	err = config.Load(&customCfg)
	require.NoError(t, err)

	// Values from override should take precedence
	assert.Equal(t, "override_value", customCfg.TestString)
	assert.Equal(t, 9999, customCfg.TestInt)
	assert.Equal(t, "override_value", customCfg.TestPriority)

	// Load override config to verify unique values
	var overrideCfg OverrideConfig
	err = config.Load(&overrideCfg)
	require.NoError(t, err)

	assert.Equal(t, "unique_to_override", overrideCfg.TestUnique)
	assert.Equal(t, "enabled", overrideCfg.TestMultiEnv)
	assert.Equal(t, "override_value", overrideCfg.TestOverriden)
}

func TestLoadEnv_NonExistentPath(t *testing.T) {
	// Try to load from non-existent file
	err := config.LoadEnv("testdata/non_existent_file.env")
	require.Error(t, err, "LoadEnv should return error with non-existent file")
}

func TestMustLoadEnv(t *testing.T) {
	// Test successful loading
	assert.NotPanics(t, func() {
		config.MustLoadEnv("testdata/.env.custom")
	}, "MustLoadEnv should not panic with valid file")

	// Test panic with non-existent file
	assert.Panics(t, func() {
		config.MustLoadEnv("testdata/non_existent_file.env")
	}, "MustLoadEnv should panic with non-existent file")
}

func TestLoadEnv_WithRequiredConfig(t *testing.T) {
	// Start with a clean environment and cache
	os.Unsetenv("TEST_CUSTOM_STRING")
	os.Unsetenv("TEST_CUSTOM_INT")
	os.Unsetenv("TEST_CUSTOM_BOOL")
	os.Unsetenv("TEST_CUSTOM_ARRAY")
	os.Unsetenv("TEST_CUSTOM_WITH_QUOTES")
	os.Unsetenv("TEST_CUSTOM_EMPTY")
	os.Unsetenv("TEST_PRIORITY")
	os.Unsetenv("TEST_OVERRIDE_UNIQUE")
	os.Unsetenv("TEST_MULTIENV_FEATURE")
	os.Unsetenv("OVERRIDDEN_REQUIRED")
	config.ResetCache()

	// This should fail without loading the env file
	var requiredCfg RequiredEnvConfig
	err := config.Load(&requiredCfg)
	require.Error(t, err, "Load should error when required field is missing")

	// Now directly set the environment variable
	t.Setenv("OVERRIDDEN_REQUIRED", "required_value")

	// Force reload of this config type since env vars changed
	var requiredCfg2 RequiredEnvConfig
	err = config.ForceReloadConfig(&requiredCfg2)
	require.NoError(t, err, "Load should succeed after setting required value")
	assert.Equal(t, "required_value", requiredCfg2.Required)
}

func TestLoadEnv_DefaultBehavior(t *testing.T) {
	// Create a temporary .env file in the current directory
	tmpEnv := ".env"

	// Clear cache before test
	config.ResetCache()

	// Backup the existing .env file if it exists
	oldEnvContent, readErr := os.ReadFile(tmpEnv)
	hasOldFile := !os.IsNotExist(readErr)

	// Ensure cleanup after test
	defer func() {
		// Remove our test .env file
		os.Remove(tmpEnv)

		// Restore the original .env file if it existed
		if hasOldFile {
			_ = os.WriteFile(tmpEnv, oldEnvContent, 0644)
		}

		// Clean the environment variable
		os.Unsetenv("DEFAULT_ENV_VAR")
	}()

	err := os.WriteFile(tmpEnv, []byte("DEFAULT_ENV_VAR=default_from_temp"), 0644)
	require.NoError(t, err, "Failed to create temporary .env file")

	// Unset the variable to ensure clean test
	os.Unsetenv("DEFAULT_ENV_VAR")

	// Call LoadEnv with no arguments should load the default .env
	err = config.LoadEnv()
	require.NoError(t, err)

	// Check if the variable was loaded
	assert.Equal(t, "default_from_temp", os.Getenv("DEFAULT_ENV_VAR"))
}
