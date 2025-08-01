package validator

import (
	"fmt"
	"time"
)

func PastDate(field string, value time.Time) Rule {
	return Rule{
		Check: func() bool {
			return value.Before(time.Now())
		},
		Error: ValidationError{
			Field:          field,
			Message:        "date must be in the past",
			TranslationKey: "validation.date_past",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

func FutureDate(field string, value time.Time) Rule {
	return Rule{
		Check: func() bool {
			return value.After(time.Now())
		},
		Error: ValidationError{
			Field:          field,
			Message:        "date must be in the future",
			TranslationKey: "validation.date_future",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

func DateAfter(field string, value time.Time, after time.Time) Rule {
	return Rule{
		Check: func() bool {
			return value.After(after)
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("date must be after %s", after.Format("2006-01-02")),
			TranslationKey: "validation.date_after",
			TranslationValues: map[string]any{
				"field": field,
				"after": after.Format("2006-01-02"),
			},
		},
	}
}

func DateBefore(field string, value time.Time, before time.Time) Rule {
	return Rule{
		Check: func() bool {
			return value.Before(before)
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("date must be before %s", before.Format("2006-01-02")),
			TranslationKey: "validation.date_before",
			TranslationValues: map[string]any{
				"field":  field,
				"before": before.Format("2006-01-02"),
			},
		},
	}
}

func DateBetween(field string, value time.Time, start time.Time, end time.Time) Rule {
	return Rule{
		Check: func() bool {
			return (value.Equal(start) || value.After(start)) && (value.Equal(end) || value.Before(end))
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("date must be between %s and %s", start.Format("2006-01-02"), end.Format("2006-01-02")),
			TranslationKey: "validation.date_between",
			TranslationValues: map[string]any{
				"field": field,
				"start": start.Format("2006-01-02"),
				"end":   end.Format("2006-01-02"),
			},
		},
	}
}

// MinAge validates minimum age by calculating years elapsed, accounting for leap years and exact dates.
func MinAge(field string, birthdate time.Time, minAge int) Rule {
	return Rule{
		Check: func() bool {
			now := time.Now()
			age := now.Year() - birthdate.Year()

			// Adjust if birthday hasn't occurred this year
			if now.Month() < birthdate.Month() ||
				(now.Month() == birthdate.Month() && now.Day() < birthdate.Day()) {
				age--
			}

			return age >= minAge
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("minimum age of %d years required", minAge),
			TranslationKey: "validation.min_age",
			TranslationValues: map[string]any{
				"field":   field,
				"min_age": minAge,
			},
		},
	}
}

func MaxAge(field string, birthdate time.Time, maxAge int) Rule {
	return Rule{
		Check: func() bool {
			now := time.Now()
			age := now.Year() - birthdate.Year()

			// Adjust if birthday hasn't occurred this year
			if now.Month() < birthdate.Month() ||
				(now.Month() == birthdate.Month() && now.Day() < birthdate.Day()) {
				age--
			}

			return age <= maxAge
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("maximum age of %d years exceeded", maxAge),
			TranslationKey: "validation.max_age",
			TranslationValues: map[string]any{
				"field":   field,
				"max_age": maxAge,
			},
		},
	}
}

func AgeBetween(field string, birthdate time.Time, minAge int, maxAge int) Rule {
	return Rule{
		Check: func() bool {
			now := time.Now()
			age := now.Year() - birthdate.Year()

			// Adjust if birthday hasn't occurred this year
			if now.Month() < birthdate.Month() ||
				(now.Month() == birthdate.Month() && now.Day() < birthdate.Day()) {
				age--
			}

			return age >= minAge && age <= maxAge
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("age must be between %d and %d years", minAge, maxAge),
			TranslationKey: "validation.age_between",
			TranslationValues: map[string]any{
				"field":   field,
				"min_age": minAge,
				"max_age": maxAge,
			},
		},
	}
}

func BusinessHours(field string, value time.Time, startHour int, endHour int) Rule {
	return Rule{
		Check: func() bool {
			hour := value.Hour()
			return hour >= startHour && hour < endHour
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("time must be within business hours (%02d:00 - %02d:00)", startHour, endHour),
			TranslationKey: "validation.business_hours",
			TranslationValues: map[string]any{
				"field":      field,
				"start_hour": startHour,
				"end_hour":   endHour,
			},
		},
	}
}

func WorkingDay(field string, value time.Time) Rule {
	return Rule{
		Check: func() bool {
			weekday := value.Weekday()
			return weekday >= time.Monday && weekday <= time.Friday
		},
		Error: ValidationError{
			Field:          field,
			Message:        "date must be a working day (Monday-Friday)",
			TranslationKey: "validation.working_day",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

func Weekend(field string, value time.Time) Rule {
	return Rule{
		Check: func() bool {
			weekday := value.Weekday()
			return weekday == time.Saturday || weekday == time.Sunday
		},
		Error: ValidationError{
			Field:          field,
			Message:        "date must be a weekend (Saturday-Sunday)",
			TranslationKey: "validation.weekend",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

// TimeAfter compares time-of-day only, ignoring date component.
func TimeAfter(field string, value time.Time, after time.Time) Rule {
	return Rule{
		Check: func() bool {
			valueTime := value.Hour()*3600 + value.Minute()*60 + value.Second()
			afterTime := after.Hour()*3600 + after.Minute()*60 + after.Second()
			return valueTime > afterTime
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("time must be after %s", after.Format("15:04:05")),
			TranslationKey: "validation.time_after",
			TranslationValues: map[string]any{
				"field": field,
				"after": after.Format("15:04:05"),
			},
		},
	}
}

func TimeBefore(field string, value time.Time, before time.Time) Rule {
	return Rule{
		Check: func() bool {
			valueTime := value.Hour()*3600 + value.Minute()*60 + value.Second()
			beforeTime := before.Hour()*3600 + before.Minute()*60 + before.Second()
			return valueTime < beforeTime
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("time must be before %s", before.Format("15:04:05")),
			TranslationKey: "validation.time_before",
			TranslationValues: map[string]any{
				"field":  field,
				"before": before.Format("15:04:05"),
			},
		},
	}
}

func TimeBetween(field string, value time.Time, start time.Time, end time.Time) Rule {
	return Rule{
		Check: func() bool {
			valueTime := value.Hour()*3600 + value.Minute()*60 + value.Second()
			startTime := start.Hour()*3600 + start.Minute()*60 + start.Second()
			endTime := end.Hour()*3600 + end.Minute()*60 + end.Second()
			return valueTime >= startTime && valueTime <= endTime
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("time must be between %s and %s", start.Format("15:04:05"), end.Format("15:04:05")),
			TranslationKey: "validation.time_between",
			TranslationValues: map[string]any{
				"field": field,
				"start": start.Format("15:04:05"),
				"end":   end.Format("15:04:05"),
			},
		},
	}
}

// ValidBirthdate ensures reasonable birthdate constraints: not future, not older than 150 years.
func ValidBirthdate(field string, value time.Time) Rule {
	return Rule{
		Check: func() bool {
			now := time.Now()
			if value.After(now) {
				return false
			}
			// 150 years is reasonable maximum human lifespan
			maxAge := now.AddDate(-150, 0, 0)
			return value.After(maxAge)
		},
		Error: ValidationError{
			Field:          field,
			Message:        "birthdate must be a valid date not in the future and not more than 150 years ago",
			TranslationKey: "validation.valid_birthdate",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}
