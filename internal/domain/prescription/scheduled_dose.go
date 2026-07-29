package prescription

import "time"

// BrazilLocation is the fixed timezone used for medication schedule
// wall-clock interpretation in CareConnect. Brazil / BRT, UTC-3, no DST.
// Defined as a package-level constant so the API and scheduler agree
// if/when we want to unify them on a single source of truth.
var BrazilLocation = time.FixedZone("BRT", -3*60*60)

// ScheduledDose is one slot in a user's medication schedule. It is
// produced by expanding a prescription's medicaments and overlaid with
// existing dose_records so that confirmed / missed status wins. JSON
// tags match the existing DoseRecord pattern so the API can return the
// struct directly without a DTO.
type ScheduledDose struct {
	PrescriptionID string     `json:"prescription_id"`
	MedicamentName string     `json:"medicament_name"`
	Dosage         string     `json:"dosage"`
	ScheduledAt    time.Time  `json:"scheduled_at"`
	Status         DoseStatus `json:"status"`
	DoseRecordID   *string    `json:"dose_record_id,omitempty"`
	ConfirmedAt    *time.Time `json:"confirmed_at,omitempty"`
}

// ExpandSchedule emits one ScheduledDose per planned dose for the
// prescription, anchored to CreatedAt and stepped by the medicament's
// frequency / times. Wall-clock times in medicament.Times are
// interpreted in loc. Slots strictly in the past (before now) are
// skipped so callers can rely on the result to represent upcoming or
// current doses.
func (p *Prescription) ExpandSchedule(loc *time.Location) []ScheduledDose {
	now := time.Now()
	var out []ScheduledDose
	for _, m := range p.Medicaments {
		interval := intervalForMedicament(m)
		first, err := firstScheduleTimeAfter(p.CreatedAt, m.Times, interval, loc)
		if err != nil || len(m.Times) == 0 {
			continue
		}
		for i := 0; i < m.Doses; i++ {
			at := first.Add(time.Duration(i) * interval)
			if at.Before(now) {
				continue
			}
			out = append(out, ScheduledDose{
				PrescriptionID: p.ID,
				MedicamentName: m.Name,
				Dosage:         m.Dosage,
				ScheduledAt:    at,
				Status:         DoseStatusPending,
			})
		}
	}
	return out
}

// intervalForMedicament returns the duration between scheduled doses for
// a medicament, derived from its Frequency string. Mirrors the scheduler
// helper at internal/infrastructure/scheduler/redis_scheduler.go but lives
// in the domain so the API can reuse it without pulling the scheduler
// package as a dependency.
func intervalForMedicament(m Medicament) time.Duration {
	if m.Frequency != "" {
		d, err := parseClockDuration(m.Frequency)
		if err == nil && d > 0 {
			return d
		}
	}
	return 24 * time.Hour
}

// firstScheduleTimeAfter returns the first time at or after start at
// which any of times falls, interpreted in loc. The result is the
// earliest of each `parseClockTime(t)` on or after start.
func firstScheduleTimeAfter(start time.Time, times []string, interval time.Duration, loc *time.Location) (time.Time, error) {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	var earliest time.Time
	for _, ts := range times {
		h, mi, s, err := parseClockTime(ts)
		if err != nil {
			return time.Time{}, err
		}
		candidate := time.Date(start.Year(), start.Month(), start.Day(), h, mi, s, 0, loc)
		for candidate.Before(start) {
			candidate = candidate.Add(interval)
		}
		if earliest.IsZero() || candidate.Before(earliest) {
			earliest = candidate
		}
	}
	if earliest.IsZero() {
		return time.Time{}, ErrInvalidTimes
	}
	return earliest, nil
}
