# Validator Package

A high-performance, type-safe validation package for Go using generics with comprehensive translation support. Provides composable validation rules with zero reflection overhead and structured error handling.

## Overview

The `validator` package provides a clean, composable way to validate input data with compile-time type safety and comprehensive internationalization support. It eliminates reflection entirely through generic type constraints, offering better performance and clearer APIs for validation logic. The package includes built-in translation support for multilingual applications.

## Features

- **Zero Reflection**: All validation uses generics for compile-time type safety
- **High Performance**: No runtime type checking or conversions
- **Translation Support**: Built-in internationalization with translation keys and values
- **Composable Rules**: Build complex validation by combining simple rules
- **Structured Errors**: Rich error information with field-specific details
- **Type Safety**: Generic functions prevent type mismatches at compile time
- **Multiple Errors**: Collect all validation errors, not just the first one

## Core Types

### ValidationError

A single validation error with comprehensive translation support:

```go
type ValidationError struct {
    Field             string
    Message           string
    TranslationKey    string
    TranslationValues map[string]any
}
```

### ValidationErrors

A collection of validation errors:

```go
type ValidationErrors []ValidationError
```

#### Methods

- `Error() string` - Implements error interface with formatted message
- `Add(err ValidationError)` - Add a validation error to the collection
- `Has(field string) bool` - Check if a field has errors
- `Get(field string) []string` - Get all error messages for a field
- `GetErrors(field string) []ValidationError` - Get all ValidationError objects for a field
- `Fields() []string` - Get all fields with errors
- `IsEmpty() bool` - Check if there are any errors
- `GetTranslatableErrors() []ValidationError` - Get all errors with translation data

### Rule

A validation rule that can be applied to check a specific condition:

```go
type Rule struct {
    Check func() bool
    Error ValidationError
}
```

### Type Constraints

```go
type Numeric interface {
    ~int | ~int8 | ~int16 | ~int32 | ~int64 |
    ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
    ~float32 | ~float64
}
```

## Validation Functions

### String Validators

Type-safe string validation with automatic whitespace trimming for required checks:

```go
validator.RequiredString(field, value string) Rule
validator.MinLenString(field, value string, min int) Rule
validator.MaxLenString(field, value string, max int) Rule
validator.LenString(field, value string, exact int) Rule
```

### Numeric Validators

Generic numeric validation supporting all integer and float types:

```go
validator.RequiredNum[T Numeric](field string, value T) Rule
validator.MinNum[T Numeric](field string, value T, min T) Rule
validator.MaxNum[T Numeric](field string, value T, max T) Rule
```

### Slice Validators

Generic slice validation for any slice type:

```go
validator.RequiredSlice[T any](field string, value []T) Rule
validator.MinLenSlice[T any](field string, value []T, min int) Rule
validator.MaxLenSlice[T any](field string, value []T, max int) Rule
validator.LenSlice[T any](field string, value []T, exact int) Rule
```

### Map Validators

Generic map validation for any map type:

```go
validator.RequiredMap[K comparable, V any](field string, value map[K]V) Rule
validator.MinLenMap[K comparable, V any](field string, value map[K]V, min int) Rule
validator.MaxLenMap[K comparable, V any](field string, value map[K]V, max int) Rule
validator.LenMap[K comparable, V any](field string, value map[K]V, exact int) Rule
```

### Comparable Validators

For any comparable type (useful for custom types):

### Comparable Validation Functions

```go
func RequiredComparable[T comparable](field string, value T) Rule
```

### Format Validation Functions

Email, URL, phone, IP, and MAC address validation:

```go
func ValidEmail(field, value string) Rule
func ValidURL(field, value string) Rule
func ValidURLWithScheme(field, value string, schemes []string) Rule
func ValidPhone(field, value string) Rule
func ValidIPv4(field, value string) Rule
func ValidIPv6(field, value string) Rule
func ValidIP(field, value string) Rule
func ValidMAC(field, value string) Rule
func ValidAlphanumeric(field, value string) Rule
func ValidAlpha(field, value string) Rule
func ValidNumericString(field, value string) Rule
```

### UUID Validation Functions

UUID format and version validation:

