package config_test

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/config"
)

type TestConfigDefault struct {
	TestString string `env:"TEST_STRING_DEFAULT" envDefault:"default_value"`
	TestInt    int    `env:"TEST_INT_DEFAULT" envDefault:"42"`
	TestBool   bool   `env:"TEST_BOOL_DEFAULT" envDefault:"true"`
}

type TestConfigSuccess struct {
	TestString string `env:"TEST_STRING_SUCCESS" envDefault:"default_value"`
	TestInt    int    `env:"TEST_INT_SUCCESS" envDefault:"42"`
	TestBool   bool   `env:"TEST_BOOL_SUCCESS" envDefault:"true"`
}

type TestConfigSingleton struct {
	TestString string `env:"TEST_STRING_SINGLETON" envDefault:"default_value"`
}

type TestConfigDifferent1 struct {
	Value string `env:"VALUE_TYPE1" envDefault:"default1"`
}

type TestConfigDifferent2 struct {
	Value string `env:"VALUE_TYPE2" envDefault:"default2"`
}

type RequiredConfig struct {
	Required string `env:"REQUIRED_VALUE,required"`
}

func TestLoad_Success(t *testing.T) {
	t.Setenv("TEST_STRING_SUCCESS", "test_value")
	t.Setenv("TEST_INT_SUCCESS", "100")
	t.Setenv("TEST_BOOL_SUCCESS", "false")

	var cfg TestConfigSuccess
	err := config.Load(&cfg)

	require.NoError(t, err, "Load should not return an error with valid environment variables")
	assert.Equal(t, "test_value", cfg.TestString, "TestString should match environment variable")
	assert.Equal(t, 100, cfg.TestInt, "TestInt should match environment variable")
	assert.Equal(t, false, cfg.TestBool, "TestBool should match environment variable")
}

func TestLoad_DefaultValues(t *testing.T) {
	// Clean environment variables to ensure defaults are used
	os.Unsetenv("TEST_STRING_DEFAULT")
	os.Unsetenv("TEST_INT_DEFAULT")
	os.Unsetenv("TEST_BOOL_DEFAULT")

	var cfg TestConfigDefault
	err := config.Load(&cfg)

	require.NoError(t, err, "Load should not return an error when using default values")
	assert.Equal(t, "default_value", cfg.TestString, "TestString should use default value")
	assert.Equal(t, 42, cfg.TestInt, "TestInt should use default value")
	assert.Equal(t, true, cfg.TestBool, "TestBool should use default value")
}

func TestLoad_MissingRequired(t *testing.T) {
	os.Unsetenv("REQUIRED_VALUE")

	var cfg RequiredConfig
	err := config.Load(&cfg)

	require.Error(t, err, "Load should return an error when a required value is missing")
	assert.True(t, errors.Is(err, config.ErrParsingConfig), "Error should be ErrParsingConfig")
}

func TestLoad_Singleton(t *testing.T) {
	t.Setenv("TEST_STRING_SINGLETON", "first_value")

	var firstConfig TestConfigSingleton
	err := config.Load(&firstConfig)
	require.NoError(t, err, "First load should not return an error")

	// Change environment variable to verify caching behavior
	t.Setenv("TEST_STRING_SINGLETON", "second_value")

	var secondConfig TestConfigSingleton
	err = config.Load(&secondConfig)
	require.NoError(t, err, "Second load should not return an error")

	// Both configs should have the same value due to singleton pattern
	assert.Equal(t, firstConfig.TestString, secondConfig.TestString,
		"Both configs should have the same value due to singleton pattern")
	assert.Equal(t, "first_value", secondConfig.TestString,
		"Second config should have the first value due to caching")
}

func TestLoad_DifferentTypes(t *testing.T) {
	t.Setenv("VALUE_TYPE1", "test_type1")
	t.Setenv("VALUE_TYPE2", "test_type2")

	var config1 TestConfigDifferent1
	err := config.Load(&config1)
	require.NoError(t, err, "Loading first config type should not error")

	var config2 TestConfigDifferent2
	err = config.Load(&config2)
	require.NoError(t, err, "Loading second config type should not error")

	assert.Equal(t, "test_type1", config1.Value, "First config should have its own value")
	assert.Equal(t, "test_type2", config2.Value, "Second config should have its own value")
}

func TestLoad_NilPointer(t *testing.T) {
	var cfg *TestConfigSuccess = nil
	err := config.Load(cfg)

	require.Error(t, err, "Load should return an error when given a nil pointer")
	assert.ErrorIs(t, err, config.ErrNilPointer, "Error should be ErrNilPointer")
}
