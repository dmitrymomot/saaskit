package validator_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/validator"
)

func TestPastDate(t *testing.T) {
	now := time.Now()

	t.Run("valid past dates", func(t *testing.T) {
		pastDates := []time.Time{
			now.AddDate(-1, 0, 0), // 1 year ago
			now.AddDate(0, -1, 0), // 1 month ago
			now.AddDate(0, 0, -1), // 1 day ago
			now.Add(-time.Hour),   // 1 hour ago
			now.Add(-time.Minute), // 1 minute ago
		}

		for _, date := range pastDates {
			rule := validator.PastDate("date", date)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Date should be in the past: %s", date.Format(time.RFC3339))
		}
	})

	t.Run("invalid past dates", func(t *testing.T) {
		futureDates := []time.Time{
			now.AddDate(1, 0, 0), // 1 year from now
			now.AddDate(0, 1, 0), // 1 month from now
			now.AddDate(0, 0, 1), // 1 day from now
			now.Add(time.Hour),   // 1 hour from now
			now.Add(time.Minute), // 1 minute from now
		}

		for _, date := range futureDates {
			rule := validator.PastDate("date", date)
			err := validator.Apply(rule)
			assert.Error(t, err, "Date should be rejected as future: %s", date.Format(time.RFC3339))

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.date_past", validationErr[0].TranslationKey)
		}
	})
}

func TestFutureDate(t *testing.T) {
	now := time.Now()

	t.Run("valid future dates", func(t *testing.T) {
		futureDates := []time.Time{
			now.AddDate(1, 0, 0), // 1 year from now
			now.AddDate(0, 1, 0), // 1 month from now
			now.AddDate(0, 0, 1), // 1 day from now
			now.Add(time.Hour),   // 1 hour from now
			now.Add(time.Minute), // 1 minute from now
		}

		for _, date := range futureDates {
			rule := validator.FutureDate("date", date)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Date should be in the future: %s", date.Format(time.RFC3339))
		}
	})

	t.Run("invalid future dates", func(t *testing.T) {
		pastDates := []time.Time{
			now.AddDate(-1, 0, 0), // 1 year ago
			now.AddDate(0, -1, 0), // 1 month ago
			now.AddDate(0, 0, -1), // 1 day ago
			now.Add(-time.Hour),   // 1 hour ago
			now.Add(-time.Minute), // 1 minute ago
		}

		for _, date := range pastDates {
			rule := validator.FutureDate("date", date)
			err := validator.Apply(rule)
			assert.Error(t, err, "Date should be rejected as past: %s", date.Format(time.RFC3339))

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.date_future", validationErr[0].TranslationKey)
		}
	})
}

func TestDateAfter(t *testing.T) {
	referenceDate := time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC)

	t.Run("valid dates after reference", func(t *testing.T) {
		validDates := []time.Time{
			referenceDate.AddDate(0, 0, 1), // 1 day after
			referenceDate.AddDate(0, 1, 0), // 1 month after
			referenceDate.AddDate(1, 0, 0), // 1 year after
			referenceDate.Add(time.Hour),   // 1 hour after
		}

		for _, date := range validDates {
			rule := validator.DateAfter("date", date, referenceDate)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Date should be after reference: %s", date.Format(time.RFC3339))
		}
	})

	t.Run("invalid dates before or equal to reference", func(t *testing.T) {
		invalidDates := []time.Time{
			referenceDate,                   // same date
			referenceDate.AddDate(0, 0, -1), // 1 day before
			referenceDate.AddDate(0, -1, 0), // 1 month before
			referenceDate.Add(-time.Hour),   // 1 hour before
		}

		for _, date := range invalidDates {
			rule := validator.DateAfter("date", date, referenceDate)
			err := validator.Apply(rule)
			assert.Error(t, err, "Date should be rejected: %s", date.Format(time.RFC3339))

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.date_after", validationErr[0].TranslationKey)
		}
	})
}

