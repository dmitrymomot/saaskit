package validator

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

var (
	// ISO 4217 currency codes - subset for common international commerce
	validCurrencyCodes = map[string]bool{
		"USD": true, "EUR": true, "GBP": true, "JPY": true, "AUD": true, "CAD": true,
		"CHF": true, "CNY": true, "SEK": true, "NZD": true, "MXN": true, "SGD": true,
		"HKD": true, "NOK": true, "KRW": true, "TRY": true, "RUB": true, "INR": true,
		"BRL": true, "ZAR": true, "PLN": true, "CZK": true, "HUF": true, "ILS": true,
		"CLP": true, "PHP": true, "AED": true, "COP": true, "SAR": true, "MYR": true,
		"RON": true, "THB": true, "BGN": true, "HRK": true, "ISK": true, "DKK": true,
	}

	// Currency code regex (3 uppercase letters)
	currencyCodeRegex = regexp.MustCompile(`^[A-Z]{3}$`)
)

func PositiveAmount[T Numeric](field string, value T) Rule {
	return Rule{
		Check: func() bool {
			return value > 0
		},
		Error: ValidationError{
			Field:          field,
			Message:        "amount must be positive",
			TranslationKey: "validation.positive_amount",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

func NonNegativeAmount[T Numeric](field string, value T) Rule {
	return Rule{
		Check: func() bool {
			return value >= 0
		},
		Error: ValidationError{
			Field:          field,
			Message:        "amount cannot be negative",
			TranslationKey: "validation.non_negative_amount",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

func AmountRange[T Numeric](field string, value T, min T, max T) Rule {
	return Rule{
		Check: func() bool {
			return value >= min && value <= max
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("amount must be between %v and %v", min, max),
			TranslationKey: "validation.amount_range",
			TranslationValues: map[string]any{
				"field": field,
				"min":   min,
				"max":   max,
			},
		},
	}
}

// DecimalPrecision prevents floating-point precision issues in financial calculations.
func DecimalPrecision(field string, value float64, maxDecimals int) Rule {
	return Rule{
		Check: func() bool {
			multiplier := math.Pow(10, float64(maxDecimals))
			return math.Floor(value*multiplier) == value*multiplier
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("value cannot have more than %d decimal places", maxDecimals),
			TranslationKey: "validation.decimal_precision",
			TranslationValues: map[string]any{
				"field":        field,
				"max_decimals": maxDecimals,
			},
		},
	}
}

// CurrencyPrecision enforces currency-specific decimal rules (USD=2, JPY=0, etc.).
func CurrencyPrecision(field string, value float64, decimals int) Rule {
	return DecimalPrecision(field, value, decimals)
}

// ValidCurrencyCode validates that a string is a valid ISO 4217 currency code.
func ValidCurrencyCode(field, value string) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}
			upper := strings.ToUpper(value)
			return currencyCodeRegex.MatchString(upper) && validCurrencyCodes[upper]
		},
		Error: ValidationError{
			Field:          field,
			Message:        "must be a valid ISO 4217 currency code",
			TranslationKey: "validation.currency_code",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// ValidTaxRate validates that a tax rate is reasonable (0-100%).
func ValidTaxRate(field string, value float64) Rule {
	return Rule{
		Check: func() bool {
			return value >= 0 && value <= 100
		},
		Error: ValidationError{
			Field:          field,
			Message:        "tax rate must be between 0% and 100%",
			TranslationKey: "validation.tax_rate",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// ValidInterestRate validates that an interest rate is reasonable.
func ValidInterestRate(field string, value float64, maxRate float64) Rule {
	return Rule{
		Check: func() bool {
			return value >= 0 && value <= maxRate
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("interest rate must be between 0%% and %.2f%%", maxRate),
			TranslationKey: "validation.interest_rate",
			TranslationValues: map[string]any{
				"field":    field,
				"max_rate": maxRate,
			},
		},
	}
}

// ValidPercentage validates that a value is a valid percentage (0-100).
func ValidPercentage(field string, value float64) Rule {
	return Rule{
		Check: func() bool {
			return value >= 0 && value <= 100
		},
		Error: ValidationError{
			Field:          field,
			Message:        "percentage must be between 0% and 100%",
			TranslationKey: "validation.percentage",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// ValidDiscount validates that a discount percentage is reasonable (0-100%).
func ValidDiscount(field string, value float64) Rule {
	return ValidPercentage(field, value)
}

// MinimumPurchase validates that a purchase amount meets a minimum threshold.
func MinimumPurchase[T Numeric](field string, value T, minimum T) Rule {
	return Rule{
		Check: func() bool {
			return value >= minimum
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("minimum purchase amount is %v", minimum),
			TranslationKey: "validation.minimum_purchase",
			TranslationValues: map[string]any{
				"field":   field,
				"minimum": minimum,
			},
		},
	}
}

// MaximumTransaction validates that a transaction amount doesn't exceed a maximum limit.
func MaximumTransaction[T Numeric](field string, value T, maximum T) Rule {
	return Rule{
		Check: func() bool {
			return value <= maximum
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("maximum transaction amount is %v", maximum),
			TranslationKey: "validation.maximum_transaction",
			TranslationValues: map[string]any{
				"field":   field,
				"maximum": maximum,
			},
		},
	}
}

// ValidCreditCardChecksum validates a credit card number using the Luhn algorithm.
func ValidCreditCardChecksum(field, value string) Rule {
	return Rule{
		Check: func() bool {
			// Remove spaces and dashes
			cleaned := strings.ReplaceAll(strings.ReplaceAll(value, " ", ""), "-", "")

			// Must be all digits
			if !regexp.MustCompile(`^\d+$`).MatchString(cleaned) {
				return false
			}

			// Must be between 13-19 digits
			if len(cleaned) < 13 || len(cleaned) > 19 {
				return false
			}

			// Luhn algorithm
			sum := 0
			isEven := false

			// Process digits from right to left
			for i := len(cleaned) - 1; i >= 0; i-- {
				digit := int(cleaned[i] - '0')

				if isEven {
					digit *= 2
					if digit > 9 {
						digit = digit/10 + digit%10
					}
				}

				sum += digit
				isEven = !isEven
			}

			return sum%10 == 0
		},
		Error: ValidationError{
			Field:          field,
			Message:        "invalid credit card number",
			TranslationKey: "validation.credit_card",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// ValidAccountNumber validates that a bank account number is in a reasonable format.
// This is a basic validation - real implementations should use country-specific rules.
func ValidAccountNumber(field, value string) Rule {
	return Rule{
		Check: func() bool {
			// Remove spaces and dashes
			cleaned := strings.ReplaceAll(strings.ReplaceAll(value, " ", ""), "-", "")

			// Must be alphanumeric and reasonable length
			if !regexp.MustCompile(`^[A-Za-z0-9]+$`).MatchString(cleaned) {
				return false
			}

			return len(cleaned) >= 4 && len(cleaned) <= 34 // IBAN max length
		},
		Error: ValidationError{
			Field:          field,
			Message:        "invalid account number format",
			TranslationKey: "validation.account_number",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// ValidRoutingNumber validates that a routing number is in the correct format (US).
func ValidRoutingNumber(field, value string) Rule {
	return Rule{
		Check: func() bool {
			// Remove spaces and dashes
			cleaned := strings.ReplaceAll(strings.ReplaceAll(value, " ", ""), "-", "")

			// Must be exactly 9 digits
			if !regexp.MustCompile(`^\d{9}$`).MatchString(cleaned) {
				return false
			}

			// ABA routing number checksum validation
			weights := []int{3, 7, 1, 3, 7, 1, 3, 7, 1}
			sum := 0

			for i, digit := range cleaned {
				sum += int(digit-'0') * weights[i]
			}

			return sum%10 == 0
		},
		Error: ValidationError{
			Field:          field,
			Message:        "invalid routing number",
			TranslationKey: "validation.routing_number",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}
