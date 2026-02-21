package main

import (
	"fmt"
	"strings"

	"github.com/tmc/macgo/codesign"
)

func runDoctor() error {
	identities, err := codesign.ListAvailableIdentities()
	if err != nil {
		return fmt.Errorf("listing identities: %w", err)
	}

	// Classify identities.
	var devIDs []string
	var appleDev []string
	var other []string
	for _, id := range identities {
		switch {
		case strings.Contains(id, "Developer ID Application"):
			devIDs = append(devIDs, id)
		case strings.Contains(id, "Apple Development"):
			appleDev = append(appleDev, id)
		default:
			other = append(other, id)
		}
	}

	best := codesign.FindBestIdentity()
	teamID := ""
	if best != "" {
		teamID = codesign.ExtractTeamIDFromCertificate(best)
	}

	// Print environment summary.
	fmt.Println("Signing Environment")
	fmt.Printf("  Keychain identities:  %d found\n", len(identities))
	if len(devIDs) > 0 {
		for _, id := range devIDs {
			fmt.Printf("  Developer ID:         %s\n", id)
		}
	} else {
		fmt.Println("  Developer ID:         none")
	}
	if len(appleDev) > 0 {
		for _, id := range appleDev {
			fmt.Printf("  Apple Development:    %s\n", id)
		}
	} else {
		fmt.Println("  Apple Development:    none")
	}
	for _, id := range other {
		fmt.Printf("  Other:                %s\n", id)
	}
	if teamID != "" {
		fmt.Printf("  Team ID:              %s\n", teamID)
	} else {
		fmt.Println("  Team ID:              none")
	}

	// Diagnosis.
	fmt.Println()
	fmt.Println("Diagnosis")
	if len(devIDs) == 0 {
		fmt.Println("  ! No Developer ID Application cert — Gatekeeper will block non-ad-hoc apps")
	}
	if best != "" {
		fmt.Printf("  . AutoSign will use: %s\n", best)
	} else {
		fmt.Println("  . AutoSign will use: ad-hoc (-)")
	}
	fmt.Println("  . Ad-hoc fallback available for LaunchServices-only apps")

	// Recommendations.
	if len(devIDs) == 0 {
		fmt.Println()
		fmt.Println("Recommendations")
		fmt.Println("  To sign for distribution: create a Developer ID Application cert")
		fmt.Println("    https://developer.apple.com/account/resources/certificates/list")
	}

	return nil
}