func TestDateBefore(t *testing.T) {
	referenceDate := time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC)

	t.Run("valid dates before reference", func(t *testing.T) {
		validDates := []time.Time{
			referenceDate.AddDate(0, 0, -1), // 1 day before
			referenceDate.AddDate(0, -1, 0), // 1 month before
			referenceDate.AddDate(-1, 0, 0), // 1 year before
			referenceDate.Add(-time.Hour),   // 1 hour before
		}

		for _, date := range validDates {
			rule := validator.DateBefore("date", date, referenceDate)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Date should be before reference: %s", date.Format(time.RFC3339))
		}
	})

	t.Run("invalid dates after or equal to reference", func(t *testing.T) {
		invalidDates := []time.Time{
			referenceDate,                  // same date
			referenceDate.AddDate(0, 0, 1), // 1 day after
			referenceDate.AddDate(0, 1, 0), // 1 month after
			referenceDate.Add(time.Hour),   // 1 hour after
		}

		for _, date := range invalidDates {
			rule := validator.DateBefore("date", date, referenceDate)
			err := validator.Apply(rule)
			assert.Error(t, err, "Date should be rejected: %s", date.Format(time.RFC3339))

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.date_before", validationErr[0].TranslationKey)
		}
	})
}

func TestDateBetween(t *testing.T) {
	startDate := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2023, 6, 30, 0, 0, 0, 0, time.UTC)

	t.Run("valid dates between range", func(t *testing.T) {
		validDates := []time.Time{
			startDate, // start date (inclusive)
			endDate,   // end date (inclusive)
			time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC), // middle
			startDate.Add(time.Hour),                     // just after start
			endDate.Add(-time.Hour),                      // just before end
		}

		for _, date := range validDates {
			rule := validator.DateBetween("date", date, startDate, endDate)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Date should be between range: %s", date.Format(time.RFC3339))
		}
	})

	t.Run("invalid dates outside range", func(t *testing.T) {
		invalidDates := []time.Time{
			startDate.AddDate(0, 0, -1), // before start
			endDate.AddDate(0, 0, 1),    // after end
			startDate.Add(-time.Hour),   // just before start
			endDate.Add(time.Hour),      // just after end
		}

		for _, date := range invalidDates {
			rule := validator.DateBetween("date", date, startDate, endDate)
			err := validator.Apply(rule)
			assert.Error(t, err, "Date should be rejected: %s", date.Format(time.RFC3339))

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.date_between", validationErr[0].TranslationKey)
		}
	})
}

func TestMinAge(t *testing.T) {
	now := time.Now()

	t.Run("valid minimum ages", func(t *testing.T) {
		testCases := []struct {
			name     string
			age      int
			minAge   int
			birthday time.Time
		}{
			{"exactly min age", 18, 18, now.AddDate(-18, 0, 0)},
			{"older than min age", 25, 18, now.AddDate(-25, 0, 0)},
			{"much older", 65, 18, now.AddDate(-65, 0, 0)},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				rule := validator.MinAge("birthdate", tc.birthday, tc.minAge)
				err := validator.Apply(rule)
				assert.NoError(t, err, "Age should meet minimum requirement")
			})
		}
	})

	t.Run("invalid minimum ages", func(t *testing.T) {
		testCases := []struct {
			name     string
			minAge   int
			birthday time.Time
		}{
			{"too young", 18, now.AddDate(-17, 0, 0)},
			{"much too young", 21, now.AddDate(-10, 0, 0)},
			{"barely too young", 18, now.AddDate(-18, 0, 1)}, // 1 day short
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				rule := validator.MinAge("birthdate", tc.birthday, tc.minAge)
				err := validator.Apply(rule)
				assert.Error(t, err, "Age should be rejected as too young")

				validationErr := validator.ExtractValidationErrors(err)
				require.NotNil(t, validationErr)
				assert.Equal(t, "validation.min_age", validationErr[0].TranslationKey)
			})
		}
	})
}

func TestMaxAge(t *testing.T) {
	now := time.Now()

	t.Run("valid maximum ages", func(t *testing.T) {
		testCases := []struct {
			name     string
			maxAge   int
			birthday time.Time
		}{
			{"exactly max age", 65, now.AddDate(-65, 0, 0)},
			{"younger than max age", 65, now.AddDate(-30, 0, 0)},
			{"much younger", 65, now.AddDate(-18, 0, 0)},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				rule := validator.MaxAge("birthdate", tc.birthday, tc.maxAge)
				err := validator.Apply(rule)
				assert.NoError(t, err, "Age should meet maximum requirement")
			})
		}
	})

	t.Run("invalid maximum ages", func(t *testing.T) {
		testCases := []struct {
			name     string
			maxAge   int
			birthday time.Time
		}{
			{"too old", 65, now.AddDate(-70, 0, 0)},
			{"much too old", 65, now.AddDate(-100, 0, 0)},
			{"barely too old", 65, now.AddDate(-66, 0, 0)},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				rule := validator.MaxAge("birthdate", tc.birthday, tc.maxAge)
				err := validator.Apply(rule)
				assert.Error(t, err, "Age should be rejected as too old")

				validationErr := validator.ExtractValidationErrors(err)
				require.NotNil(t, validationErr)
				assert.Equal(t, "validation.max_age", validationErr[0].TranslationKey)
			})
		}
	})
}

