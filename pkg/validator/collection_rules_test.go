package validator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/pkg/validator"
)

func TestRequiredSlice(t *testing.T) {
	t.Run("passes for non-empty string slice", func(t *testing.T) {
		rule := validator.RequiredSlice("items", []string{"item1"})
		assert.True(t, rule.Check())
		assert.Equal(t, "items", rule.Error.Field)
		assert.Equal(t, "field is required", rule.Error.Message)
		assert.Equal(t, "validation.required", rule.Error.TranslationKey)
	})

	t.Run("fails for empty string slice", func(t *testing.T) {
		rule := validator.RequiredSlice("items", []string{})
		assert.False(t, rule.Check())
	})

	t.Run("fails for nil string slice", func(t *testing.T) {
		var items []string
		rule := validator.RequiredSlice("items", items)
		assert.False(t, rule.Check())
	})

	t.Run("passes for non-empty int slice", func(t *testing.T) {
		rule := validator.RequiredSlice("numbers", []int{1, 2, 3})
		assert.True(t, rule.Check())
	})

	t.Run("fails for empty int slice", func(t *testing.T) {
		rule := validator.RequiredSlice("numbers", []int{})
		assert.False(t, rule.Check())
	})

	t.Run("passes for non-empty custom type slice", func(t *testing.T) {
		type CustomType struct {
			ID   int
			Name string
		}
		rule := validator.RequiredSlice("objects", []CustomType{{ID: 1, Name: "test"}})
		assert.True(t, rule.Check())
	})
}

func TestMinLenSlice(t *testing.T) {
	t.Run("passes when slice equals minimum length", func(t *testing.T) {
		rule := validator.MinLenSlice("items", []string{"a", "b", "c"}, 3)
		assert.True(t, rule.Check())
		assert.Equal(t, "items", rule.Error.Field)
		assert.Equal(t, "must have at least 3 items", rule.Error.Message)
		assert.Equal(t, "validation.min_items", rule.Error.TranslationKey)
	})

	t.Run("passes when slice exceeds minimum length", func(t *testing.T) {
		rule := validator.MinLenSlice("items", []string{"a", "b", "c", "d"}, 3)
		assert.True(t, rule.Check())
	})

	t.Run("fails when slice is shorter than minimum", func(t *testing.T) {
		rule := validator.MinLenSlice("items", []string{"a", "b"}, 3)
		assert.False(t, rule.Check())
	})

	t.Run("passes with zero minimum", func(t *testing.T) {
		rule := validator.MinLenSlice("items", []string{}, 0)
		assert.True(t, rule.Check())
	})

	t.Run("works with int slice", func(t *testing.T) {
		rule := validator.MinLenSlice("numbers", []int{1, 2, 3, 4, 5}, 3)
		assert.True(t, rule.Check())
	})

	t.Run("handles large minimum length", func(t *testing.T) {
		rule := validator.MinLenSlice("items", []string{"a"}, 100)
		assert.False(t, rule.Check())
		assert.Equal(t, "must have at least 100 items", rule.Error.Message)
	})
}

func TestMaxLenSlice(t *testing.T) {
	t.Run("passes when slice equals maximum length", func(t *testing.T) {
		rule := validator.MaxLenSlice("items", []string{"a", "b", "c"}, 3)
		assert.True(t, rule.Check())
		assert.Equal(t, "items", rule.Error.Field)
		assert.Equal(t, "must have at most 3 items", rule.Error.Message)
		assert.Equal(t, "validation.max_items", rule.Error.TranslationKey)
	})

	t.Run("passes when slice is shorter than maximum", func(t *testing.T) {
		rule := validator.MaxLenSlice("items", []string{"a", "b"}, 3)
		assert.True(t, rule.Check())
	})

	t.Run("fails when slice exceeds maximum length", func(t *testing.T) {
		rule := validator.MaxLenSlice("items", []string{"a", "b", "c", "d"}, 3)
		assert.False(t, rule.Check())
	})

	t.Run("passes with zero maximum for empty slice", func(t *testing.T) {
		rule := validator.MaxLenSlice("items", []string{}, 0)
		assert.True(t, rule.Check())
	})

	t.Run("fails for any content when max is zero", func(t *testing.T) {
		rule := validator.MaxLenSlice("items", []string{"a"}, 0)
		assert.False(t, rule.Check())
	})

	t.Run("works with int slice", func(t *testing.T) {
		rule := validator.MaxLenSlice("numbers", []int{1, 2}, 5)
		assert.True(t, rule.Check())
	})
}

