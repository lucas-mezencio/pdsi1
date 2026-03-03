package prescription

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Medicament represents a medication with dosage and schedule information
type Medicament struct {
	Name      string   `json:"name"`
	Dosage    string   `json:"dosage"`
	Frequency string   `json:"frequency"` // Duration between doses (e.g., "24:00", "12:00", "08:00")
	Times     []string `json:"time"`      // Specific times to take medication (e.g., ["08:00", "20:00"])
	Doses     int      `json:"doses"`     // Total number of doses (e.g., 14 for "twice daily for a week")
}

// Validate validates the medicament fields
func (m *Medicament) Validate() error {
	if m.Name == "" {
		return ErrInvalidMedicamentName
	}
	if m.Dosage == "" {
		return ErrInvalidDosage
	}
	if m.Frequency == "" {
		return ErrInvalidFrequency
	}
	if len(m.Times) == 0 {
		return ErrInvalidTimes
	}
	if m.Doses <= 0 {
		return ErrInvalidDoses
	}

	// Validate frequency format (HH:MM) - can be up to 24:00
	if err := validateFrequencyFormat(m.Frequency); err != nil {
		return fmt.Errorf("invalid frequency format: %w", err)
	}

	// Validate all times (must be valid clock times 00:00-23:59)
	for _, t := range m.Times {
		if err := validateTimeFormat(t); err != nil {
			return fmt.Errorf("invalid time format '%s': %w", t, err)
		}
	}

	// Validate that frequency matches number of times
	if err := m.validateFrequencyConsistency(); err != nil {
		return err
	}

	return nil
}

// validateTimeFormat validates that a time string is in HH:MM format (00:00-23:59)
func validateTimeFormat(timeStr string) error {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return ErrInvalidTimeFormat
	}

	// Require exactly 2 digits for hours
	if len(parts[0]) != 2 {
		return ErrInvalidTimeFormat
	}

	hours, err := strconv.Atoi(parts[0])
	if err != nil || hours < 0 || hours > 23 {
		return ErrInvalidTimeFormat
	}

	// Require exactly 2 digits for minutes
	if len(parts[1]) != 2 {
		return ErrInvalidTimeFormat
	}

	minutes, err := strconv.Atoi(parts[1])
	if err != nil || minutes < 0 || minutes > 59 {
		return ErrInvalidTimeFormat
	}

	return nil
}

// validateFrequencyFormat validates frequency format (can be 00:00 to 24:00)
func validateFrequencyFormat(freqStr string) error {
	parts := strings.Split(freqStr, ":")
	if len(parts) != 2 {
		return ErrInvalidTimeFormat
	}

	// Require exactly 2 digits for hours
	if len(parts[0]) != 2 {
		return ErrInvalidTimeFormat
	}

	hours, err := strconv.Atoi(parts[0])
	if err != nil || hours < 0 || hours > 24 {
		return ErrInvalidTimeFormat
	}

	// Require exactly 2 digits for minutes
	if len(parts[1]) != 2 {
		return ErrInvalidTimeFormat
	}

	minutes, err := strconv.Atoi(parts[1])
	if err != nil || minutes < 0 || minutes > 59 {
		return ErrInvalidTimeFormat
	}

	// If hours is 24, minutes must be 00
	if hours == 24 && minutes != 0 {
		return ErrInvalidTimeFormat
	}

	return nil
}

// validateFrequencyConsistency checks if frequency matches the number of times per day
func (m *Medicament) validateFrequencyConsistency() error {
	freqParts := strings.Split(m.Frequency, ":")
	freqHours, _ := strconv.Atoi(freqParts[0])
	freqMinutes, _ := strconv.Atoi(freqParts[1])

	frequencyDuration := time.Duration(freqHours)*time.Hour + time.Duration(freqMinutes)*time.Minute

	// Calculate expected number of doses per day
	expectedDoses := int(24 * time.Hour / frequencyDuration)

	if len(m.Times) != expectedDoses {
		return fmt.Errorf("%w: frequency %s suggests %d doses per day, but %d times provided",
			ErrFrequencyTimesMismatch, m.Frequency, expectedDoses, len(m.Times))
	}

	return nil
}

// GetNextNotificationTime calculates the next notification time for this medicament
func (m *Medicament) GetNextNotificationTime(now time.Time) (time.Time, error) {
	if len(m.Times) == 0 {
		return time.Time{}, ErrInvalidTimes
	}

	currentTime := now.Format("15:04")

	// Find the next time today
	for _, t := range m.Times {
		if t > currentTime {
			return parseTimeToday(now, t)
		}
	}

	// If no time found today, return first time tomorrow
	tomorrow := now.Add(24 * time.Hour)
	return parseTimeToday(tomorrow, m.Times[0])
}

// CalculateEndDate calculates when the medicament prescription ends based on start date and total doses
func (m *Medicament) CalculateEndDate(startDate time.Time) time.Time {
	if m.Doses <= 0 || len(m.Times) == 0 {
		return startDate
	}

	// Calculate how many days needed to complete all doses
	dosesPerDay := len(m.Times)
	totalDays := (m.Doses + dosesPerDay - 1) / dosesPerDay // Ceiling division

	// Add days to start date
	return startDate.AddDate(0, 0, totalDays-1)
}

// CalculateDaysRemaining calculates how many days remain in the prescription
func (m *Medicament) CalculateDaysRemaining(startDate time.Time, now time.Time) int {
	endDate := m.CalculateEndDate(startDate)
	if now.After(endDate) {
		return 0
	}

	duration := endDate.Sub(now)
	days := int(duration.Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

// IsCompleted checks if all doses have been taken based on start date and current time
func (m *Medicament) IsCompleted(startDate time.Time, now time.Time) bool {
	endDate := m.CalculateEndDate(startDate)
	return now.After(endDate)
}

// parseTimeToday parses a time string (HH:MM) and returns it as a time.Time for the given date
func parseTimeToday(date time.Time, timeStr string) (time.Time, error) {
	parts := strings.Split(timeStr, ":")
	hours, _ := strconv.Atoi(parts[0])
	minutes, _ := strconv.Atoi(parts[1])

	return time.Date(
		date.Year(), date.Month(), date.Day(),
		hours, minutes, 0, 0,
		date.Location(),
	), nil
}

// Domain errors
var (
	ErrInvalidMedicamentName  = errors.New("invalid medicament name")
	ErrInvalidDosage          = errors.New("invalid dosage")
	ErrInvalidFrequency       = errors.New("invalid frequency")
	ErrInvalidTimes           = errors.New("invalid times")
	ErrInvalidDoses           = errors.New("invalid doses: must be greater than 0")
	ErrInvalidTimeFormat      = errors.New("time must be in HH:MM format")
	ErrFrequencyTimesMismatch = errors.New("frequency does not match number of times")
)