```go
func ValidUUID(field, value string) Rule
func NonNilUUID(field string, value uuid.UUID) Rule
func NonNilUUIDString(field, value string) Rule
func ValidUUIDVersion(field string, value uuid.UUID, version int) Rule
func ValidUUIDVersionString(field, value string, version int) Rule
func ValidUUIDv1(field string, value uuid.UUID) Rule
func ValidUUIDv1String(field, value string) Rule
func ValidUUIDv3(field string, value uuid.UUID) Rule
func ValidUUIDv3String(field, value string) Rule
func ValidUUIDv4(field string, value uuid.UUID) Rule
func ValidUUIDv4String(field, value string) Rule
func ValidUUIDv5(field string, value uuid.UUID) Rule
func ValidUUIDv5String(field, value string) Rule
func RequiredUUID(field string, value uuid.UUID) Rule
func RequiredUUIDString(field, value string) Rule
```

### Password Validation Functions

Password strength and security validation:

```go
func StrongPassword(field, value string, config PasswordStrengthConfig) Rule
func PasswordUppercase(field, value string) Rule
func PasswordLowercase(field, value string) Rule
func PasswordDigit(field, value string) Rule
func PasswordSpecialChar(field, value string) Rule
func NotCommonPassword(field, value string) Rule
func PasswordEntropy(field, value string, minEntropy float64) Rule
func NoRepeatingChars(field, value string, maxRepeats int) Rule
func NoSequentialChars(field, value string, maxSequential int) Rule
```

### Date/Time Validation Functions

Date and time validation with business logic:

```go
func PastDate(field string, value time.Time) Rule
func FutureDate(field string, value time.Time) Rule
func DateAfter(field string, value time.Time, after time.Time) Rule
func DateBefore(field string, value time.Time, before time.Time) Rule
func DateBetween(field string, value time.Time, start time.Time, end time.Time) Rule
func MinAge(field string, birthdate time.Time, minAge int) Rule
func MaxAge(field string, birthdate time.Time, maxAge int) Rule
func AgeBetween(field string, birthdate time.Time, minAge int, maxAge int) Rule
func BusinessHours(field string, value time.Time, startHour int, endHour int) Rule
func WorkingDay(field string, value time.Time) Rule
func Weekend(field string, value time.Time) Rule
func TimeAfter(field string, value time.Time, after time.Time) Rule
func TimeBefore(field string, value time.Time, before time.Time) Rule
func TimeBetween(field string, value time.Time, start time.Time, end time.Time) Rule
func ValidBirthdate(field string, value time.Time) Rule
```

### Financial Validation Functions

Currency, monetary amounts, and financial validation:

```go
func PositiveAmount[T Numeric](field string, value T) Rule
func NonNegativeAmount[T Numeric](field string, value T) Rule
func AmountRange[T Numeric](field string, value T, min T, max T) Rule
func DecimalPrecision(field string, value float64, maxDecimals int) Rule
func CurrencyPrecision(field string, value float64, decimals int) Rule
func ValidCurrencyCode(field, value string) Rule
func ValidTaxRate(field string, value float64) Rule
func ValidInterestRate(field string, value float64, maxRate float64) Rule
func ValidPercentage(field string, value float64) Rule
func ValidDiscount(field string, value float64) Rule
func MinimumPurchase[T Numeric](field string, value T, minimum T) Rule
func MaximumTransaction[T Numeric](field string, value T, maximum T) Rule
func ValidCreditCardChecksum(field, value string) Rule
func ValidAccountNumber(field, value string) Rule
func ValidRoutingNumber(field, value string) Rule
```

### Identifier Validation Functions

Custom ID formats and alphanumeric codes:

```go
func ValidSlug(field, value string) Rule
func ValidUsername(field, value string, minLen int, maxLen int) Rule
func ValidHandle(field, value string, minLen int, maxLen int) Rule
func ValidSKU(field, value string) Rule
func ValidProductCode(field, value string, pattern string) Rule
func ValidHexString(field, value string, exactLength int) Rule
func ValidBase64(field, value string) Rule
func ValidCustomID(field, value string, pattern string, description string) Rule
func ValidDomainName(field, value string) Rule
func ValidSubdomain(field, value string) Rule
func ValidAPIKey(field, value string, minLength int, maxLength int) Rule
func ValidTicketNumber(field, value string, prefix string) Rule
func ValidVersion(field, value string) Rule
func ValidOTP(field, value string, length int) Rule
```

