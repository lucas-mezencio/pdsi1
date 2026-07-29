package prescription

import (
	"testing"
	"time"
)

func newTestPrescription(createdAt time.Time, meds ...Medicament) *Prescription {
	return &Prescription{
		ID:          "pres-1",
		UserID:      "user-1",
		MedicID:     "doc-1",
		Medicaments: meds,
		Active:      true,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}
}

func TestBrazilLocation(t *testing.T) {
	if BrazilLocation == nil {
		t.Fatal("BrazilLocation must not be nil")
	}
	name, offset := time.Date(2026, 7, 29, 8, 0, 0, 0, BrazilLocation).Zone()
	if name != "BRT" {
		t.Errorf("zone name = %q, want BRT", name)
	}
	if offset != -3*60*60 {
		t.Errorf("offset = %d, want -10800", offset)
	}
}

func TestPrescription_ExpandSchedule_EmptyMedicaments(t *testing.T) {
	p := newTestPrescription(time.Now())
	got := p.ExpandSchedule(BrazilLocation)
	if len(got) != 0 {
		t.Errorf("expected empty, got %d entries", len(got))
	}
}

func TestPrescription_ExpandSchedule_OnceDailyThreeDoses(t *testing.T) {
    // Anchor at start of tomorrow so all 3 slots are guaranteed future.
    tomorrow := time.Now().In(BrazilLocation).AddDate(0, 0, 1)
    createdAt := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, BrazilLocation)
    p := newTestPrescription(createdAt, Medicament{
        Name:      "AAS",
        Dosage:    "100mg",
        Frequency: "24:00",
        Times:     []string{"08:00"},
        Doses:     3,
    })

	got := p.ExpandSchedule(BrazilLocation)
	if len(got) != 3 {
		t.Fatalf("expected 3 slots, got %d", len(got))
	}
	for i, slot := range got {
		if slot.Status != DoseStatusPending {
			t.Errorf("slot %d status = %q, want PENDING", i, slot.Status)
		}
		if slot.MedicamentName != "AAS" {
			t.Errorf("slot %d medicament = %q, want AAS", i, slot.MedicamentName)
		}
		if slot.Dosage != "100mg" {
			t.Errorf("slot %d dosage = %q, want 100mg", i, slot.Dosage)
		}
		if slot.ScheduledAt.Location().String() != "BRT" {
			t.Errorf("slot %d location = %q, want BRT", i, slot.ScheduledAt.Location().String())
		}
		_, off := slot.ScheduledAt.Zone()
		if off != -3*60*60 {
			t.Errorf("slot %d offset = %d, want -10800", i, off)
		}
if slot.ScheduledAt.Hour() != 8 || slot.ScheduledAt.Minute() != 0 {
			t.Errorf("slot %d wall clock = %02d:%02d, want 08:00", i, slot.ScheduledAt.Hour(), slot.ScheduledAt.Minute())
		}
	}

	for i := 1; i < len(got); i++ {
		gap := got[i].ScheduledAt.Sub(got[i-1].ScheduledAt)
		if gap != 24*time.Hour {
			t.Errorf("gap[%d-%d] = %v, want 24h", i, i-1, gap)
		}
	}
}

func TestPrescription_ExpandSchedule_TwiceDailyFourDoses(t *testing.T) {
    tomorrow := time.Now().In(BrazilLocation).AddDate(0, 0, 1)
    createdAt := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, BrazilLocation)
    p := newTestPrescription(createdAt, Medicament{
        Name:      "Lisinopril",
        Dosage:    "10mg",
        Frequency: "12:00",
        Times:     []string{"08:00", "20:00"},
        Doses:     4,
    })

	got := p.ExpandSchedule(BrazilLocation)
	if len(got) != 4 {
		t.Fatalf("expected 4 slots, got %d", len(got))
	}
	wantHours := []int{8, 20, 8, 20}
	for i, slot := range got {
		if slot.ScheduledAt.Hour() != wantHours[i] {
			t.Errorf("slot %d hour = %d, want %d", i, slot.ScheduledAt.Hour(), wantHours[i])
		}
	}
}

func TestPrescription_ExpandSchedule_AnchorsAtCreatedAt(t *testing.T) {
	tomorrow := time.Now().In(BrazilLocation).AddDate(0, 0, 1)
	createdAt := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 6, 0, 0, 0, BrazilLocation)
	p := newTestPrescription(createdAt, Medicament{
		Name: "AAS", Dosage: "100mg",
		Frequency: "24:00", Times: []string{"08:00"}, Doses: 2,
	})
	got := p.ExpandSchedule(BrazilLocation)
	if len(got) != 2 {
		t.Fatalf("expected 2 slots, got %d", len(got))
	}
	first := got[0].ScheduledAt
	if first.Day() != tomorrow.Day() || first.Hour() != 8 {
		t.Errorf("first slot = day %d hour %d, want day %d hour 8", first.Day(), first.Hour(), tomorrow.Day())
	}
}

func TestPrescription_ExpandSchedule_SkipsPastSlots(t *testing.T) {
	createdAt := time.Date(2020, 1, 1, 0, 0, 0, 0, BrazilLocation)
	p := newTestPrescription(createdAt, Medicament{
		Name: "AAS", Dosage: "100mg",
		Frequency: "24:00", Times: []string{"08:00"}, Doses: 1,
	})
	got := p.ExpandSchedule(BrazilLocation)
	if len(got) != 0 {
		t.Errorf("expected past slot to be filtered, got %d entries", len(got))
	}
}

func TestPrescription_ExpandSchedule_MultipleMedicaments(t *testing.T) {
	tomorrow := time.Now().In(BrazilLocation).AddDate(0, 0, 1)
	createdAt := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, BrazilLocation)
	p := newTestPrescription(createdAt,
		Medicament{Name: "AAS", Dosage: "100mg", Frequency: "24:00", Times: []string{"08:00"}, Doses: 2},
		Medicament{Name: "Lisinopril", Dosage: "10mg", Frequency: "12:00", Times: []string{"08:00", "20:00"}, Doses: 2},
	)
	got := p.ExpandSchedule(BrazilLocation)
	if len(got) != 4 {
		t.Errorf("expected 4 slots (2+2), got %d", len(got))
	}
}
