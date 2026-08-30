package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/AlecAivazis/survey/v2"
)

func handleUploadFlow() {
	if _, err := exec.LookPath("flow"); err != nil {
		showFlowMissingMessage()
		os.Exit(1)
	}

	// Contextual Workflow Briefing
	fmt.Println("\n────────────────────────────────────────────────────────────")
	fmt.Println("📦 Generate & Register New FLIX Template (v1.1)")
	fmt.Println("────────────────────────────────────────────────────────────")
	fmt.Println("ℹ️  This command compiles your Cadence transaction & metadata")
	fmt.Println("   into an Interaction Template (FLIX) and registers it to Blsqui.")
	fmt.Println()
	fmt.Println("   Prerequisites:")
	fmt.Println("   • 1. Cadence file (.cdc) - First argument must be (nonce: String) if you use Blsqui SDK")
	fmt.Println("   • 2. Metadata file (.json) - Descriptions, icons, and i18n messages")
	fmt.Println("   • 3. flow.json - Project network aliases & contract deployment addresses")
	fmt.Println("        (Run 'flow init' if you haven't set up flow.json yet)")
	fmt.Println("\n📖 Step-by-step tutorial:")
	fmt.Println("   👉 https://blsqui.net/developer-guide/blsqui-cli")
	fmt.Println("────────────────────────────────────────────────────────────\n")

	templatePath, _, success := generateAndProcessFlixTemplate()
	if !success {
		return
	}

	var confirmUpload bool
	confirmPrompt := &survey.Confirm{
		Message: "Ready to upload to Blsqui Registry? (Y/n)",
		Default: true,
	}
	survey.AskOne(confirmPrompt, &confirmUpload)

	if !confirmUpload {
		fmt.Println("⚠️ Upload aborted by developer.")
		return
	}

	uploadTemplateToBackend(templatePath)
}