### Pattern Validation Functions

Regular expressions and text pattern validation:

```go
func MatchesRegex(field, value string, pattern string, description string) Rule
func DoesNotMatchRegex(field, value string, pattern string, description string) Rule
func ContainsPattern(field, value string, pattern string, description string) Rule
func StartsWithPattern(field, value string, pattern string) Rule
func EndsWithPattern(field, value string, pattern string) Rule
func NoWhitespace(field, value string) Rule
func OnlyWhitespace(field, value string) Rule
func NoControlChars(field, value string) Rule
func PrintableChars(field, value string) Rule
func ASCIIOnly(field, value string) Rule
func NoSpecialChars(field, value string) Rule
func ContainsUppercase(field, value string) Rule
func ContainsLowercase(field, value string) Rule
func ContainsDigit(field, value string) Rule
func BalancedParentheses(field, value string) Rule
func WordCount(field, value string, min int, max int) Rule
func LineCount(field, value string, min int, max int) Rule
```

### Choice Validation Functions

In-list, not-in-list, and enum validation:

```go
func InList[T comparable](field string, value T, allowedValues []T) Rule
func NotInList[T comparable](field string, value T, forbiddenValues []T) Rule
func InListString(field, value string, allowedValues []string) Rule
func NotInListString(field, value string, forbiddenValues []string) Rule
func InListCaseInsensitive(field, value string, allowedValues []string) Rule
func NotInListCaseInsensitive(field, value string, forbiddenValues []string) Rule
func OneOf[T comparable](field string, value T, options []T) Rule
func OneOfString(field, value string, options []string) Rule
func NoneOf[T comparable](field string, value T, options []T) Rule
func NoneOfString(field, value string, options []string) Rule
func ValidEnum(field, value string, enumValues []string) Rule
func ValidEnumCaseInsensitive(field, value string, enumValues []string) Rule
func ValidStatus(field, value string, allowedStatuses []string) Rule
func ValidRole(field, value string, allowedRoles []string) Rule
func ValidPermission(field, value string, allowedPermissions []string) Rule
func ValidCategory(field, value string, allowedCategories []string) Rule
```

### Convenience Aliases

For common string and numeric use cases:

```go
func Required(field, value string) Rule           // Alias for RequiredString
func MinLen(field, value string, min int) Rule    // Alias for MinLenString
func MaxLen(field, value string, max int) Rule    // Alias for MaxLenString
func Len(field, value string, exact int) Rule     // Alias for LenString
func Min[T Numeric](field string, value T, min T) Rule  // Alias for MinNum
func Max[T Numeric](field string, value T, max T) Rule  // Alias for MaxNum
```

## Core Functions

### Apply

Executes multiple validation rules and returns any errors:

```go
func Apply(rules ...Rule) error
```

Returns `nil` if all validations pass, or `ValidationErrors` if any fail.

### Helper Functions

#### ExtractValidationErrors

Extracts `ValidationErrors` from an error:

```go
func ExtractValidationErrors(err error) ValidationErrors
```

#### IsValidationError

Checks if an error is a `ValidationErrors` type:

```go
func IsValidationError(err error) bool
```

## Translation Support

The validator package includes comprehensive translation support with predefined translation keys:

### Translation Keys

#### Basic Validation

- `validation.required` - Field is required
- `validation.min_length` - Minimum length validation
- `validation.max_length` - Maximum length validation
- `validation.exact_length` - Exact length validation
- `validation.min` - Minimum value validation
- `validation.max` - Maximum value validation
- `validation.min_items` - Minimum items validation
- `validation.max_items` - Maximum items validation
- `validation.exact_items` - Exact items validation

#### Format Validation

- `validation.email` - Valid email address
- `validation.url` - Valid URL
- `validation.url_scheme` - URL with specific scheme
- `validation.phone` - Valid phone number
- `validation.ipv4` - Valid IPv4 address
- `validation.ipv6` - Valid IPv6 address
- `validation.ip` - Valid IP address
- `validation.mac` - Valid MAC address
- `validation.alphanumeric` - Alphanumeric characters only
- `validation.alpha` - Letters only
- `validation.numeric_string` - Digits only

#### UUID Validation

- `validation.uuid` - Valid UUID format
- `validation.uuid_not_nil` - UUID not nil
- `validation.uuid_version` - Specific UUID version

