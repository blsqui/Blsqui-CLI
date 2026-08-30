package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
)

const Version = "1.1.5"

func init() {
	// Globally override survey's error template with valid template syntax
	survey.ErrorTemplate = `{{- color "red"}}❌ {{.Error.Error}}{{color "reset"}}` + "\n"
}

func printBanner() {
	banner := `
  ____  _     ____   ___  _   _ ___ 
 | __ )| |   / ___| / _ \| | | |_ _|
 |  _ \| |   \___ \| | | | | | || | 
 | |_) | |___ ___) | |_| | |_| || | 
 |____/|_____|____/ \__\_\\___/|___|  v%s

  ⚡ Flow Game Engine SDK & FLIX Registry Suite
  ─────────────────────────────────────────────
`
	fmt.Printf(banner, Version)
}

func main() {
	versionFlag := flag.Bool("version", false, "Print the version of Blsqui CLI")
	vFlag := flag.Bool("v", false, "Print the version of Blsqui CLI")

	flag.Parse()

	if *versionFlag || *vFlag {
		fmt.Printf("blsqui-cli v%s\n", Version)
		os.Exit(0)
	}

	printBanner()

	// Descriptive Action Prompts
	const (
		optDeploy = "🚀 Deploy Contract   — Auto-deploy/migrate Cadence smart contracts to Testnet"
		optUpload = "📦 Upload FLIX       — Generate and register a new FLIX v1.1 template"
		optUpdate = "🔄 Update Template   — Sync IP branding, metadata changes, or updated Cadence"
		optStatus = "🔍 Verify Status     — Inspect audit progress and publish templates to public"
		optExit   = "🚪 Exit              — Close developer tool"
	)

	modeQuestion := &survey.Select{
		Message: "Select a development workflow:",
		Options: []string{
			optDeploy,
			optUpload,
			optUpdate,
			optStatus,
			optExit,
		},
		PageSize: 5,
	}

	var selectedMode string
	err := survey.AskOne(modeQuestion, &selectedMode, survey.WithIcons(func(icons *survey.IconSet) {
		icons.Question.Text = "⚡"
		icons.SelectFocus.Text = "▶"
	}))

	if err != nil {
		fmt.Println("\n👋 Session closed.")
		return
	}

	switch selectedMode {
	case optDeploy:
		DeployTestnetContract()
	case optUpload:
		handleUploadFlow()
	case optUpdate:
		handleUpdateFlow()
	case optStatus:
		handleStatusAndPromotionFlow()
	case optExit:
		fmt.Println("👋 Exited Blsqui CLI. Happy building!")
		os.Exit(0)
	}
}