func TestLenSlice(t *testing.T) {
	t.Run("passes when slice equals exact length", func(t *testing.T) {
		rule := validator.LenSlice("items", []string{"a", "b", "c"}, 3)
		assert.True(t, rule.Check())
		assert.Equal(t, "items", rule.Error.Field)
		assert.Equal(t, "must have exactly 3 items", rule.Error.Message)
		assert.Equal(t, "validation.exact_items", rule.Error.TranslationKey)
	})

	t.Run("fails when slice is shorter", func(t *testing.T) {
		rule := validator.LenSlice("items", []string{"a", "b"}, 3)
		assert.False(t, rule.Check())
	})

	t.Run("fails when slice is longer", func(t *testing.T) {
		rule := validator.LenSlice("items", []string{"a", "b", "c", "d"}, 3)
		assert.False(t, rule.Check())
	})

	t.Run("passes for empty slice with zero length requirement", func(t *testing.T) {
		rule := validator.LenSlice("items", []string{}, 0)
		assert.True(t, rule.Check())
	})

	t.Run("fails for non-empty slice when zero length required", func(t *testing.T) {
		rule := validator.LenSlice("items", []string{"a"}, 0)
		assert.False(t, rule.Check())
	})

	t.Run("works with int slice", func(t *testing.T) {
		rule := validator.LenSlice("numbers", []int{1, 2, 3}, 3)
		assert.True(t, rule.Check())
	})
}

func TestRequiredMap(t *testing.T) {
	t.Run("passes for non-empty map", func(t *testing.T) {
		rule := validator.RequiredMap("config", map[string]int{"key": 1})
		assert.True(t, rule.Check())
		assert.Equal(t, "config", rule.Error.Field)
		assert.Equal(t, "field is required", rule.Error.Message)
		assert.Equal(t, "validation.required", rule.Error.TranslationKey)
	})

	t.Run("fails for empty map", func(t *testing.T) {
		rule := validator.RequiredMap("config", map[string]int{})
		assert.False(t, rule.Check())
	})

	t.Run("fails for nil map", func(t *testing.T) {
		var config map[string]int
		rule := validator.RequiredMap("config", config)
		assert.False(t, rule.Check())
	})

	t.Run("passes for non-empty map with different types", func(t *testing.T) {
		rule := validator.RequiredMap("users", map[int]string{1: "John", 2: "Jane"})
		assert.True(t, rule.Check())
	})

	t.Run("passes for non-empty map with custom types", func(t *testing.T) {
		type User struct {
			Name string
			Age  int
		}
		rule := validator.RequiredMap("users", map[string]User{"john": {Name: "John", Age: 30}})
		assert.True(t, rule.Check())
	})
}

func TestMinLenMap(t *testing.T) {
	t.Run("passes when map equals minimum length", func(t *testing.T) {
		rule := validator.MinLenMap("config", map[string]int{"a": 1, "b": 2}, 2)
		assert.True(t, rule.Check())
		assert.Equal(t, "config", rule.Error.Field)
		assert.Equal(t, "must have at least 2 items", rule.Error.Message)
		assert.Equal(t, "validation.min_items", rule.Error.TranslationKey)
	})

	t.Run("passes when map exceeds minimum length", func(t *testing.T) {
		rule := validator.MinLenMap("config", map[string]int{"a": 1, "b": 2, "c": 3}, 2)
		assert.True(t, rule.Check())
	})

	t.Run("fails when map is smaller than minimum", func(t *testing.T) {
		rule := validator.MinLenMap("config", map[string]int{"a": 1}, 2)
		assert.False(t, rule.Check())
	})

	t.Run("passes with zero minimum", func(t *testing.T) {
		rule := validator.MinLenMap("config", map[string]int{}, 0)
		assert.True(t, rule.Check())
	})

	t.Run("works with different map types", func(t *testing.T) {
		rule := validator.MinLenMap("scores", map[int]float64{1: 95.5, 2: 87.3, 3: 92.1}, 3)
		assert.True(t, rule.Check())
	})

	t.Run("handles large minimum length", func(t *testing.T) {
		rule := validator.MinLenMap("config", map[string]int{"a": 1}, 100)
		assert.False(t, rule.Check())
		assert.Equal(t, "must have at least 100 items", rule.Error.Message)
	})
}

func TestMaxLenMap(t *testing.T) {
	t.Run("passes when map equals maximum length", func(t *testing.T) {
		rule := validator.MaxLenMap("config", map[string]int{"a": 1, "b": 2}, 2)
		assert.True(t, rule.Check())
		assert.Equal(t, "config", rule.Error.Field)
		assert.Equal(t, "must have at most 2 items", rule.Error.Message)
		assert.Equal(t, "validation.max_items", rule.Error.TranslationKey)
	})

	t.Run("passes when map is smaller than maximum", func(t *testing.T) {
		rule := validator.MaxLenMap("config", map[string]int{"a": 1}, 2)
		assert.True(t, rule.Check())
	})

	t.Run("fails when map exceeds maximum length", func(t *testing.T) {
		rule := validator.MaxLenMap("config", map[string]int{"a": 1, "b": 2, "c": 3}, 2)
		assert.False(t, rule.Check())
	})

	t.Run("passes with zero maximum for empty map", func(t *testing.T) {
		rule := validator.MaxLenMap("config", map[string]int{}, 0)
		assert.True(t, rule.Check())
	})

	t.Run("fails for any content when max is zero", func(t *testing.T) {
		rule := validator.MaxLenMap("config", map[string]int{"a": 1}, 0)
		assert.False(t, rule.Check())
	})

	t.Run("works with different map types", func(t *testing.T) {
		rule := validator.MaxLenMap("scores", map[int]float64{1: 95.5}, 5)
		assert.True(t, rule.Check())
	})
}