#### Password Validation

- `validation.password_strength` - Password strength requirements
- `validation.password_uppercase` - Contains uppercase letter
- `validation.password_lowercase` - Contains lowercase letter
- `validation.password_digit` - Contains digit
- `validation.password_special` - Contains special character
- `validation.password_common` - Not a common password
- `validation.password_entropy` - Minimum entropy requirement
- `validation.password_repeating` - No excessive repeating characters
- `validation.password_sequential` - No excessive sequential characters

#### Date/Time Validation

- `validation.date_past` - Date in the past
- `validation.date_future` - Date in the future
- `validation.date_after` - Date after specified date
- `validation.date_before` - Date before specified date
- `validation.date_between` - Date within range
- `validation.min_age` - Minimum age requirement
- `validation.max_age` - Maximum age requirement
- `validation.age_between` - Age within range
- `validation.business_hours` - Within business hours
- `validation.working_day` - Working day (Monday-Friday)
- `validation.weekend` - Weekend day (Saturday-Sunday)
- `validation.time_after` - Time after specified time
- `validation.time_before` - Time before specified time
- `validation.time_between` - Time within range
- `validation.valid_birthdate` - Valid birthdate

#### Financial Validation

- `validation.positive_amount` - Positive amount
- `validation.non_negative_amount` - Non-negative amount
- `validation.amount_range` - Amount within range
- `validation.decimal_precision` - Decimal precision
- `validation.currency_code` - Valid currency code
- `validation.tax_rate` - Valid tax rate
- `validation.interest_rate` - Valid interest rate
- `validation.percentage` - Valid percentage
- `validation.minimum_purchase` - Minimum purchase amount
- `validation.maximum_transaction` - Maximum transaction amount
- `validation.credit_card` - Valid credit card number
- `validation.account_number` - Valid account number
- `validation.routing_number` - Valid routing number

#### Identifier Validation

- `validation.slug` - Valid URL slug
- `validation.username` - Valid username
- `validation.handle` - Valid handle
- `validation.sku` - Valid SKU
- `validation.product_code` - Valid product code
- `validation.hex_string` - Valid hexadecimal string
- `validation.base64` - Valid base64 string
- `validation.custom_id` - Valid custom ID
- `validation.domain_name` - Valid domain name
- `validation.subdomain` - Valid subdomain
- `validation.api_key` - Valid API key
- `validation.ticket_number` - Valid ticket number
- `validation.version` - Valid semantic version
- `validation.otp_code` - Valid OTP code with specified length

#### Pattern Validation

- `validation.regex_pattern` - Matches regex pattern
- `validation.regex_not_pattern` - Does not match regex pattern
- `validation.contains_pattern` - Contains pattern
- `validation.starts_with_pattern` - Starts with pattern
- `validation.ends_with_pattern` - Ends with pattern
- `validation.no_whitespace` - No whitespace characters
- `validation.only_whitespace` - Only whitespace characters
- `validation.no_control_chars` - No control characters
- `validation.printable_chars` - Only printable characters
- `validation.ascii_only` - ASCII characters only
- `validation.no_special_chars` - No special characters
- `validation.contains_uppercase` - Contains uppercase letter
- `validation.contains_lowercase` - Contains lowercase letter
- `validation.contains_digit` - Contains digit
- `validation.balanced_parentheses` - Balanced parentheses
- `validation.word_count` - Word count within range
- `validation.line_count` - Line count within range

#### Choice Validation

- `validation.in_list` - Value in allowed list
- `validation.not_in_list` - Value not in forbidden list
- `validation.in_list_case_insensitive` - Value in list (case insensitive)
- `validation.not_in_list_case_insensitive` - Value not in list (case insensitive)
- `validation.valid_status` - Valid status value
- `validation.valid_role` - Valid role value
- `validation.valid_permission` - Valid permission value
- `validation.valid_category` - Valid category value

### Translation Values

Each validation error includes a `TranslationValues` map with context:

```go
// Example translation values for different validation types
map[string]any{
    "field": "email",           // Field name
    "min":   8,                // Minimum value/length
    "max":   100,              // Maximum value/length
    "length": 10,              // Exact length
    "count": 5,                // Exact count
}
```

## Usage Examples

### Basic Validation

