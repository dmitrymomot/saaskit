package validator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/validator"
)

func TestPositiveAmount(t *testing.T) {
	t.Parallel()
	t.Run("valid positive amounts", func(t *testing.T) {
		validAmounts := []float64{
			0.01,
			1.0,
			100.50,
			999999.99,
			0.001,
		}

		for _, amount := range validAmounts {
			rule := validator.PositiveAmount("amount", amount)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Amount should be positive: %f", amount)
		}
	})

	t.Run("invalid positive amounts", func(t *testing.T) {
		invalidAmounts := []float64{
			0.0,
			-0.01,
			-100.50,
			-999999.99,
		}

		for _, amount := range invalidAmounts {
			rule := validator.PositiveAmount("amount", amount)
			err := validator.Apply(rule)
			assert.Error(t, err, "Amount should be rejected: %f", amount)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.positive_amount", validationErr[0].TranslationKey)
		}
	})

	t.Run("positive integers", func(t *testing.T) {
		validAmounts := []int{
			1,
			100,
			999999,
		}

		for _, amount := range validAmounts {
			rule := validator.PositiveAmount("amount", amount)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Amount should be positive: %d", amount)
		}
	})
}

func TestNonNegativeAmount(t *testing.T) {
	t.Parallel()
	t.Run("valid non-negative amounts", func(t *testing.T) {
		validAmounts := []float64{
			0.0,
			0.01,
			1.0,
			100.50,
			999999.99,
		}

		for _, amount := range validAmounts {
			rule := validator.NonNegativeAmount("amount", amount)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Amount should be non-negative: %f", amount)
		}
	})

	t.Run("invalid non-negative amounts", func(t *testing.T) {
		invalidAmounts := []float64{
			-0.01,
			-100.50,
			-999999.99,
		}

		for _, amount := range invalidAmounts {
			rule := validator.NonNegativeAmount("amount", amount)
			err := validator.Apply(rule)
			assert.Error(t, err, "Amount should be rejected: %f", amount)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.non_negative_amount", validationErr[0].TranslationKey)
		}
	})
}

func TestAmountRange(t *testing.T) {
	t.Parallel()
	t.Run("valid amount ranges", func(t *testing.T) {
		testCases := []struct {
			amount float64
			min    float64
			max    float64
		}{
			{10.0, 5.0, 15.0},    // middle
			{5.0, 5.0, 15.0},     // min boundary
			{15.0, 5.0, 15.0},    // max boundary
			{100.0, 0.0, 1000.0}, // wide range
		}

		for _, tc := range testCases {
			rule := validator.AmountRange("amount", tc.amount, tc.min, tc.max)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Amount %f should be in range [%f, %f]", tc.amount, tc.min, tc.max)
		}
	})

	t.Run("invalid amount ranges", func(t *testing.T) {
		testCases := []struct {
			amount float64
			min    float64
			max    float64
		}{
			{4.99, 5.0, 15.0},  // below min
			{15.01, 5.0, 15.0}, // above max
			{-1.0, 0.0, 100.0}, // negative
		}

		for _, tc := range testCases {
			rule := validator.AmountRange("amount", tc.amount, tc.min, tc.max)
			err := validator.Apply(rule)
			assert.Error(t, err, "Amount %f should be outside range [%f, %f]", tc.amount, tc.min, tc.max)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.amount_range", validationErr[0].TranslationKey)
		}
	})
}

func TestDecimalPrecision(t *testing.T) {
	t.Parallel()
	t.Run("valid decimal precision", func(t *testing.T) {
		testCases := []struct {
			value       float64
			maxDecimals int
		}{
			{10.0, 2},    // no decimals
			{10.5, 2},    // 1 decimal
			{10.55, 2},   // 2 decimals
			{100.0, 0},   // no decimals allowed
			{123.456, 3}, // 3 decimals
		}

		for _, tc := range testCases {
			rule := validator.DecimalPrecision("amount", tc.value, tc.maxDecimals)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Value %f should have valid precision for %d decimals", tc.value, tc.maxDecimals)
		}
	})

	t.Run("invalid decimal precision", func(t *testing.T) {
		testCases := []struct {
			value       float64
			maxDecimals int
		}{
			{10.555, 2},   // too many decimals
			{10.1, 0},     // decimals not allowed
			{123.4567, 3}, // too many decimals
		}

		for _, tc := range testCases {
			rule := validator.DecimalPrecision("amount", tc.value, tc.maxDecimals)
			err := validator.Apply(rule)
			assert.Error(t, err, "Value %f should be rejected for %d decimals", tc.value, tc.maxDecimals)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.decimal_precision", validationErr[0].TranslationKey)
		}
	})
}

