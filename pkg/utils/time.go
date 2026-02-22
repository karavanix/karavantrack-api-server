package utils

import "time"

// IsBeforeEndDate checks if the current date is before or equal the specified date.
func IsBeforeOrEqual(current, date time.Time) bool {
	currentYear, currentMonth, currentDay := current.Date()
	endYear, endMonth, endDay := date.Date()

	if currentYear < endYear {
		return true
	} else if currentYear > endYear {
		return false
	}

	// Years are equal, compare months
	if currentMonth < endMonth {
		return true
	} else if currentMonth > endMonth {
		return false
	}

	// Months are also equal, compare days
	return currentDay <= endDay
}

// IsBeforeEndDate checks if the current date is before the specified date.
func IsBefore(current, date time.Time) bool {
	currentYear, currentMonth, currentDay := current.Date()
	endYear, endMonth, endDay := date.Date()

	if currentYear < endYear {
		return true
	} else if currentYear > endYear {
		return false
	}

	// Years are equal, compare months
	if currentMonth < endMonth {
		return true
	} else if currentMonth > endMonth {
		return false
	}

	// Months are also equal, compare days
	return currentDay < endDay
}

// IsAfterEndDate checks if the current date is after or equal the specified date.
func IsAfterOrEqual(current, date time.Time) bool {
	currentYear, currentMonth, currentDay := current.Date()
	endYear, endMonth, endDay := date.Date()

	if currentYear > endYear {
		return true
	} else if currentYear < endYear {
		return false
	}

	// Years are equal, compare months
	if currentMonth > endMonth {
		return true
	} else if currentMonth < endMonth {
		return false
	}

	// Months are also equal, compare days
	return currentDay >= endDay
}

// IsAfterEndDate checks if the current date is after the specified date.
func IsAfter(current, date time.Time) bool {
	currentYear, currentMonth, currentDay := current.Date()
	endYear, endMonth, endDay := date.Date()

	if currentYear > endYear {
		return true
	} else if currentYear < endYear {
		return false
	}

	// Years are equal, compare months
	if currentMonth > endMonth {
		return true
	} else if currentMonth < endMonth {
		return false
	}

	// Months are also equal, compare days
	return currentDay > endDay
}

// IsTomorrow checks if the given time is tomorrow
func IsTomorrow(t time.Time) bool {
	now := time.Now()
	startOfTomorrow := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, 1)
	endOfTomorrow := startOfTomorrow.AddDate(0, 0, 1)

	return t.After(startOfTomorrow) && t.Before(endOfTomorrow)
}

// IsToday checks if the given time 't' is today.
func IsToday(t time.Time) bool {
	// Get the current time with respect to the system's local location
	now := time.Now()

	// Compare the year, month, and day of 'now' and 't'
	return now.Year() == t.Year() && now.Month() == t.Month() && now.Day() == t.Day()
}

// StartOfDate formats date time in 00:00:00
func StartOfDate(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
}

// EndOfDate formats date time in 23:59:59
func EndOfDate(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 0, date.Location())
}

// GetStartDateByPeriod returns the start date based on the period string (day, week, month).
// Returns nil if period is unknown or empty.
func GetStartDateByPeriod(period string, now time.Time) *time.Time {
	var from time.Time
	switch period {
	case "day":
		from = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "week":
		// Monday is the start of the week
		offset := int(now.Weekday())
		if offset == 0 {
			offset = 7
		}
		offset--
		from = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -offset)
	case "month":
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	default:
		return nil
	}
	return &from
}