func TestAgeBetween(t *testing.T) {
	now := time.Now()

	t.Run("valid age ranges", func(t *testing.T) {
		testCases := []struct {
			name     string
			minAge   int
			maxAge   int
			birthday time.Time
		}{
			{"min age", 18, 65, now.AddDate(-18, 0, 0)},
			{"max age", 18, 65, now.AddDate(-65, 0, 0)},
			{"middle age", 18, 65, now.AddDate(-35, 0, 0)},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				rule := validator.AgeBetween("birthdate", tc.birthday, tc.minAge, tc.maxAge)
				err := validator.Apply(rule)
				assert.NoError(t, err, "Age should be within range")
			})
		}
	})

	t.Run("invalid age ranges", func(t *testing.T) {
		testCases := []struct {
			name     string
			minAge   int
			maxAge   int
			birthday time.Time
		}{
			{"too young", 18, 65, now.AddDate(-17, 0, 0)},
			{"too old", 18, 65, now.AddDate(-70, 0, 0)},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				rule := validator.AgeBetween("birthdate", tc.birthday, tc.minAge, tc.maxAge)
				err := validator.Apply(rule)
				assert.Error(t, err, "Age should be outside valid range")

				validationErr := validator.ExtractValidationErrors(err)
				require.NotNil(t, validationErr)
				assert.Equal(t, "validation.age_between", validationErr[0].TranslationKey)
			})
		}
	})
}

func TestBusinessHours(t *testing.T) {
	t.Run("valid business hours", func(t *testing.T) {
		validTimes := []time.Time{
			time.Date(2023, 6, 15, 9, 0, 0, 0, time.UTC),   // 9 AM
			time.Date(2023, 6, 15, 12, 30, 0, 0, time.UTC), // 12:30 PM
			time.Date(2023, 6, 15, 16, 59, 0, 0, time.UTC), // 4:59 PM
		}

		for _, timeVal := range validTimes {
			rule := validator.BusinessHours("time", timeVal, 9, 17) // 9 AM to 5 PM
			err := validator.Apply(rule)
			assert.NoError(t, err, "Time should be within business hours: %s", timeVal.Format("15:04"))
		}
	})

	t.Run("invalid business hours", func(t *testing.T) {
		invalidTimes := []time.Time{
			time.Date(2023, 6, 15, 8, 59, 0, 0, time.UTC),  // 8:59 AM (before)
			time.Date(2023, 6, 15, 17, 0, 0, 0, time.UTC),  // 5:00 PM (at end)
			time.Date(2023, 6, 15, 18, 0, 0, 0, time.UTC),  // 6:00 PM (after)
			time.Date(2023, 6, 15, 23, 30, 0, 0, time.UTC), // 11:30 PM
			time.Date(2023, 6, 15, 2, 0, 0, 0, time.UTC),   // 2:00 AM
		}

		for _, timeVal := range invalidTimes {
			rule := validator.BusinessHours("time", timeVal, 9, 17)
			err := validator.Apply(rule)
			assert.Error(t, err, "Time should be outside business hours: %s", timeVal.Format("15:04"))

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.business_hours", validationErr[0].TranslationKey)
		}
	})
}

func TestWorkingDay(t *testing.T) {
	t.Run("valid working days", func(t *testing.T) {
		// June 2023: 5th=Monday, 6th=Tuesday, 7th=Wednesday, 8th=Thursday, 9th=Friday
		workingDays := []time.Time{
			time.Date(2023, 6, 5, 0, 0, 0, 0, time.UTC), // Monday
			time.Date(2023, 6, 6, 0, 0, 0, 0, time.UTC), // Tuesday
			time.Date(2023, 6, 7, 0, 0, 0, 0, time.UTC), // Wednesday
			time.Date(2023, 6, 8, 0, 0, 0, 0, time.UTC), // Thursday
			time.Date(2023, 6, 9, 0, 0, 0, 0, time.UTC), // Friday
		}

		for _, date := range workingDays {
			rule := validator.WorkingDay("date", date)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Date should be a working day: %s", date.Format("Monday, 2006-01-02"))
		}
	})

	t.Run("invalid working days", func(t *testing.T) {
		// June 2023: 3rd=Saturday, 4th=Sunday, 10th=Saturday, 11th=Sunday
		weekendDays := []time.Time{
			time.Date(2023, 6, 3, 0, 0, 0, 0, time.UTC),  // Saturday
			time.Date(2023, 6, 4, 0, 0, 0, 0, time.UTC),  // Sunday
			time.Date(2023, 6, 10, 0, 0, 0, 0, time.UTC), // Saturday
			time.Date(2023, 6, 11, 0, 0, 0, 0, time.UTC), // Sunday
		}

		for _, date := range weekendDays {
			rule := validator.WorkingDay("date", date)
			err := validator.Apply(rule)
			assert.Error(t, err, "Date should not be a working day: %s", date.Format("Monday, 2006-01-02"))

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.working_day", validationErr[0].TranslationKey)
		}
	})
}

