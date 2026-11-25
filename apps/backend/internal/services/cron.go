package services

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// cronSchedule holds a five-field cron expression as sets of allowed values.
type cronSchedule struct {
	minute, hour, day, month, weekday [60]bool
	anyDay, anyWeekday                bool
}

var cronMacros = map[string]string{
	"@hourly":  "0 * * * *",
	"@daily":   "0 0 * * *",
	"@weekly":  "0 0 * * 0",
	"@monthly": "0 0 1 * *",
}

func parseCron(expression string) (cronSchedule, error) {
	var s cronSchedule
	expression = strings.TrimSpace(expression)
	if macro, ok := cronMacros[expression]; ok {
		expression = macro
	}
	fields := strings.Fields(expression)
	if len(fields) != 5 {
		return s, errors.New("schedule needs five fields: minute hour day month weekday")
	}
	for i, target := range []struct {
		set      *[60]bool
		min, max int
	}{{&s.minute, 0, 59}, {&s.hour, 0, 23}, {&s.day, 1, 31}, {&s.month, 1, 12}, {&s.weekday, 0, 7}} {
		if err := parseCronField(fields[i], target.min, target.max, target.set); err != nil {
			return s, err
		}
	}
	if s.weekday[7] {
		s.weekday[0] = true
	}
	s.anyDay, s.anyWeekday = fields[2] == "*", fields[4] == "*"
	return s, nil
}

func parseCronField(field string, min, max int, set *[60]bool) error {
	for _, part := range strings.Split(field, ",") {
		rangePart, stepPart, hasStep := strings.Cut(part, "/")
		step := 1
		if hasStep {
			var err error
			if step, err = strconv.Atoi(stepPart); err != nil || step < 1 {
				return errors.New("invalid step in schedule: " + part)
			}
		}
		low, high := min, max
		if rangePart != "*" {
			from, to, isRange := strings.Cut(rangePart, "-")
			var err error
			if low, err = strconv.Atoi(from); err != nil {
				return errors.New("invalid value in schedule: " + part)
			}
			high = low
			if isRange {
				if high, err = strconv.Atoi(to); err != nil {
					return errors.New("invalid range in schedule: " + part)
				}
			} else if hasStep {
				high = max
			}
		}
		if low < min || high > max || low > high {
			return errors.New("value out of range in schedule: " + part)
		}
		for value := low; value <= high; value += step {
			set[value] = true
		}
	}
	return nil
}

func (s cronSchedule) matchesDay(t time.Time) bool {
	day, weekday := s.day[t.Day()], s.weekday[int(t.Weekday())]
	switch {
	case s.anyDay && s.anyWeekday:
		return true
	case s.anyDay:
		return weekday
	case s.anyWeekday:
		return day
	default:
		return day || weekday
	}
}

// next returns the first matching minute strictly after after, in UTC.
func (s cronSchedule) next(after time.Time) time.Time {
	t := after.UTC().Truncate(time.Minute).Add(time.Minute)
	limit := t.AddDate(5, 0, 0)
	for t.Before(limit) {
		if !s.month[int(t.Month())] || !s.matchesDay(t) {
			t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, 1)
			continue
		}
		if !s.hour[t.Hour()] {
			t = t.Truncate(time.Hour).Add(time.Hour)
			continue
		}
		if !s.minute[t.Minute()] {
			t = t.Add(time.Minute)
			continue
		}
		return t
	}
	return limit
}
