package sanitizer_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/pkg/sanitizer"
)

func TestClamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    int
		min      int
		max      int
		expected int
	}{
		{
			name:     "value within range",
			value:    5,
			min:      1,
			max:      10,
			expected: 5,
		},
		{
			name:     "value below minimum",
			value:    -5,
			min:      1,
			max:      10,
			expected: 1,
		},
		{
			name:     "value above maximum",
			value:    15,
			min:      1,
			max:      10,
			expected: 10,
		},
		{
			name:     "value equals minimum",
			value:    1,
			min:      1,
			max:      10,
			expected: 1,
		},
		{
			name:     "value equals maximum",
			value:    10,
			min:      1,
			max:      10,
			expected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.Clamp(tt.value, tt.min, tt.max)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClampWithFloats(t *testing.T) {
	t.Parallel()

	result := sanitizer.Clamp(3.7, 1.5, 10.2)
	assert.Equal(t, 3.7, result)

	result = sanitizer.Clamp(0.5, 1.5, 10.2)
	assert.Equal(t, 1.5, result)

	result = sanitizer.Clamp(15.8, 1.5, 10.2)
	assert.Equal(t, 10.2, result)
}

func TestClampMin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    int
		min      int
		expected int
	}{
		{
			name:     "value above minimum",
			value:    5,
			min:      1,
			expected: 5,
		},
		{
			name:     "value below minimum",
			value:    -5,
			min:      1,
			expected: 1,
		},
		{
			name:     "value equals minimum",
			value:    1,
			min:      1,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.ClampMin(tt.value, tt.min)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClampMax(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    int
		max      int
		expected int
	}{
		{
			name:     "value below maximum",
			value:    5,
			max:      10,
			expected: 5,
		},
		{
			name:     "value above maximum",
			value:    15,
			max:      10,
			expected: 10,
		},
		{
			name:     "value equals maximum",
			value:    10,
			max:      10,
			expected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.ClampMax(tt.value, tt.max)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAbs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    int
		expected int
	}{
		{
			name:     "positive number",
			value:    5,
			expected: 5,
		},
		{
			name:     "negative number",
			value:    -5,
			expected: 5,
		},
		{
			name:     "zero",
			value:    0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.Abs(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAbsWithFloats(t *testing.T) {
	t.Parallel()

	result := sanitizer.Abs(-3.14)
	assert.Equal(t, 3.14, result)

	result = sanitizer.Abs(2.71)
	assert.Equal(t, 2.71, result)
}

func TestZeroIfNegative(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    int
		expected int
	}{
		{
			name:     "positive number",
			value:    5,
			expected: 5,
		},
		{
			name:     "negative number",
			value:    -5,
			expected: 0,
		},
		{
			name:     "zero",
			value:    0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.ZeroIfNegative(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestZeroIfPositive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    int
		expected int
	}{
		{
			name:     "positive number",
			value:    5,
			expected: 0,
		},
		{
			name:     "negative number",
			value:    -5,
			expected: -5,
		},
		{
			name:     "zero",
			value:    0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.ZeroIfPositive(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNonZero(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    int
		expected int
	}{
		{
			name:     "positive number",
			value:    5,
			expected: 5,
		},
		{
			name:     "negative number",
			value:    -5,
			expected: -5,
		},
		{
			name:     "zero",
			value:    0,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.NonZero(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRoundToDecimalPlaces(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    float64
		places   int
		expected float64
	}{
		{
			name:     "round to 2 decimal places",
			value:    3.14159,
			places:   2,
			expected: 3.14,
		},
		{
			name:     "round to 0 decimal places",
			value:    3.7,
			places:   0,
			expected: 4.0,
		},
		{
			name:     "round to 1 decimal place",
			value:    2.678,
			places:   1,
			expected: 2.7,
		},
		{
			name:     "negative places defaults to 0",
			value:    3.7,
			places:   -1,
			expected: 4.0,
		},
		{
			name:     "round up case",
			value:    1.995,
			places:   2,
			expected: 2.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.RoundToDecimalPlaces(tt.value, tt.places)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRoundUp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    float64
		expected float64
	}{
		{
			name:     "positive with decimal",
			value:    3.14,
			expected: 4.0,
		},
		{
			name:     "positive integer",
			value:    3.0,
			expected: 3.0,
		},
		{
			name:     "negative with decimal",
			value:    -2.7,
			expected: -2.0,
		},
		{
			name:     "small positive",
			value:    0.1,
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.RoundUp(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRoundDown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    float64
		expected float64
	}{
		{
			name:     "positive with decimal",
			value:    3.14,
			expected: 3.0,
		},
		{
			name:     "positive integer",
			value:    3.0,
			expected: 3.0,
		},
		{
			name:     "negative with decimal",
			value:    -2.7,
			expected: -3.0,
		},
		{
			name:     "small positive",
			value:    0.9,
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.RoundDown(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    float64
		expected float64
	}{
		{
			name:     "round up",
			value:    3.6,
			expected: 4.0,
		},
		{
			name:     "round down",
			value:    3.4,
			expected: 3.0,
		},
		{
			name:     "round half up",
			value:    3.5,
			expected: 4.0,
		},
		{
			name:     "negative round",
			value:    -2.7,
			expected: -3.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.Round(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateToInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    float64
		expected float64
	}{
		{
			name:     "positive with decimal",
			value:    3.14,
			expected: 3.0,
		},
		{
			name:     "negative with decimal",
			value:    -2.7,
			expected: -2.0,
		},
		{
			name:     "positive integer",
			value:    5.0,
			expected: 5.0,
		},
		{
			name:     "small positive",
			value:    0.9,
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.TruncateToInt(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClampPrecision(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		value         float64
		min           float64
		max           float64
		decimalPlaces int
		expected      float64
	}{
		{
			name:          "value within range after rounding",
			value:         3.14159,
			min:           1.0,
			max:           10.0,
			decimalPlaces: 2,
			expected:      3.14,
		},
		{
			name:          "value clamped to minimum after rounding",
			value:         0.456,
			min:           1.0,
			max:           10.0,
			decimalPlaces: 1,
			expected:      1.0,
		},
		{
			name:          "value clamped to maximum after rounding",
			value:         15.789,
			min:           1.0,
			max:           10.0,
			decimalPlaces: 1,
			expected:      10.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.ClampPrecision(tt.value, tt.min, tt.max, tt.decimalPlaces)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSafeDivide(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		numerator   int
		denominator int
		fallback    int
		expected    int
	}{
		{
			name:        "normal division",
			numerator:   10,
			denominator: 2,
			fallback:    0,
			expected:    5,
		},
		{
			name:        "division by zero returns fallback",
			numerator:   10,
			denominator: 0,
			fallback:    -1,
			expected:    -1,
		},
		{
			name:        "zero numerator",
			numerator:   0,
			denominator: 5,
			fallback:    -1,
			expected:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.SafeDivide(tt.numerator, tt.denominator, tt.fallback)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSafeDivideWithFloats(t *testing.T) {
	t.Parallel()

	result := sanitizer.SafeDivide(10.0, 3.0, -1.0)
	assert.InDelta(t, 3.333333333333333, result, 0.000001)

	result = sanitizer.SafeDivide(10.0, 0.0, -1.0)
	assert.Equal(t, -1.0, result)
}

func TestPercentage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		part     int
		whole    int
		expected float64
	}{
		{
			name:     "normal percentage",
			part:     25,
			whole:    100,
			expected: 25.0,
		},
		{
			name:     "partial percentage",
			part:     1,
			whole:    3,
			expected: 33.333333333333336,
		},
		{
			name:     "over 100 percent clamped",
			part:     150,
			whole:    100,
			expected: 100.0,
		},
		{
			name:     "negative percentage clamped to zero",
			part:     -25,
			whole:    100,
			expected: 0.0,
		},
		{
			name:     "zero whole returns zero",
			part:     50,
			whole:    0,
			expected: 0.0,
		},
		{
			name:     "zero part",
			part:     0,
			whole:    100,
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.Percentage(tt.part, tt.whole)
			assert.InDelta(t, tt.expected, result, 0.000001)
		})
	}
}

func TestNormalizeToRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    float64
		fromMin  float64
		fromMax  float64
		toMin    float64
		toMax    float64
		expected float64
	}{
		{
			name:     "normalize middle value",
			value:    5.0,
			fromMin:  0.0,
			fromMax:  10.0,
			toMin:    0.0,
			toMax:    100.0,
			expected: 50.0,
		},
		{
			name:     "normalize minimum value",
			value:    0.0,
			fromMin:  0.0,
			fromMax:  10.0,
			toMin:    0.0,
			toMax:    100.0,
			expected: 0.0,
		},
		{
			name:     "normalize maximum value",
			value:    10.0,
			fromMin:  0.0,
			fromMax:  10.0,
			toMin:    0.0,
			toMax:    100.0,
			expected: 100.0,
		},
		{
			name:     "normalize to different range",
			value:    2.5,
			fromMin:  0.0,
			fromMax:  5.0,
			toMin:    -10.0,
			toMax:    10.0,
			expected: 0.0,
		},
		{
			name:     "normalize with same from range returns to minimum",
			value:    5.0,
			fromMin:  3.0,
			fromMax:  3.0,
			toMin:    0.0,
			toMax:    100.0,
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.NormalizeToRange(tt.value, tt.fromMin, tt.fromMax, tt.toMin, tt.toMax)
			assert.InDelta(t, tt.expected, result, 0.000001)
		})
	}
}

func TestClampToPositive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    int
		expected int
	}{
		{
			name:     "positive value unchanged",
			value:    5,
			expected: 5,
		},
		{
			name:     "zero becomes one",
			value:    0,
			expected: 1,
		},
		{
			name:     "negative becomes one",
			value:    -5,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.ClampToPositive(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClampToPositiveWithFloats(t *testing.T) {
	t.Parallel()

	result := sanitizer.ClampToPositive(3.14)
	assert.Equal(t, 3.14, result)

	result = sanitizer.ClampToPositive(0.0)
	assert.Equal(t, 1.0, result)

	result = sanitizer.ClampToPositive(-2.7)
	assert.Equal(t, 1.0, result)
}

func TestClampToNonNegative(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    int
		expected int
	}{
		{
			name:     "positive value unchanged",
			value:    5,
			expected: 5,
		},
		{
			name:     "zero unchanged",
			value:    0,
			expected: 0,
		},
		{
			name:     "negative becomes zero",
			value:    -5,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizer.ClampToNonNegative(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClampToNonNegativeWithFloats(t *testing.T) {
	t.Parallel()

	result := sanitizer.ClampToNonNegative(3.14)
	assert.Equal(t, 3.14, result)

	result = sanitizer.ClampToNonNegative(0.0)
	assert.Equal(t, 0.0, result)

	result = sanitizer.ClampToNonNegative(-2.7)
	assert.Equal(t, 0.0, result)
}

func TestNumericApplyPattern(t *testing.T) {
	t.Parallel()

	t.Run("apply pattern with numeric functions", func(t *testing.T) {
		t.Parallel()

		// Test Apply pattern with numeric transformations
		input := -15.789
		result := sanitizer.Apply(input,
			sanitizer.Abs[float64],
			func(v float64) float64 { return sanitizer.ClampMax(v, 10.0) },
			func(v float64) float64 { return sanitizer.RoundToDecimalPlaces(v, 1) },
		)
		assert.Equal(t, 10.0, result)
	})

	t.Run("compose numeric transformations", func(t *testing.T) {
		t.Parallel()

		// Create a reusable price sanitizer
		priceSanitizer := sanitizer.Compose(
			func(v float64) float64 { return sanitizer.ClampToNonNegative(v) },
			func(v float64) float64 { return sanitizer.RoundToDecimalPlaces(v, 2) },
			func(v float64) float64 { return sanitizer.ClampMax(v, 999999.99) },
		)

		// Test with different price inputs
		inputs := []float64{-10.567, 25.999, 1000000.123}
		expected := []float64{0.0, 26.0, 999999.99}

		for i, input := range inputs {
			result := priceSanitizer(input)
			assert.Equal(t, expected[i], result, "Failed for input: %f", input)
		}
	})
}

func TestRealWorldNumericUsage(t *testing.T) {
	t.Parallel()

	t.Run("user age sanitization", func(t *testing.T) {
		t.Parallel()

		// Create age sanitizer: non-negative, reasonable max age
		ageSanitizer := sanitizer.Compose(
			func(v int) int { return sanitizer.ClampToNonNegative(v) },
			func(v int) int { return sanitizer.ClampMax(v, 150) },
		)

		ages := []int{-5, 25, 200, 0, 35}
		expectedAges := []int{0, 25, 150, 0, 35}

		for i, age := range ages {
			result := ageSanitizer(age)
			assert.Equal(t, expectedAges[i], result)
		}
	})

	t.Run("percentage score sanitization", func(t *testing.T) {
		t.Parallel()

		// Create percentage sanitizer: 0-100 range, 1 decimal place
		percentageSanitizer := sanitizer.Compose(
			func(v float64) float64 { return sanitizer.Clamp(v, 0.0, 100.0) },
			func(v float64) float64 { return sanitizer.RoundToDecimalPlaces(v, 1) },
		)

		scores := []float64{-10.555, 87.678, 150.999, 45.234}
		expectedScores := []float64{0.0, 87.7, 100.0, 45.2}

		for i, score := range scores {
			result := percentageSanitizer(score)
			assert.Equal(t, expectedScores[i], result)
		}
	})

	t.Run("quantity sanitization for e-commerce", func(t *testing.T) {
		t.Parallel()

		// Create quantity sanitizer: positive integers only, max 1000
		quantitySanitizer := sanitizer.Compose(
			func(v int) int { return sanitizer.ClampToPositive(v) },
			func(v int) int { return sanitizer.ClampMax(v, 1000) },
		)

		quantities := []int{-5, 0, 25, 1500, 10}
		expectedQuantities := []int{1, 1, 25, 1000, 10}

		for i, quantity := range quantities {
			result := quantitySanitizer(quantity)
			assert.Equal(t, expectedQuantities[i], result)
		}
	})
}