```go
import "github.com/dmitrymomot/saaskit/pkg/validator"

type LoginParams struct {
    Email    string
    Password string
}

func (p LoginParams) Validate() error {
    return validator.Apply(
        validator.RequiredString("email", p.Email),
        validator.ValidEmail("email", p.Email),
        validator.RequiredString("password", p.Password),
        validator.MinLenString("password", p.Password, 8),
    )
}
```

### Advanced Validation with New Rules

```go
type UserRegistrationParams struct {
    Email       string
    Password    string
    Username    string
    Website     string
    Phone       string
    Birthdate   time.Time
    UserID      uuid.UUID
}

func (p UserRegistrationParams) Validate() error {
    return validator.Apply(
        // Email validation
        validator.RequiredString("email", p.Email),
        validator.ValidEmail("email", p.Email),

        // Password validation
        validator.RequiredString("password", p.Password),
        validator.StrongPassword("password", p.Password, validator.DefaultPasswordStrength()),
        validator.NotCommonPassword("password", p.Password),

        // Username validation
        validator.RequiredString("username", p.Username),
        validator.ValidUsername("username", p.Username, 3, 20),

        // Website validation (optional)
        validator.ValidURL("website", p.Website),
        validator.ValidURLWithScheme("website", p.Website, []string{"https"}),

        // Phone validation (optional)
        validator.ValidPhone("phone", p.Phone),

        // Birthdate validation
        validator.ValidBirthdate("birthdate", p.Birthdate),
        validator.MinAge("birthdate", p.Birthdate, 13),
        validator.MaxAge("birthdate", p.Birthdate, 120),

        // UUID validation
        validator.RequiredUUID("user_id", p.UserID),
        validator.ValidUUIDv4("user_id", p.UserID),
    )
}
```

### Financial Validation Example

```go
type PaymentParams struct {
    Amount     float64
    Currency   string
    CardNumber string
    TaxRate    float64
}

func (p PaymentParams) Validate() error {
    return validator.Apply(
        // Amount validation
        validator.PositiveAmount("amount", p.Amount),
        validator.DecimalPrecision("amount", p.Amount, 2),
        validator.AmountRange("amount", p.Amount, 0.01, 10000.00),

        // Currency validation
        validator.RequiredString("currency", p.Currency),
        validator.ValidCurrencyCode("currency", p.Currency),

        // Credit card validation
        validator.RequiredString("card_number", p.CardNumber),
        validator.ValidCreditCardChecksum("card_number", p.CardNumber),

        // Tax rate validation
        validator.ValidTaxRate("tax_rate", p.TaxRate),
    )
}
```

### Complex Multi-Type Validation

```go
type CreateUserParams struct {
    Email       string
    Password    string
    Age         int
    Username    string
    Bio         string
    Tags        []string
    Settings    map[string]string
    Score       float64
}

func (p CreateUserParams) Validate() error {
    return validator.Apply(
        // String validation
        validator.RequiredString("email", p.Email),
        validator.RequiredString("password", p.Password),
        validator.RequiredString("username", p.Username),
        validator.MinLenString("password", p.Password, 8),
        validator.MaxLenString("password", p.Password, 128),
        validator.MinLenString("username", p.Username, 3),
        validator.MaxLenString("username", p.Username, 30),
        validator.MaxLenString("bio", p.Bio, 500),

        // Numeric validation
        validator.RequiredNum("age", p.Age),
        validator.MinNum("age", p.Age, 13),
        validator.MaxNum("age", p.Age, 120),
        validator.MinNum("score", p.Score, 0.0),
        validator.MaxNum("score", p.Score, 100.0),

        // Collection validation
        validator.MaxLenSlice("tags", p.Tags, 10),
        validator.MaxLenMap("settings", p.Settings, 20),
    )
}
```

### Error Handling with Translation Support

```go
func handleValidationError(err error) {
    if !validator.IsValidationError(err) {
        log.Printf("Unexpected error: %v", err)
        return
    }

    validationErr := validator.ExtractValidationErrors(err)
    if validationErr == nil {
        return
    }

    // Process validation errors with translation data
    for _, validationError := range validationErr.GetTranslatableErrors() {
        fmt.Printf("Field: %s\n", validationError.Field)
        fmt.Printf("Message: %s\n", validationError.Message)
        fmt.Printf("Translation Key: %s\n", validationError.TranslationKey)
        fmt.Printf("Translation Values: %+v\n", validationError.TranslationValues)
    }
}
```

