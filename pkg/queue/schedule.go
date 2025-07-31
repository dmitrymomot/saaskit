package queue

import (
	"fmt"
	"time"
)

// Schedule determines when a periodic task should run
type Schedule interface {
	Next(from time.Time) time.Time
	String() string
}

// intervalSchedule runs at fixed intervals
type intervalSchedule struct {
	every time.Duration
}

func (s intervalSchedule) Next(from time.Time) time.Time {
	return from.Add(s.every)
}

func (s intervalSchedule) String() string {
	return fmt.Sprintf("every %v", s.every)
}

// dailySchedule runs once per day at specified time
type dailySchedule struct {
	hour   int
	minute int
}

func (s dailySchedule) Next(from time.Time) time.Time {
	next := time.Date(
		from.Year(), from.Month(), from.Day(),
		s.hour, s.minute, 0, 0, from.Location(),
	)
	if !next.After(from) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

func (s dailySchedule) String() string {
	return fmt.Sprintf("daily at %02d:%02d", s.hour, s.minute)
}

// weeklySchedule runs once per week on specified day and time
type weeklySchedule struct {
	weekday time.Weekday
	hour    int
	minute  int
}

func (s weeklySchedule) Next(from time.Time) time.Time {
	// Calculate days until target weekday (handles week wraparound with modulo)
	daysUntil := (int(s.weekday) - int(from.Weekday()) + 7) % 7

	next := from.AddDate(0, 0, daysUntil)
	next = time.Date(
		next.Year(), next.Month(), next.Day(),
		s.hour, s.minute, 0, 0, next.Location(),
	)

	if !next.After(from) {
		next = next.AddDate(0, 0, 7) // Next week
	}
	return next
}

func (s weeklySchedule) String() string {
	return fmt.Sprintf("weekly on %s at %02d:%02d", s.weekday, s.hour, s.minute)
}

// monthlySchedule runs once per month on specified day and time
type monthlySchedule struct {
	day    int
	hour   int
	minute int
}

// hourlySchedule runs every hour at specified minute
type hourlySchedule struct {
	minute int
}

func (s hourlySchedule) Next(from time.Time) time.Time {
	next := time.Date(
		from.Year(), from.Month(), from.Day(),
		from.Hour(), s.minute, 0, 0, from.Location(),
	)
	if !next.After(from) {
		next = next.Add(time.Hour)
	}
	return next
}

func (s hourlySchedule) String() string {
	return fmt.Sprintf("hourly at :%02d", s.minute)
}

func (s monthlySchedule) Next(from time.Time) time.Time {
	year, month := from.Year(), from.Month()

	// Handle month-end overflow (e.g., requesting 31st of February becomes 28th/29th)
	day := min(s.day, daysInMonth(year, month))
	next := time.Date(year, month, day, s.hour, s.minute, 0, 0, from.Location())

	if !next.After(from) {
		// Move to next month
		if month == time.December {
			year++
			month = time.January
		} else {
			month++
		}

		// Recalculate day for new month
		day = min(s.day, daysInMonth(year, month))
		next = time.Date(year, month, day, s.hour, s.minute, 0, 0, from.Location())
	}

	return next
}

func (s monthlySchedule) String() string {
	return fmt.Sprintf("monthly on day %d at %02d:%02d", s.day, s.hour, s.minute)
}

// Factory functions for creating schedules

// EveryInterval creates a schedule that runs at fixed intervals
func EveryInterval(d time.Duration) Schedule {
	return intervalSchedule{every: d}
}

// EveryMinutes creates a schedule that runs every n minutes
func EveryMinutes(n int) Schedule {
	return intervalSchedule{every: time.Duration(n) * time.Minute}
}

// EveryHours creates a schedule that runs every n hours
func EveryHours(n int) Schedule {
	return intervalSchedule{every: time.Duration(n) * time.Hour}
}

// DailyAt creates a schedule that runs daily at specified time
func DailyAt(hour, minute int) Schedule {
	return dailySchedule{hour: hour, minute: minute}
}

// WeeklyOn creates a schedule that runs weekly on specified day and time
func WeeklyOn(weekday time.Weekday, hour, minute int) Schedule {
	return weeklySchedule{weekday: weekday, hour: hour, minute: minute}
}

// MonthlyOn creates a schedule that runs monthly on specified day and time
func MonthlyOn(day, hour, minute int) Schedule {
	return monthlySchedule{day: day, hour: hour, minute: minute}
}

// Convenience factories

// EveryMinute creates a schedule that runs every minute
func EveryMinute() Schedule {
	return intervalSchedule{every: time.Minute}
}

// Hourly creates a schedule that runs every hour at :00
func Hourly() Schedule {
	return intervalSchedule{every: time.Hour}
}

// HourlyAt creates a schedule that runs every hour at specified minute
func HourlyAt(minute int) Schedule {
	return hourlySchedule{minute: minute}
}

// Daily creates a schedule that runs daily at midnight
func Daily() Schedule {
	return dailySchedule{hour: 0, minute: 0}
}

// Weekly creates a schedule that runs weekly on specified day at midnight
func Weekly(weekday time.Weekday) Schedule {
	return weeklySchedule{weekday: weekday, hour: 0, minute: 0}
}

// Monthly creates a schedule that runs monthly on specified day at midnight
func Monthly(day int) Schedule {
	return monthlySchedule{day: day, hour: 0, minute: 0}
}

// Helper function to get days in month
func daysInMonth(year int, month time.Month) int {
	// Get first day of next month, then subtract one day
	firstOfNext := time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC)
	lastOfMonth := firstOfNext.AddDate(0, 0, -1)
	return lastOfMonth.Day()
}