func TestValidCurrencyCode(t *testing.T) {
	t.Parallel()
	t.Run("valid currency codes", func(t *testing.T) {
		validCodes := []string{
			"USD",
			"EUR",
			"GBP",
			"JPY",
			"AUD",
			"CAD",
			"CHF",
			"CNY",
			"usd", // should work with lowercase
			"eur",
		}

		for _, code := range validCodes {
			rule := validator.ValidCurrencyCode("currency", code)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Currency code should be valid: %s", code)
		}
	})

	t.Run("invalid currency codes", func(t *testing.T) {
		invalidCodes := []string{
			"",
			"   ",
			"US",   // too short
			"USDD", // too long
			"123",  // numbers
			"US1",  // mixed
			"XXX",  // not in valid list
			"ABC",  // not in valid list
		}

		for _, code := range invalidCodes {
			rule := validator.ValidCurrencyCode("currency", code)
			err := validator.Apply(rule)
			assert.Error(t, err, "Currency code should be invalid: %s", code)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.currency_code", validationErr[0].TranslationKey)
		}
	})
}

func TestValidTaxRate(t *testing.T) {
	t.Parallel()
	t.Run("valid tax rates", func(t *testing.T) {
		validRates := []float64{
			0.0,   // no tax
			5.5,   // typical sales tax
			10.0,  // VAT
			25.0,  // high VAT
			100.0, // maximum
		}

		for _, rate := range validRates {
			rule := validator.ValidTaxRate("tax_rate", rate)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Tax rate should be valid: %f", rate)
		}
	})

	t.Run("invalid tax rates", func(t *testing.T) {
		invalidRates := []float64{
			-0.1,  // negative
			100.1, // over 100%
			200.0, // way over 100%
		}

		for _, rate := range invalidRates {
			rule := validator.ValidTaxRate("tax_rate", rate)
			err := validator.Apply(rule)
			assert.Error(t, err, "Tax rate should be invalid: %f", rate)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.tax_rate", validationErr[0].TranslationKey)
		}
	})
}

func TestValidInterestRate(t *testing.T) {
	t.Parallel()
	t.Run("valid interest rates", func(t *testing.T) {
		maxRate := 25.0
		validRates := []float64{
			0.0,  // zero interest
			2.5,  // low rate
			15.0, // moderate rate
			25.0, // maximum allowed
		}

		for _, rate := range validRates {
			rule := validator.ValidInterestRate("interest_rate", rate, maxRate)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Interest rate should be valid: %f", rate)
		}
	})

	t.Run("invalid interest rates", func(t *testing.T) {
		maxRate := 25.0
		invalidRates := []float64{
			-0.1, // negative
			25.1, // over maximum
			50.0, // way over maximum
		}

		for _, rate := range invalidRates {
			rule := validator.ValidInterestRate("interest_rate", rate, maxRate)
			err := validator.Apply(rule)
			assert.Error(t, err, "Interest rate should be invalid: %f", rate)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.interest_rate", validationErr[0].TranslationKey)
		}
	})
}

func TestValidPercentage(t *testing.T) {
	t.Parallel()
	t.Run("valid percentages", func(t *testing.T) {
		validPercentages := []float64{
			0.0,
			25.5,
			50.0,
			75.25,
			100.0,
		}

		for _, percentage := range validPercentages {
			rule := validator.ValidPercentage("percentage", percentage)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Percentage should be valid: %f", percentage)
		}
	})

	t.Run("invalid percentages", func(t *testing.T) {
		invalidPercentages := []float64{
			-0.1,
			100.1,
			200.0,
		}

		for _, percentage := range invalidPercentages {
			rule := validator.ValidPercentage("percentage", percentage)
			err := validator.Apply(rule)
			assert.Error(t, err, "Percentage should be invalid: %f", percentage)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.percentage", validationErr[0].TranslationKey)
		}
	})
}

func TestMinimumPurchase(t *testing.T) {
	t.Parallel()
	t.Run("valid minimum purchases", func(t *testing.T) {
		minimum := 10.0
		validAmounts := []float64{
			10.0,  // exact minimum
			15.0,  // above minimum
			100.0, // well above minimum
		}

		for _, amount := range validAmounts {
			rule := validator.MinimumPurchase("amount", amount, minimum)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Amount should meet minimum: %f", amount)
		}
	})

	t.Run("invalid minimum purchases", func(t *testing.T) {
		minimum := 10.0
		invalidAmounts := []float64{
			9.99, // below minimum
			5.0,  // well below minimum
			0.0,  // zero
		}

		for _, amount := range invalidAmounts {
			rule := validator.MinimumPurchase("amount", amount, minimum)
			err := validator.Apply(rule)
			assert.Error(t, err, "Amount should be below minimum: %f", amount)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.minimum_purchase", validationErr[0].TranslationKey)
		}
	})
}

func TestMaximumTransaction(t *testing.T) {
	t.Parallel()
	t.Run("valid maximum transactions", func(t *testing.T) {
		maximum := 1000.0
		validAmounts := []float64{
			100.0,  // below maximum
			500.0,  // well below maximum
			1000.0, // exact maximum
		}

		for _, amount := range validAmounts {
			rule := validator.MaximumTransaction("amount", amount, maximum)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Amount should be within maximum: %f", amount)
		}
	})

	t.Run("invalid maximum transactions", func(t *testing.T) {
		maximum := 1000.0
		invalidAmounts := []float64{
			1000.01, // slightly over maximum
			1500.0,  // well over maximum
			10000.0, // way over maximum
		}

		for _, amount := range invalidAmounts {
			rule := validator.MaximumTransaction("amount", amount, maximum)
			err := validator.Apply(rule)
			assert.Error(t, err, "Amount should exceed maximum: %f", amount)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.maximum_transaction", validationErr[0].TranslationKey)
		}
	})
}

