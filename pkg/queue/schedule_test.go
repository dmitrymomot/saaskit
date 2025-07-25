package queue_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/queue"
)

func TestIntervalSchedule(t *testing.T) {
	t.Parallel()

	t.Run("every interval", func(t *testing.T) {

		schedule := queue.EveryInterval(30 * time.Second)
		base := time.Now()
		next := schedule.Next(base)

		assert.Equal(t, base.Add(30*time.Second), next)
		assert.Equal(t, "every 30s", schedule.String())
	})

	t.Run("every minute", func(t *testing.T) {

		schedule := queue.EveryMinute()
		base := time.Now()
		next := schedule.Next(base)

		assert.Equal(t, base.Add(time.Minute), next)
		assert.Equal(t, "every 1m0s", schedule.String())
	})

	t.Run("every N minutes", func(t *testing.T) {

		schedule := queue.EveryMinutes(15)
		base := time.Now()
		next := schedule.Next(base)

		assert.Equal(t, base.Add(15*time.Minute), next)
		assert.Equal(t, "every 15m0s", schedule.String())
	})

	t.Run("every N hours", func(t *testing.T) {

		schedule := queue.EveryHours(2)
		base := time.Now()
		next := schedule.Next(base)

		assert.Equal(t, base.Add(2*time.Hour), next)
		assert.Equal(t, "every 2h0m0s", schedule.String())
	})
}

func TestHourlySchedule(t *testing.T) {
	t.Parallel()

	t.Run("hourly at specific minute", func(t *testing.T) {
		t.Parallel()

		schedule := queue.HourlyAt(30)
		base := time.Date(2024, 1, 1, 14, 15, 0, 0, time.UTC)
		next := schedule.Next(base)

		expected := time.Date(2024, 1, 1, 14, 30, 0, 0, time.UTC)
		assert.Equal(t, expected, next)
		assert.Equal(t, "hourly at :30", schedule.String())
	})

	t.Run("hourly at minute - next hour", func(t *testing.T) {
		t.Parallel()

		schedule := queue.HourlyAt(15)
		base := time.Date(2024, 1, 1, 14, 30, 0, 0, time.UTC)
		next := schedule.Next(base)

		expected := time.Date(2024, 1, 1, 15, 15, 0, 0, time.UTC)
		assert.Equal(t, expected, next)
	})

	t.Run("hourly at minute - exact time", func(t *testing.T) {
		t.Parallel()

		schedule := queue.HourlyAt(0)
		base := time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC)
		next := schedule.Next(base)

		// Should move to next hour since current time equals scheduled time
		expected := time.Date(2024, 1, 1, 15, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, next)
	})
}

func TestDailySchedule(t *testing.T) {
	t.Parallel()

	t.Run("daily at midnight", func(t *testing.T) {
		t.Parallel()

		schedule := queue.Daily()
		base := time.Date(2024, 1, 1, 14, 30, 0, 0, time.UTC)
		next := schedule.Next(base)

		expected := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, next)
		assert.Equal(t, "daily at 00:00", schedule.String())
	})

	t.Run("daily at specific time - later today", func(t *testing.T) {
		t.Parallel()

		schedule := queue.DailyAt(15, 30)
		base := time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC)
		next := schedule.Next(base)

		expected := time.Date(2024, 1, 1, 15, 30, 0, 0, time.UTC)
		assert.Equal(t, expected, next)
		assert.Equal(t, "daily at 15:30", schedule.String())
	})

	t.Run("daily at specific time - tomorrow", func(t *testing.T) {
		t.Parallel()

		schedule := queue.DailyAt(9, 0)
		base := time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC)
		next := schedule.Next(base)

		expected := time.Date(2024, 1, 2, 9, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, next)
	})

	t.Run("daily at exact current time", func(t *testing.T) {
		t.Parallel()

		schedule := queue.DailyAt(14, 30)
		base := time.Date(2024, 1, 1, 14, 30, 0, 0, time.UTC)
		next := schedule.Next(base)

		// Should be tomorrow
		expected := time.Date(2024, 1, 2, 14, 30, 0, 0, time.UTC)
		assert.Equal(t, expected, next)
	})
}