func TestLenMap(t *testing.T) {
	t.Run("passes when map equals exact length", func(t *testing.T) {
		rule := validator.LenMap("config", map[string]int{"a": 1, "b": 2}, 2)
		assert.True(t, rule.Check())
		assert.Equal(t, "config", rule.Error.Field)
		assert.Equal(t, "must have exactly 2 items", rule.Error.Message)
		assert.Equal(t, "validation.exact_items", rule.Error.TranslationKey)
	})

	t.Run("fails when map is smaller", func(t *testing.T) {
		rule := validator.LenMap("config", map[string]int{"a": 1}, 2)
		assert.False(t, rule.Check())
	})

	t.Run("fails when map is larger", func(t *testing.T) {
		rule := validator.LenMap("config", map[string]int{"a": 1, "b": 2, "c": 3}, 2)
		assert.False(t, rule.Check())
	})

	t.Run("passes for empty map with zero length requirement", func(t *testing.T) {
		rule := validator.LenMap("config", map[string]int{}, 0)
		assert.True(t, rule.Check())
	})

	t.Run("fails for non-empty map when zero length required", func(t *testing.T) {
		rule := validator.LenMap("config", map[string]int{"a": 1}, 0)
		assert.False(t, rule.Check())
	})

	t.Run("works with different map types", func(t *testing.T) {
		rule := validator.LenMap("scores", map[int]float64{1: 95.5, 2: 87.3}, 2)
		assert.True(t, rule.Check())
	})
}

func TestCollectionRulesIntegration(t *testing.T) {
	t.Run("validates complete collection input", func(t *testing.T) {
		tags := []string{"go", "backend", "api"}
		settings := map[string]string{"theme": "dark", "lang": "en"}
		scores := []float64{85.5, 92.3, 78.9}

		err := validator.Apply(
			validator.RequiredSlice("tags", tags),
			validator.MinLenSlice("tags", tags, 1),
			validator.MaxLenSlice("tags", tags, 10),
			validator.RequiredMap("settings", settings),
			validator.MinLenMap("settings", settings, 1),
			validator.MaxLenMap("settings", settings, 20),
			validator.LenSlice("scores", scores, 3),
		)

		assert.NoError(t, err)
	})

	t.Run("collects multiple collection validation errors", func(t *testing.T) {
		emptyTags := []string{}
		emptySettings := map[string]string{}
		tooManyItems := []int{1, 2, 3, 4, 5, 6}

		err := validator.Apply(
			validator.RequiredSlice("tags", emptyTags),
			validator.RequiredMap("settings", emptySettings),
			validator.MaxLenSlice("items", tooManyItems, 5),
			validator.LenSlice("items", tooManyItems, 3),
		)

		assert.Error(t, err)
		assert.True(t, validator.IsValidationError(err))

		validationErr := validator.ExtractValidationErrors(err)
		assert.True(t, validationErr.Has("tags"))
		assert.True(t, validationErr.Has("settings"))
		assert.True(t, validationErr.Has("items"))

		tagsErrors := validationErr.Get("tags")
		assert.Contains(t, tagsErrors, "field is required")

		settingsErrors := validationErr.Get("settings")
		assert.Contains(t, settingsErrors, "field is required")

		itemsErrors := validationErr.Get("items")
		assert.Len(t, itemsErrors, 2) // Should have both max and exact length errors
		assert.Contains(t, itemsErrors, "must have at most 5 items")
		assert.Contains(t, itemsErrors, "must have exactly 3 items")
	})

	t.Run("validates mixed slice and map types", func(t *testing.T) {
		stringSlice := []string{"a", "b"}
		intSlice := []int{1, 2, 3}
		stringMap := map[string]int{"x": 1, "y": 2}
		intMap := map[int]string{1: "one", 2: "two", 3: "three"}

		err := validator.Apply(
			validator.MinLenSlice("stringSlice", stringSlice, 2),
			validator.MaxLenSlice("intSlice", intSlice, 5),
			validator.LenMap("stringMap", stringMap, 2),
			validator.MinLenMap("intMap", intMap, 3),
		)

		assert.NoError(t, err)
	})
}