### Pattern and Choice Validation Examples

```go
type ProductParams struct {
    SKU        string
    Status     string
    Category   string
    Version    string
    APIKey     string
}

func (p ProductParams) Validate() error {
    return validator.Apply(
        // SKU validation
        validator.RequiredString("sku", p.SKU),
        validator.ValidSKU("sku", p.SKU),

        // Status validation
        validator.ValidStatus("status", p.Status, []string{"active", "inactive", "draft"}),

        // Category validation (case insensitive)
        validator.ValidCategory("category", p.Category, []string{"electronics", "clothing", "books"}),

        // Version validation
        validator.ValidVersion("version", p.Version),

        // API key validation
        validator.ValidAPIKey("api_key", p.APIKey, 32, 64),
    )
}
```

### OTP Validation Examples

```go
type VerifyOTPParams struct {
    Email  string
    Code   string
    Action string
}

func (p VerifyOTPParams) Validate() error {
    return validator.Apply(
        // Email validation
        validator.RequiredString("email", p.Email),
        validator.ValidEmail("email", p.Email),

        // OTP code validation (6-digit)
        validator.RequiredString("code", p.Code),
        validator.ValidOTP("code", p.Code, 6),

        // Action validation
        validator.RequiredString("action", p.Action),
        validator.InListString("action", p.Action, []string{"email_verification", "password_reset"}),
    )
}

// Different OTP lengths for different use cases
type SecurityParams struct {
    PIN           string // 4-digit PIN
    OTPCode       string // 6-digit OTP
    SecurityCode  string // 8-digit security code
}

func (p SecurityParams) Validate() error {
    return validator.Apply(
        // 4-digit PIN validation
        validator.ValidOTP("pin", p.PIN, 4),

        // 6-digit OTP validation
        validator.ValidOTP("otp_code", p.OTPCode, 6),

        // 8-digit security code validation
        validator.ValidOTP("security_code", p.SecurityCode, 8),
    )
}
```

### Custom Validation Rules

Create domain-specific validation rules:

```go
// Custom email domain validation
func AllowedEmailDomain(field, email string, allowedDomains []string) validator.Rule {
    return validator.Rule{
        Check: func() bool {
            parts := strings.Split(email, "@")
            if len(parts) != 2 {
                return false
            }
            domain := parts[1]
            for _, allowed := range allowedDomains {
                if domain == allowed {
                    return true
                }
            }
            return false
        },
        Error: validator.ValidationError{
            Field:             field,
            Message:           fmt.Sprintf("email domain must be one of: %s", strings.Join(allowedDomains, ", ")),
            TranslationKey:    "validation.allowed_email_domain",
            TranslationValues: map[string]any{
                "field":   field,
                "domains": allowedDomains,
            },
        },
    }
}

// Custom business hours validation
func BusinessHoursValidation(field string, dateTime time.Time) validator.Rule {
    return validator.Rule{
        Check: func() bool {
            // Must be working day and within business hours
            return validator.Apply(
                validator.WorkingDay(field, dateTime),
                validator.BusinessHours(field, dateTime, 9, 17),
            ) == nil
        },
        Error: validator.ValidationError{
            Field:             field,
            Message:           "must be during business hours (9 AM - 5 PM, Monday-Friday)",
            TranslationKey:    "validation.business_hours_only",
            TranslationValues: map[string]any{
                "field": field,
            },
        },
    }
}

// Usage
func (p SignupParams) Validate() error {
    return validator.Apply(
        validator.RequiredString("email", p.Email),
        validator.ValidEmail("email", p.Email),
        AllowedEmailDomain("email", p.Email, []string{"company.com", "partner.com"}),
    )
}
```

## Best Practices

1. **Validation Structure**:
    - Always validate input at the transport layer
    - Use struct methods to encapsulate validation logic
    - Combine multiple rules for comprehensive validation

2. **Error Handling**:
    - Use `IsValidationError` to distinguish validation errors
    - Extract translation data for internationalized applications
    - Provide clear, actionable error messages

3. **Performance**:
    - Leverage generic type constraints for compile-time safety
    - Use appropriate validation functions for each data type
    - Cache validation results for frequently validated data

