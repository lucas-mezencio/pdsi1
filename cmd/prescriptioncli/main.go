package main

import (
	"flag"
	"fmt"
	"os"
)

// TODO: replace with the UUID you seeded for Dr. Test Silva
// (the doctor row must already exist in postgres before this CLI can submit).
const defaultMedicID = "11111111-1111-1111-1111-111111111111"

const defaultAPI = "http://localhost:8080/api/v1"

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

	fmt.Printf("prescriptioncli starting\n")
	fmt.Printf("  api:     %s\n", *apiURL)
	fmt.Printf("  medic:   %s\n", *medicID)
	fmt.Printf("  secret:  <set, %d chars>\n", len(*secret))
}