func TestValidCreditCardChecksum(t *testing.T) {
	t.Parallel()
	t.Run("valid credit card numbers", func(t *testing.T) {
		validCards := []string{
			"4532015112830366",    // Visa
			"5555555555554444",    // Mastercard
			"378282246310005",     // American Express
			"6011111111111117",    // Discover
			"4532 0151 1283 0366", // Visa with spaces
			"4532-0151-1283-0366", // Visa with dashes
		}

		for _, card := range validCards {
			rule := validator.ValidCreditCardChecksum("card", card)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Credit card should be valid: %s", card)
		}
	})

	t.Run("invalid credit card numbers", func(t *testing.T) {
		invalidCards := []string{
			"",
			"   ",
			"4532015112830367",     // Invalid checksum
			"1234567890123456",     // Invalid checksum
			"123",                  // Too short
			"12345678901234567890", // Too long
			"abcd1234567890123",    // Contains letters
			"4532015112830366123",  // Too long
		}

		for _, card := range invalidCards {
			rule := validator.ValidCreditCardChecksum("card", card)
			err := validator.Apply(rule)
			assert.Error(t, err, "Credit card should be invalid: %s", card)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.credit_card", validationErr[0].TranslationKey)
		}
	})
}

func TestValidAccountNumber(t *testing.T) {
	t.Parallel()
	t.Run("valid account numbers", func(t *testing.T) {
		validAccounts := []string{
			"1234567890",
			"ABCD1234567890",
			"123-456-789",
			"123 456 789",
			"GB82WEST12345698765432", // IBAN format
		}

		for _, account := range validAccounts {
			rule := validator.ValidAccountNumber("account", account)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Account number should be valid: %s", account)
		}
	})

	t.Run("invalid account numbers", func(t *testing.T) {
		invalidAccounts := []string{
			"",
			"   ",
			"123",                                 // Too short
			"12345678901234567890123456789012345", // Too long (over 34 chars)
			"123@456",                             // Invalid characters
			"123.456",                             // Invalid characters
		}

		for _, account := range invalidAccounts {
			rule := validator.ValidAccountNumber("account", account)
			err := validator.Apply(rule)
			assert.Error(t, err, "Account number should be invalid: %s", account)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.account_number", validationErr[0].TranslationKey)
		}
	})
}

func TestValidRoutingNumber(t *testing.T) {
	t.Parallel()
	t.Run("valid routing numbers", func(t *testing.T) {
		validRouting := []string{
			"021000021",   // Valid ABA routing number
			"011401533",   // Valid ABA routing number
			"021-000-021", // With dashes
			"021 000 021", // With spaces
		}

		for _, routing := range validRouting {
			rule := validator.ValidRoutingNumber("routing", routing)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Routing number should be valid: %s", routing)
		}
	})

	t.Run("invalid routing numbers", func(t *testing.T) {
		invalidRouting := []string{
			"",
			"   ",
			"12345678",    // Too short
			"1234567890",  // Too long
			"021000022",   // Invalid checksum
			"abc123456",   // Contains letters
			"021-000-022", // Invalid checksum with dashes
		}

		for _, routing := range invalidRouting {
			rule := validator.ValidRoutingNumber("routing", routing)
			err := validator.Apply(rule)
			assert.Error(t, err, "Routing number should be invalid: %s", routing)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.routing_number", validationErr[0].TranslationKey)
		}
	})
}

func TestFinancialValidationCombination(t *testing.T) {
	t.Parallel()
	t.Run("comprehensive financial validation", func(t *testing.T) {
		amount := 99.99
		currency := "USD"
		taxRate := 8.25

		err := validator.Apply(
			validator.PositiveAmount("amount", amount),
			validator.DecimalPrecision("amount", amount, 2),
			validator.AmountRange("amount", amount, 1.0, 1000.0),
			validator.ValidCurrencyCode("currency", currency),
			validator.ValidTaxRate("tax_rate", taxRate),
		)

		assert.NoError(t, err, "Valid financial data should pass all validations")
	})

	t.Run("invalid financial data fails multiple validations", func(t *testing.T) {
		amount := -50.555
		currency := "INVALID"
		taxRate := 150.0

		err := validator.Apply(
			validator.PositiveAmount("amount", amount),
			validator.DecimalPrecision("amount", amount, 2),
			validator.ValidCurrencyCode("currency", currency),
			validator.ValidTaxRate("tax_rate", taxRate),
		)

		assert.Error(t, err, "Invalid financial data should fail validations")

		validationErr := validator.ExtractValidationErrors(err)
		require.NotNil(t, validationErr)
		assert.True(t, len(validationErr) > 1, "Should have multiple validation errors")
	})
}