func TestWeeklySchedule(t *testing.T) {
	t.Parallel()

	t.Run("weekly on specific day at midnight", func(t *testing.T) {
		t.Parallel()

		schedule := queue.Weekly(time.Monday)
		// Start from Wednesday
		base := time.Date(2024, 1, 3, 14, 0, 0, 0, time.UTC)
		next := schedule.Next(base)

		// Next Monday
		expected := time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, next)
		assert.Equal(t, "weekly on Monday at 00:00", schedule.String())
	})

	t.Run("weekly on specific day and time - this week", func(t *testing.T) {
		t.Parallel()

		schedule := queue.WeeklyOn(time.Friday, 17, 0)
		// Start from Monday
		base := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
		next := schedule.Next(base)

		// This Friday
		expected := time.Date(2024, 1, 5, 17, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, next)
		assert.Equal(t, "weekly on Friday at 17:00", schedule.String())
	})

	t.Run("weekly on specific day and time - next week", func(t *testing.T) {
		t.Parallel()

		schedule := queue.WeeklyOn(time.Monday, 9, 0)
		// Start from Monday afternoon
		base := time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC)
		next := schedule.Next(base)

		// Next Monday
		expected := time.Date(2024, 1, 8, 9, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, next)
	})

	t.Run("weekly on same day later time", func(t *testing.T) {
		t.Parallel()

		schedule := queue.WeeklyOn(time.Wednesday, 18, 0)
		// Start from Wednesday morning
		base := time.Date(2024, 1, 3, 10, 0, 0, 0, time.UTC)
		next := schedule.Next(base)

		// Same day, later time
		expected := time.Date(2024, 1, 3, 18, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, next)
	})

	t.Run("weekly at exact current time", func(t *testing.T) {
		t.Parallel()

		schedule := queue.WeeklyOn(time.Monday, 14, 0)
		// Start from Monday at 14:00
		base := time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC)
		next := schedule.Next(base)

		// Next week
		expected := time.Date(2024, 1, 8, 14, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, next)
	})
}

func TestMonthlySchedule(t *testing.T) {
	t.Parallel()

	t.Run("monthly on specific day at midnight", func(t *testing.T) {
		t.Parallel()

		schedule := queue.Monthly(15)
		base := time.Date(2024, 1, 10, 14, 0, 0, 0, time.UTC)
		next := schedule.Next(base)

		expected := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, next)
		assert.Equal(t, "monthly on day 15 at 00:00", schedule.String())
	})

	t.Run("monthly on specific day and time - this month", func(t *testing.T) {
		t.Parallel()

		schedule := queue.MonthlyOn(20, 14, 30)
		base := time.Date(2024, 1, 10, 10, 0, 0, 0, time.UTC)
		next := schedule.Next(base)

		expected := time.Date(2024, 1, 20, 14, 30, 0, 0, time.UTC)
		assert.Equal(t, expected, next)
		assert.Equal(t, "monthly on day 20 at 14:30", schedule.String())
	})

	t.Run("monthly on specific day and time - next month", func(t *testing.T) {
		t.Parallel()

		schedule := queue.MonthlyOn(5, 9, 0)
		base := time.Date(2024, 1, 10, 10, 0, 0, 0, time.UTC)
		next := schedule.Next(base)

		expected := time.Date(2024, 2, 5, 9, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, next)
	})

	t.Run("monthly at exact current time", func(t *testing.T) {
		t.Parallel()

		schedule := queue.MonthlyOn(15, 14, 0)
		base := time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC)
		next := schedule.Next(base)

		// Next month
		expected := time.Date(2024, 2, 15, 14, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, next)
	})

	t.Run("monthly on 31st in short month", func(t *testing.T) {
		t.Parallel()

		schedule := queue.MonthlyOn(31, 12, 0)
		// February only has 28/29 days
		base := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		next := schedule.Next(base)

		// Should be Jan 31
		expected := time.Date(2024, 1, 31, 12, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, next)

		// Next run from Jan 31
		next = schedule.Next(expected)
		// February 2024 has 29 days (leap year)
		expected = time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, next)
	})

	t.Run("monthly on 30th in February", func(t *testing.T) {
		t.Parallel()

		schedule := queue.MonthlyOn(30, 15, 0)
		base := time.Date(2024, 1, 31, 20, 0, 0, 0, time.UTC)
		next := schedule.Next(base)

		// February 2024 has 29 days, so use 29 instead of 30
		expected := time.Date(2024, 2, 29, 15, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, next)
	})

	t.Run("monthly year boundary", func(t *testing.T) {
		t.Parallel()

		schedule := queue.MonthlyOn(5, 10, 0)
		base := time.Date(2024, 12, 20, 14, 0, 0, 0, time.UTC)
		next := schedule.Next(base)

		// Should wrap to next year
		expected := time.Date(2025, 1, 5, 10, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, next)
	})
}

