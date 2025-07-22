package sanitizer

import (
	"math"
)

// Numeric represents numeric types that support basic arithmetic operations.
type Numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// Signed represents signed numeric types.
type Signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~float32 | ~float64
}

// Float represents floating-point numeric types.
type Float interface {
	~float32 | ~float64
}

// Clamp constrains a numeric value to be within the specified range [min, max].
// If the value is less than min, it returns min. If greater than max, it returns max.
func Clamp[T Numeric](value T, min T, max T) T {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// ClampMin ensures a numeric value is not less than the specified minimum.
func ClampMin[T Numeric](value T, min T) T {
	if value < min {
		return min
	}
	return value
}

// ClampMax ensures a numeric value is not greater than the specified maximum.
func ClampMax[T Numeric](value T, max T) T {
	if value > max {
		return max
	}
	return value
}

// Abs returns the absolute value of a signed numeric value.
func Abs[T Signed](value T) T {
	if value < 0 {
		return -value
	}
	return value
}

// ZeroIfNegative returns zero if the value is negative, otherwise returns the value.
func ZeroIfNegative[T Signed](value T) T {
	if value < 0 {
		return 0
	}
	return value
}

// ZeroIfPositive returns zero if the value is positive, otherwise returns the value.
func ZeroIfPositive[T Signed](value T) T {
	if value > 0 {
		return 0
	}
	return value
}

// NonZero returns 1 if the value is zero, otherwise returns the value unchanged.
func NonZero[T Numeric](value T) T {
	if value == 0 {
		return 1
	}
	return value
}

// RoundToDecimalPlaces rounds a floating-point number to the specified number of decimal places.
func RoundToDecimalPlaces[T Float](value T, places int) T {
	if places < 0 {
		places = 0
	}

	multiplier := math.Pow(10, float64(places))
	return T(math.Round(float64(value)*multiplier) / multiplier)
}

// RoundUp rounds a floating-point number up to the nearest integer.
func RoundUp[T Float](value T) T {
	return T(math.Ceil(float64(value)))
}

// RoundDown rounds a floating-point number down to the nearest integer.
func RoundDown[T Float](value T) T {
	return T(math.Floor(float64(value)))
}

// Round rounds a floating-point number to the nearest integer.
func Round[T Float](value T) T {
	return T(math.Round(float64(value)))
}

// TruncateToInt truncates a floating-point number to an integer, removing the decimal part.
func TruncateToInt[T Float](value T) T {
	return T(math.Trunc(float64(value)))
}

// ClampPrecision limits floating-point precision by rounding to specified decimal places
// and ensuring the result is within the given range.
func ClampPrecision[T Float](value T, min T, max T, decimalPlaces int) T {
	rounded := RoundToDecimalPlaces(value, decimalPlaces)
	return Clamp(rounded, min, max)
}

// SafeDivide performs division with protection against division by zero.
// Returns the result of numerator/denominator, or fallback if denominator is zero.
func SafeDivide[T Numeric](numerator T, denominator T, fallback T) T {
	if denominator == 0 {
		return fallback
	}
	return numerator / denominator
}

// Percentage calculates what percentage 'part' is of 'whole', clamped between 0 and 100.
// Returns 0 if whole is zero.
func Percentage[T Numeric](part T, whole T) float64 {
	if whole == 0 {
		return 0.0
	}

	percentage := (float64(part) / float64(whole)) * 100.0
	return math.Max(0.0, math.Min(100.0, percentage))
}

// NormalizeToRange maps a value from one range to another range proportionally.
// Maps value from [fromMin, fromMax] to [toMin, toMax].
func NormalizeToRange[T Float](value T, fromMin T, fromMax T, toMin T, toMax T) T {
	if fromMax == fromMin {
		return toMin
	}

	ratio := (value - fromMin) / (fromMax - fromMin)
	return toMin + ratio*(toMax-toMin)
}

// ClampToPositive ensures the value is positive (> 0). Returns 1 if value <= 0.
func ClampToPositive[T Numeric](value T) T {
	if value <= 0 {
		return 1
	}
	return value
}

// ClampToNonNegative ensures the value is non-negative (>= 0). Returns 0 if value < 0.
func ClampToNonNegative[T Signed](value T) T {
	if value < 0 {
		return 0
	}
	return value
}
