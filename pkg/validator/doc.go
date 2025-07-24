// Package validator provides a composable set of generic, type-safe validation
// helpers and rule-building utilities for common data types such as strings,
// numbers, collections, financial values, dates, UUIDs, and more.
//
// The package promotes declarative validation by letting you build small Rule
// values that encapsulate a boolean Check function together with rich,
// translation-friendly error metadata. Rules are evaluated with the Apply
// helper which aggregates any failures into a ValidationErrors slice that
// satisfies the error interface, making it convenient to bubble up multiple
// field-specific problems in a single error return.
//
// # Architecture
//
// Each source file groups a family of rules for a specific domain
// (`string_rules.go`, `numeric_rules.go`, `date_rules.go`, etc.). Every
// exported validation function simply constructs and returns a Rule instance;
// there is no hidden global state, therefore the package is completely
// stateless, allocation-light, and goroutine-safe.
//
// Core building blocks:
//   - Rule              – lightweight struct containing Check func and error meta
//   - ValidationError   – describes a single failure and supports i18n keys
//   - ValidationErrors  – slice type that implements the error interface
//   - Numeric interface – generic constraint used by numeric helpers
//
// # Usage
//
//	error := validator.Apply(
//	    validator.RequiredSlice("items", items),
//	    validator.ValidEmail("email", email),
//	    validator.MinNum("age", age, 18),
//	)
//	if err != nil {
//	    if verrs := validator.ExtractValidationErrors(err); verrs != nil {
//	        // iterate over field-level messages or translate them
//	    }
//	}
//
// # Error Handling
//
// ValidationErrors implements `Is`, `As`, and `Error`, so you can use
// `errors.Is/As` to detect validation problems while preserving rich details.
// Individual field errors can be inspected with the helper methods Has, Get,
// GetErrors and Fields.
//
// # Performance Considerations
//
// All helpers are simple, allocation-free comparisons or pattern checks.
// Long-running or expensive validations (e.g. network calls) should be
// implemented outside this package and adapted into a Rule where appropriate.
//
// # Examples
//
// See the companion *_test.go files for runnable examples covering each rule
// set.
package validator