func TestWeekend(t *testing.T) {
	t.Run("valid weekend days", func(t *testing.T) {
		weekendDays := []time.Time{
			time.Date(2023, 6, 3, 0, 0, 0, 0, time.UTC),  // Saturday
			time.Date(2023, 6, 4, 0, 0, 0, 0, time.UTC),  // Sunday
			time.Date(2023, 6, 10, 0, 0, 0, 0, time.UTC), // Saturday
			time.Date(2023, 6, 11, 0, 0, 0, 0, time.UTC), // Sunday
		}

		for _, date := range weekendDays {
			rule := validator.Weekend("date", date)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Date should be a weekend day: %s", date.Format("Monday, 2006-01-02"))
		}
	})

	t.Run("invalid weekend days", func(t *testing.T) {
		workingDays := []time.Time{
			time.Date(2023, 6, 5, 0, 0, 0, 0, time.UTC), // Monday
			time.Date(2023, 6, 6, 0, 0, 0, 0, time.UTC), // Tuesday
			time.Date(2023, 6, 7, 0, 0, 0, 0, time.UTC), // Wednesday
			time.Date(2023, 6, 8, 0, 0, 0, 0, time.UTC), // Thursday
			time.Date(2023, 6, 9, 0, 0, 0, 0, time.UTC), // Friday
		}

		for _, date := range workingDays {
			rule := validator.Weekend("date", date)
			err := validator.Apply(rule)
			assert.Error(t, err, "Date should not be a weekend day: %s", date.Format("Monday, 2006-01-02"))

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.weekend", validationErr[0].TranslationKey)
		}
	})
}

func TestTimeAfter(t *testing.T) {
	referenceTime := time.Date(2023, 6, 15, 12, 0, 0, 0, time.UTC) // 12:00 PM

	t.Run("valid times after reference", func(t *testing.T) {
		validTimes := []time.Time{
			time.Date(2023, 6, 15, 12, 0, 1, 0, time.UTC),   // 1 second after
			time.Date(2023, 6, 15, 12, 30, 0, 0, time.UTC),  // 30 minutes after
			time.Date(2023, 6, 15, 15, 0, 0, 0, time.UTC),   // 3 hours after
			time.Date(2023, 6, 15, 23, 59, 59, 0, time.UTC), // late evening
		}

		for _, timeVal := range validTimes {
			rule := validator.TimeAfter("time", timeVal, referenceTime)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Time should be after reference: %s", timeVal.Format("15:04:05"))
		}
	})

	t.Run("invalid times before or equal to reference", func(t *testing.T) {
		invalidTimes := []time.Time{
			referenceTime, // same time
			time.Date(2023, 6, 15, 11, 59, 59, 0, time.UTC), // 1 second before
			time.Date(2023, 6, 15, 11, 30, 0, 0, time.UTC),  // 30 minutes before
			time.Date(2023, 6, 15, 9, 0, 0, 0, time.UTC),    // 3 hours before
		}

		for _, timeVal := range invalidTimes {
			rule := validator.TimeAfter("time", timeVal, referenceTime)
			err := validator.Apply(rule)
			assert.Error(t, err, "Time should be rejected: %s", timeVal.Format("15:04:05"))

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.time_after", validationErr[0].TranslationKey)
		}
	})
}