func handleUpdateFlow() {
	if _, err := exec.LookPath("flow"); err != nil {
		showFlowMissingMessage()
		os.Exit(1)
	}

	// Contextual Workflow Briefing
	fmt.Println("\n────────────────────────────────────────────────────────────")
	fmt.Println("🔄 Update FLIX Template To Match Your Preferred Style")
	fmt.Println("────────────────────────────────────────────────────────────")
	fmt.Println("ℹ️  This command regenerates and pushes an updated FLIX template.")
	fmt.Println("   • If you only modified metadata/IP (icons, titles, translations),")
	fmt.Println("     the update bypasses audits and goes live immediately.")
	fmt.Println("   • If Cadence code changed, it will stage for audit review.")
	fmt.Println("\n📖 Step-by-step tutorial:")
	fmt.Println("   👉 https://blsqui.net/developer-guide/blsqui-cli")
	fmt.Println("────────────────────────────────────────────────────────────\n")

	// Clear Input Prompt with Hex Length Validation
	var flixID string
	flixIDPrompt := &survey.Input{
		Message: "Enter the target FLIX ID to update:",
		Help:    "A 64-character hex ID (e.g., 8babf9501081633c2fad1247ceb910428d7e9fd69f00a000460d5dfd12e89ecc)",
	}

	err := survey.AskOne(flixIDPrompt, &flixID, survey.WithValidator(survey.Required), survey.WithValidator(func(val interface{}) error {
		str, ok := val.(string)
		if !ok || len(strings.TrimSpace(str)) == 0 {
			return fmt.Errorf("FLIX ID is required")
		}
		clean := strings.TrimSpace(str)
		if len(clean) != 64 {
			return fmt.Errorf("FLIX ID must be exactly 64 hexadecimal characters (got %d)", len(clean))
		}
		return nil
	}))

	if err != nil {
		fmt.Println("❌ Operation aborted.")
		return
	}

	flixID = strings.TrimSpace(flixID)

	// 3. Gentle Notice before asking for file paths
	fmt.Println("\n📁 Now preparing local sources for compilation...")
	fmt.Println("   You will need: (1) .cdc transaction, (2) metadata.json, (3) flow.json\n")

	_, localBytes, success := generateAndProcessFlixTemplate()
	if !success {
		return
	}

	fmt.Println("\nFetching Blsqui registry metrics to cross-analyze Cadence integrity...")
	remoteTemplate, err := fetchRemoteFlixTemplate(flixID)
	if err != nil {
		fmt.Printf("⚠️ Warning: Could not reach Blsqui registry (%v).\n", err)
	}

	var localTemplate FlixTemplateSchema
	_ = json.Unmarshal(localBytes, &localTemplate)

	cadenceCodeChanged := true
	if remoteTemplate != nil {
		localClean := normalizeCadenceCode(localTemplate.Data.Cadence.Body)
		remoteClean := normalizeCadenceCode(remoteTemplate.Data.Cadence.Body)

		if localClean != "" && remoteClean != "" && localClean == remoteClean {
			cadenceCodeChanged = false
		} else {
			fmt.Println("\n🔍 [Registry Keeper Diagnostics] Dissecting Cadence String Mismatch:")
			fmt.Printf("📏 Normalized Lengths -> Local: %d characters | Remote: %d characters\n", len(localClean), len(remoteClean))

			minLen := len(localClean)
			if len(remoteClean) < minLen {
				minLen = len(remoteClean)
			}

			diffIdx := -1
			for i := 0; i < minLen; i++ {
				if localClean[i] != remoteClean[i] {
					diffIdx = i
					break
				}
			}

			if diffIdx != -1 {
				fmt.Printf("📍 First difference found at index position: %d\n", diffIdx)
				start := diffIdx - 20
				if start < 0 { start = 0 }
				endLocal := diffIdx + 40
				if endLocal > len(localClean) { endLocal = len(localClean) }
				endRemote := diffIdx + 40
				if endRemote > len(remoteClean) { endRemote = len(remoteClean) }

				fmt.Printf("💻 Local Snippet:  ... %s ...\n", localClean[start:endLocal])
				fmt.Printf("🌐 Remote Snippet: ... %s ...\n", remoteClean[start:endRemote])
			} else {
				fmt.Println("📍 One code body contains extra trailing commands or lines at the end.")
				if len(localClean) > len(remoteClean) {
					fmt.Printf("💻 Local Extra text: %s\n", localClean[minLen:])
				} else {
					fmt.Printf("🌐 Remote Extra text: %s\n", remoteClean[minLen:])
				}
			}
		}
	}

	publicationState := "CADENCE_CHANGED"
	promoteToPublic := false

	if !cadenceCodeChanged {
		fmt.Println("\n✅ Verification Complete: Cadence transaction code is completely unchanged.")
		fmt.Println("👉 Hey, you are not changing the cadence code, so that you don't need the audit and can promote to public as soon as this is uploaded.")

		prompt := &survey.Confirm{
			Message: "Do you want to promote this updated FLIX Template directly to public?",
			Default: true,
		}
		survey.AskOne(prompt, &promoteToPublic)

		if promoteToPublic {
			publicationState = "UPDATE_SOON"
		} else {
			publicationState = "PUBLISH_LATER"
		}
	} else {
		fmt.Println("\n⚠️ Alert: You changed the cadence code. This needs audit, and the target template will be on public while you apply and pass the audit.")
		publicationState = "CADENCE_CHANGED"
	}

	var confirmUpload bool
	confirmPrompt := &survey.Confirm{
		Message: "Ready to upload to Blsqui Registry? (Y/n)",
		Default: true,
	}
	survey.AskOne(confirmPrompt, &confirmUpload)

	if !confirmUpload {
		fmt.Println("Upload aborted by developer.")
		return
	}

	executeFlixUpdatePayload(flixID, publicationState, promoteToPublic, localBytes, cadenceCodeChanged)
}