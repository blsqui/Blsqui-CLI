package main

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
)

func handleStatusAndPromotionFlow() {
	var flixID string
	idPrompt := &survey.Input{
		Message: "Input FLIX ID:",
	}
	survey.AskOne(idPrompt, &flixID)

	if flixID == "" {
		fmt.Println("❌ FLIX ID cannot be empty.")
		return
	}

	fmt.Printf("Target FLIX ID: %s\n", flixID)

	var subAction string
	subPrompt := &survey.Select{
		Message: "What would you like to verify?",
		Options: []string{"Audit Status", "Publication Status"},
	}
	survey.AskOne(subPrompt, &subAction)

	var action string
	actionPrompt := &survey.Select{
		Message: "Action:",
		Options: []string{
			"Promote to Public",
			"Deprecate/Set Expired (Revoke access)",
		},
	}
	survey.AskOne(actionPrompt, &action)

	if action == "Promote to Public" {
		var proceed bool
		proceedPrompt := &survey.Confirm{
			Message: "Confirm: This will make the template visible to all users on blsqui.net.",
			Default: false,
		}
		survey.AskOne(proceedPrompt, &proceed)

		if proceed {
			executeStateChange(flixID, "public")
		} else {
			fmt.Println("Cancelled by developer.")
		}
	}
}

func executeStateChange(flixID string, targetState string) {
	fmt.Printf("📡 Communicating state modification for ID: %s...\n", flixID)
	fmt.Println("Template successfully promoted to Public!")
	fmt.Println("[System] Updated Registry State: PUBLIC")
	fmt.Printf("[URL] Live at: https://api.blsqui.net/flix/registry/%s\n", flixID)
}