func TestTimeBefore(t *testing.T) {
	referenceTime := time.Date(2023, 6, 15, 12, 0, 0, 0, time.UTC) // 12:00 PM

	t.Run("valid times before reference", func(t *testing.T) {
		validTimes := []time.Time{
			time.Date(2023, 6, 15, 11, 59, 59, 0, time.UTC), // 1 second before
			time.Date(2023, 6, 15, 11, 30, 0, 0, time.UTC),  // 30 minutes before
			time.Date(2023, 6, 15, 9, 0, 0, 0, time.UTC),    // 3 hours before
			time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC),    // midnight
		}

		for _, timeVal := range validTimes {
			rule := validator.TimeBefore("time", timeVal, referenceTime)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Time should be before reference: %s", timeVal.Format("15:04:05"))
		}
	})

	t.Run("invalid times after or equal to reference", func(t *testing.T) {
		invalidTimes := []time.Time{
			referenceTime, // same time
			time.Date(2023, 6, 15, 12, 0, 1, 0, time.UTC),  // 1 second after
			time.Date(2023, 6, 15, 12, 30, 0, 0, time.UTC), // 30 minutes after
			time.Date(2023, 6, 15, 15, 0, 0, 0, time.UTC),  // 3 hours after
		}

		for _, timeVal := range invalidTimes {
			rule := validator.TimeBefore("time", timeVal, referenceTime)
			err := validator.Apply(rule)
			assert.Error(t, err, "Time should be rejected: %s", timeVal.Format("15:04:05"))

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.time_before", validationErr[0].TranslationKey)
		}
	})
}

func TestTimeBetween(t *testing.T) {
	startTime := time.Date(2023, 6, 15, 9, 0, 0, 0, time.UTC) // 9:00 AM
	endTime := time.Date(2023, 6, 15, 17, 0, 0, 0, time.UTC)  // 5:00 PM

	t.Run("valid times between range", func(t *testing.T) {
		validTimes := []time.Time{
			startTime, // start time (inclusive)
			endTime,   // end time (inclusive)
			time.Date(2023, 6, 15, 12, 0, 0, 0, time.UTC),  // noon
			time.Date(2023, 6, 15, 9, 30, 0, 0, time.UTC),  // 9:30 AM
			time.Date(2023, 6, 15, 16, 30, 0, 0, time.UTC), // 4:30 PM
		}

		for _, timeVal := range validTimes {
			rule := validator.TimeBetween("time", timeVal, startTime, endTime)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Time should be between range: %s", timeVal.Format("15:04:05"))
		}
	})

	t.Run("invalid times outside range", func(t *testing.T) {
		invalidTimes := []time.Time{
			time.Date(2023, 6, 15, 8, 59, 59, 0, time.UTC), // just before start
			time.Date(2023, 6, 15, 17, 0, 1, 0, time.UTC),  // just after end
			time.Date(2023, 6, 15, 6, 0, 0, 0, time.UTC),   // early morning
			time.Date(2023, 6, 15, 20, 0, 0, 0, time.UTC),  // evening
		}

		for _, timeVal := range invalidTimes {
			rule := validator.TimeBetween("time", timeVal, startTime, endTime)
			err := validator.Apply(rule)
			assert.Error(t, err, "Time should be outside range: %s", timeVal.Format("15:04:05"))

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.time_between", validationErr[0].TranslationKey)
		}
	})
}

func TestValidBirthdate(t *testing.T) {
	now := time.Now()

	t.Run("valid birthdates", func(t *testing.T) {
		validBirthdates := []time.Time{
			now.AddDate(-25, 0, 0), // 25 years ago
			now.AddDate(-50, 0, 0), // 50 years ago
			now.AddDate(-18, 0, 0), // 18 years ago
			now.AddDate(-80, 0, 0), // 80 years ago
			now.AddDate(-1, 0, -1), // just over 1 year ago
		}

		for _, birthdate := range validBirthdates {
			rule := validator.ValidBirthdate("birthdate", birthdate)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Birthdate should be valid: %s", birthdate.Format("2006-01-02"))
		}
	})

	t.Run("invalid birthdates", func(t *testing.T) {
		invalidBirthdates := []time.Time{
			now.AddDate(0, 0, 1),    // future date
			now.AddDate(1, 0, 0),    // 1 year in future
			now.AddDate(-200, 0, 0), // 200 years ago (too old)
			now.Add(time.Hour),      // 1 hour in future
		}

		for _, birthdate := range invalidBirthdates {
			rule := validator.ValidBirthdate("birthdate", birthdate)
			err := validator.Apply(rule)
			assert.Error(t, err, "Birthdate should be invalid: %s", birthdate.Format("2006-01-02"))

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.valid_birthdate", validationErr[0].TranslationKey)
		}
	})
}
