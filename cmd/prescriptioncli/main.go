package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// TODO: replace with the UUID you seeded for Dr. Test Silva
// (the doctor row must already exist in postgres before this CLI can submit).
const defaultMedicID = "11111111-1111-1111-1111-111111111111"

const defaultAPI = "https://careconnect.lmezencio.dev/api/v1"

func main() {
	secret := flag.String("secret", "", "DEMO_PRESCRIPTION_SECRET value (raw, sent as Authorization header)")
	apiURL := flag.String("api", defaultAPI, "API base URL")
	medicID := flag.String("medic-id", defaultMedicID, "doctor UUID creating the prescription")
	flag.Parse()

	if *secret == "" {
		fmt.Fprintln(os.Stderr, "error: -secret is required (raw DEMO_PRESCRIPTION_SECRET value)")
		flag.Usage()
		os.Exit(2)
	}

	api := New(*apiURL, *secret)
	SetCurrentAPI(&api)
	SetCurrentMedicID(medicID)

	p := tea.NewProgram(NewModel(api), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