4. **Translation**:
    - Use consistent translation keys across your application
    - Provide meaningful translation values for context
    - Implement proper fallback messages for missing translations

## API Reference

### Core Functions

```go
func Apply(rules ...Rule) error
```

Executes multiple validation rules and returns any validation errors.

```go
func ExtractValidationErrors(err error) ValidationErrors
```

Extracts ValidationErrors from an error, returns nil if not a validation error.

```go
func IsValidationError(err error) bool
```

Checks if an error is a ValidationErrors type.

### String Validation Functions

```go
func RequiredString(field, value string) Rule
func MinLenString(field, value string, min int) Rule
func MaxLenString(field, value string, max int) Rule
func LenString(field, value string, exact int) Rule
func Required(field, value string) Rule      // Alias for RequiredString
func MinLen(field, value string, min int) Rule    // Alias for MinLenString
func MaxLen(field, value string, max int) Rule    // Alias for MaxLenString
func Len(field, value string, exact int) Rule     // Alias for LenString
```

### Numeric Validation Functions

```go
func RequiredNum[T Numeric](field string, value T) Rule
func MinNum[T Numeric](field string, value T, min T) Rule
func MaxNum[T Numeric](field string, value T, max T) Rule
func Min[T Numeric](field string, value T, min T) Rule  // Alias for MinNum
func Max[T Numeric](field string, value T, max T) Rule  // Alias for MaxNum
```

### Collection Validation Functions

```go
func RequiredSlice[T any](field string, value []T) Rule
func MinLenSlice[T any](field string, value []T, min int) Rule
func MaxLenSlice[T any](field string, value []T, max int) Rule
func LenSlice[T any](field string, value []T, exact int) Rule
func RequiredMap[K comparable, V any](field string, value map[K]V) Rule
func MinLenMap[K comparable, V any](field string, value map[K]V, min int) Rule
func MaxLenMap[K comparable, V any](field string, value map[K]V, max int) Rule
func LenMap[K comparable, V any](field string, value map[K]V, exact int) Rule
```

### Comparable Validation Functions

```go
func RequiredComparable[T comparable](field string, value T) Rule
```

### Error Types

```go
var ErrValidationFailed = errors.New("validation failed")
var ErrFieldRequired = errors.New("field is required")
var ErrInvalidLength = errors.New("invalid length")
var ErrInvalidValue = errors.New("invalid value")
var ErrOutOfRange = errors.New("value out of range")
var ErrInvalidFormat = errors.New("invalid format")
```

## Benefits of Generic Approach

### Compile-Time Type Safety

```go
// This will cause a compile error:
validator.MinNum("age", "not-a-number", 18)  // ❌ Compile error

// This is type-safe:
validator.MinNum("age", 25, 18)              // ✅ Compiles fine
```

### Zero Runtime Overhead

- No reflection-based type checking
- No runtime type conversions
- Direct value comparisons
- Optimal performance for request validation

### Translation Support

- Structured translation keys for consistent internationalization
- Rich context data for dynamic message generation
- Fallback to default messages when translations unavailable

## Supported Types

The validator package works with:

- **Strings**: All string operations with length validation and format checking
- **All Numeric Types**: int, int8-64, uint, uint8-64, float32, float64
- **Slices**: Any slice type `[]T`
- **Maps**: Any map type `map[K]V` where K is comparable
- **Comparable Types**: Any type that supports `==` comparison
- **UUIDs**: github.com/google/uuid.UUID type
- **Time**: time.Time for date and time validation
- **Financial Types**: Monetary amounts with precision validation
- **Pattern Types**: Regular expressions and text patterns
- **Choice Types**: Enum-like validation with lists of allowed values

## Password Strength Configuration

```go
type PasswordStrengthConfig struct {
    MinLength        int
    MaxLength        int
    RequireUppercase bool
    RequireLowercase bool
    RequireDigits    bool
    RequireSpecial   bool
    MinCharClasses   int // Minimum number of different character classes required
}

// Get default configuration
config := validator.DefaultPasswordStrength()

// Custom configuration
customConfig := validator.PasswordStrengthConfig{
    MinLength:        12,
    MaxLength:        100,
    RequireUppercase: true,
    RequireLowercase: true,
    RequireDigits:    true,
    RequireSpecial:   true,
    MinCharClasses:   4,
}
```