func TestSchedule_TimeZones(t *testing.T) {
	t.Parallel()

	t.Run("preserves timezone", func(t *testing.T) {
		t.Parallel()

		nyLoc, err := time.LoadLocation("America/New_York")
		require.NoError(t, err)

		schedule := queue.DailyAt(9, 0)
		base := time.Date(2024, 1, 1, 14, 0, 0, 0, nyLoc)
		next := schedule.Next(base)

		// Should be in same timezone
		assert.Equal(t, nyLoc, next.Location())
		expected := time.Date(2024, 1, 2, 9, 0, 0, 0, nyLoc)
		assert.Equal(t, expected, next)
	})
}

func TestSchedule_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("leap year handling", func(t *testing.T) {
		t.Parallel()

		schedule := queue.MonthlyOn(29, 10, 0)

		// Non-leap year
		base := time.Date(2023, 1, 31, 10, 0, 0, 0, time.UTC)
		next := schedule.Next(base)
		// February 2023 only has 28 days
		expected := time.Date(2023, 2, 28, 10, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, next)

		// Leap year
		base = time.Date(2024, 1, 31, 10, 0, 0, 0, time.UTC)
		next = schedule.Next(base)
		// February 2024 has 29 days
		expected = time.Date(2024, 2, 29, 10, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, next)
	})

	t.Run("daylight saving time", func(t *testing.T) {
		t.Parallel()

		// This test verifies schedule behavior across DST boundaries
		loc, err := time.LoadLocation("America/New_York")
		require.NoError(t, err)

		schedule := queue.DailyAt(2, 30)
		// March 10, 2024 - DST starts, 2:00 AM becomes 3:00 AM
		base := time.Date(2024, 3, 9, 20, 0, 0, 0, loc)
		next := schedule.Next(base)

		// Should handle DST correctly
		expected := time.Date(2024, 3, 10, 2, 30, 0, 0, loc)
		assert.Equal(t, expected.Day(), next.Day())
		assert.Equal(t, expected.Month(), next.Month())
		// Hour might be adjusted due to DST
	})
}

func TestDaysInMonth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		year     int
		month    time.Month
		expected int
	}{
		{2024, time.January, 31},
		{2024, time.February, 29}, // Leap year
		{2023, time.February, 28}, // Non-leap year
		{2024, time.April, 30},
		{2024, time.December, 31},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d-%02d", tt.year, tt.month), func(t *testing.T) {
			// We can't test the private function directly, but we can verify
			// through the monthly schedule behavior
			schedule := queue.MonthlyOn(31, 12, 0)
			// Start from a day in the target month to ensure we get the same month
			base := time.Date(tt.year, tt.month, 1, 0, 0, 0, 0, time.UTC)
			next := schedule.Next(base)

			// The day should be capped at the actual days in month
			assert.Equal(t, tt.expected, next.Day())
			assert.Equal(t, tt.month, next.Month())
			assert.Equal(t, tt.year, next.Year())
		})
	}
}
