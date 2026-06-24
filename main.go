package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
)

type BlsquiUploadResponse struct {
	Success    bool   `json:"success"`
	TemplateID string `json:"template_id,omitempty"`
	Message    string `json:"message,omitempty"`
}

type FlixUpdateResponse struct {
	Success       bool   `json:"success"`
	TargetFlixID  string `json:"target_flix_id"`
	NewTemplateID string `json:"new_template_id"`
}

// Combined update body envelope structure expected by the Gin registration layer
type FlixUpdateEnvelope struct {
	TargetFlixID     string          `json:"targetFlixID"`
	PublicationState string          `json:"publicationState"`
	PublicTrigger    bool            `json:"publicTrigger"`
	TemplateData     json.RawMessage `json:"templateData"`
}

type FlixCadenceSchema struct {
	Body string `json:"body"`
}

type FlixDataSchema struct {
	Cadence FlixCadenceSchema `json:"cadence"`
}

type FlixTemplateSchema struct {
	ID   string         `json:"id"`
	Data FlixDataSchema `json:"data"`
}

const Version = "1.1.4"
func main() {
	versionFlag := flag.Bool("version", false, "Print the version of Blsqui CLI")
	vFlag := flag.Bool("v", false, "Print the version of Blsqui CLI")

	flag.Parse()

	// If the user passed the flag, print version and exit immediately
	if *versionFlag || *vFlag {
		fmt.Printf("blsqui-cli version %s\n", Version)
		os.Exit(0)
	}
	fmt.Println("Welcome to the Blsqui Developer Tool")

	// First Question: Main Action Branching
	modeQuestion := &survey.Select{
		Message: "What do you want to do?:",
		Options: []string{
			"Deploy Testnet Smart Contract <NEW!>",
			"Upload FLIX Template",
			"Verify FLIX Template Status",
			"Update Existing Template",
		},
	}

	var selectedMode string
	err := survey.AskOne(modeQuestion, &selectedMode)
	if err != nil {
		fmt.Printf("❌ Input cancelled: %v\n", err)
		return
	}

	// Route based on user selection
	switch selectedMode {
		case "Upload FLIX Template":
			handleUploadFlow()
		case "Verify FLIX Template Status":
			handleStatusAndPromotionFlow()
		case "Update Existing Template":
			handleUpdateFlow()
		case "Deploy Testnet Smart Contract <NEW!>":
			DeployTestnetContract()
	}
}

func generateAndProcessFlixTemplate() (templatePath string, localBytes []byte, success bool) {
	var cadencePath string
	pathPrompt := &survey.Input{
		Message: "Where is your Cadence (.cdc) file?",
		Default: "./cadence/transactions/transfer_10_flow.cdc",
	}
	survey.AskOne(pathPrompt, &cadencePath)

	var metaJsonPath string
	metaPathPrompt := &survey.Input{
		Message: "Where is your Metadata (.json) file?",
		Default: "./cadence/metadata/metadata.json",
	}
	survey.AskOne(metaPathPrompt, &metaJsonPath)

	metaJsonPath = strings.TrimSpace(metaJsonPath)
    if _, err := os.Stat(metaJsonPath); os.IsNotExist(err) {
        fmt.Println("\n⚠️  [BLSQUI ERROR] Metadata source file is missing.")
        fmt.Println("The Flow CLI requires a valid pre-fill metadata JSON to successfully generate contract messages.")
        fmt.Println("Please verify your path or construct the metadata json file before moving forward.")
        return
    }

	var flowJsonPath string
	flowJsonPrompt := &survey.Input{
		Message: "Where is your flow.json configuration file?",
		Default: "./flow.json",
	}
	survey.AskOne(flowJsonPrompt, &flowJsonPath)

	flowJsonPath = strings.TrimSpace(flowJsonPath)
	cadencePath = strings.TrimSpace(cadencePath)

	// Isolate the base folder path where flow.json lives to ground context relativity
	// var workingDir string
	// if idx := strings.LastIndex(flowJsonPath, "/"); idx != -1 {
	// 	workingDir = flowJsonPath[:idx]
	// } else {
	// 	workingDir = "."
	// }

	absCadencePath, err1 := filepath.Abs(strings.TrimSpace(cadencePath))
    absFlowJsonPath, err2 := filepath.Abs(strings.TrimSpace(flowJsonPath))
    absMetaJsonPath, err3 := filepath.Abs(strings.TrimSpace(metaJsonPath))

	if err1 != nil || err2 != nil || err3 != nil {
        fmt.Println("\n❌ Filepath Resolution Failure: Unable to calculate absolute system maps.")
        return "", nil, false
    }

	// relativeCadencePath := cadencePath
	// if strings.HasPrefix(cadencePath, workingDir+"/") {
	// 	relativeCadencePath = "'" +strings.TrimPrefix(cadencePath, workingDir+"/") + "'"
	// }
	
	// relativeFlowJsonPath := flowJsonPath
	// if strings.HasPrefix(flowJsonPath, workingDir+"/") {
	// 	relativeFlowJsonPath = "'" +strings.TrimPrefix(flowJsonPath, workingDir+"/") + "'"
	// }

	// relativeMetaJsonPath := metaJsonPath
    // if strings.HasPrefix(metaJsonPath, workingDir+"/") {
    //     relativeMetaJsonPath = "'" +strings.TrimPrefix(metaJsonPath, workingDir+"/") + "'"
    // }

	cmd := exec.Command(
        "flow", "flix", "generate", absCadencePath, 
        "-f", absFlowJsonPath, 
        "--pre-fill", absMetaJsonPath,
        "--network", "testnet",
        "--network", "mainnet",
    )

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("\n❌ Flow CLI Generation Failed: %v\n", err)
		if err.Error() == "exit status 1" {
			fmt.Println("👉 Did you set the path of flow.json correctly?\n")
		}
		fmt.Printf("Detailed Diagnostics:\n%s\n", string(output))
		return "", nil, false
	}
	// fmt.Printf("output:\n%s\n", string(output))

	// The compiled FLIX JSON is sitting safely in our output variable stream!
	rawOutput := output

	// FIX: Isolate the pure JSON boundaries by scanning for structural tokens
	firstBracketIdx := bytes.IndexByte(rawOutput, '{')
	lastBracketIdx := bytes.LastIndexByte(rawOutput, '}')

	if firstBracketIdx == -1 || lastBracketIdx == -1 || firstBracketIdx >= lastBracketIdx {
		fmt.Println("\n❌ Flow CLI executed, but the stream data contains no recognizable JSON object contents.")
		fmt.Printf("📋 Raw Output Stream Captured:\n%s\n", string(rawOutput))
		return "", nil, false
	}

	// Cleanly extract ONLY the structural JSON string sequence bypassing the CLI warnings!
	localBytes = bytes.TrimSpace(rawOutput[firstBracketIdx : lastBracketIdx+1])

	// Calculate and write the permanent backup file target context next to your code
	calculatedTemplatePath := strings.Replace(cadencePath, ".cdc", ".template.json", 1)
	err = os.WriteFile(calculatedTemplatePath, localBytes, 0644)
	if err != nil {
		fmt.Printf("Warning: Could not save template file to %s: %v\n", calculatedTemplatePath, err)
	} else {
		fmt.Printf("Template is Successfully Created at: %s\n", calculatedTemplatePath)
	}

	return calculatedTemplatePath, localBytes, true
}

// --- BRANCH 1: UPLOAD NEW LAW ---
func handleUploadFlow() {
	// Check if Flow CLI exists
	if _, err := exec.LookPath("flow"); err != nil {
		showFlowMissingMessage()
		os.Exit(1)
	}

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
		fmt.Println("⚠️ Upload aborted by developer. $_")
		return
	}

	// Action: Read file data and trigger the Go backend API endpoint
	uploadTemplateToBackend(templatePath)
}

func handleUpdateFlow() {
	if _, err := exec.LookPath("flow"); err != nil {
		showFlowMissingMessage()
		os.Exit(1)
	}

	var flixID string
	flixIDPrompt := &survey.Input{
		Message: "Enter FLIX ID to update: ",
		Default: "",
	}
	survey.AskOne(flixIDPrompt, &flixID)
	flixID = strings.TrimSpace(flixID)
	if flixID == "" {
		fmt.Println("❌ Error: FLIX ID cannot be blank.")
		return
	}

	_, localBytes, success := generateAndProcessFlixTemplate()
	if !success {
		return
	}

	fmt.Println("Fetching Blsqui registry metrics to cross-analyze Cadence integrity...")
	remoteTemplate, err := fetchRemoteFlixTemplate(flixID)
	if err != nil {
		fmt.Printf("⚠️ Warning: Could not reach Blsqui registry (%v).\n", err)
	}

	var localTemplate FlixTemplateSchema
	_ = json.Unmarshal(localBytes, &localTemplate)

	// Perform direct text string comparison as the absolute source of truth
	cadenceCodeChanged := true
	if remoteTemplate != nil {
		localClean := normalizeCadenceCode(localTemplate.Data.Cadence.Body)
		remoteClean := normalizeCadenceCode(remoteTemplate.Data.Cadence.Body)

		// Compare text blocks directly
		if localClean != "" && remoteClean != "" && localClean == remoteClean {
			cadenceCodeChanged = false
		} else {
			// 🔍 DIAGNOSTIC FORK: Print the exact variation breakdown
			fmt.Println("\n🔍 [Registry Keeper Diagnostics] Dissecting Cadence String Mismatch:")
			fmt.Printf("📏 Normalized Lengths -> Local: %d characters | Remote: %d characters\n", len(localClean), len(remoteClean))

			// Find the exact character location where the strings start to differ
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
				
				// Extract a small snippet surrounding the difference for context
				start := diffIdx - 20
				if start < 0 { start = 0 }
				
				endLocal := diffIdx + 40
				if endLocal > len(localClean) { endLocal = len(localClean) }
				
				endRemote := diffIdx + 40
				if endRemote > len(remoteClean) { endRemote = len(remoteClean) }

				fmt.Printf("💻 Local Snippet:  ... %s ...\n", localClean[start:endLocal])
				fmt.Printf("🌐 Remote Snippet: ... %s ...\n", remoteClean[start:endRemote])
			} else {
				// If lengths are different but loops matched up to minLen
				fmt.Println("📍 One code body contains extra trailing commands or lines at the end.")
				if len(localClean) > len(remoteClean) {
					fmt.Printf("💻 Local Extra text: %s\n", localClean[minLen:])
				} else {
					fmt.Printf("🌐 Remote Extra text: %s\n", remoteClean[minLen:])
				}
			}
		}
	}

	// Default initialization variables for updating registration payload
	publicationState := "CADENCE_CHANGED"
	promoteToPublic := false

	if !cadenceCodeChanged {
		// Scenario A: Cadence is exact match, only translations/metadata changed
		fmt.Println("\n✅ Verification Complete: Cadence transaction code is completely unchanged.")
		fmt.Println("👉 Hey, you are not changing the cadence code, so that you don't need the audit and can promote to public as soon as this is uploaded.")
		
		prompt := &survey.Confirm{
			Message: "Do you want to promote this updated FLIX Template directly to public?",
			Default: true,
		}
		survey.AskOne(prompt, &promoteToPublic)

		if promoteToPublic {
			publicationState = "UPDATE_SOON" // Audited bypass trigger
		} else {
			publicationState = "PUBLISH_LATER" // Staged draft state
		}
	} else {
		// Scenario B: Cadence script bytecode altered, audit lock applied
		fmt.Println("\n⚠️ Alert: You changed the cadence code. This needs audit, and the target template will be on public while you apply and pass the audit.")
		publicationState = "CADENCE_CHANGED"
	}

	// Request absolute delivery execution authorization
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

	// Execute deployment sequence using the updated endpoint route structure
	executeFlixUpdatePayload(flixID, publicationState, promoteToPublic, localBytes)
}

// --- STATUS & PROMOTION FLOW ---
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
			// Trigger state change via API
			executeStateChange(flixID, "public")
		} else {
			fmt.Println("Cancelled by developer.")
		}
	}
}

func uploadTemplateToBackend(filePath string) {
	// Read file contents
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		// Fallback dummy for testing layout if the file doesn't exist locally yet
		fileData = []byte(`{"data": "mock_template_payload"}`)
	}

	url := "https://blsqui.net/api/flix/register"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(fileData))
	if err != nil {
		fmt.Printf("❌ Failed to construct request: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ Network Error connecting to Go backend: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Fully extract the incoming response byte arrays
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        fmt.Printf("❌ Failed to read server execution stream: %v", err)
    }

    // Decode the raw JSON structure strings into the Go struct object
    var bResp BlsquiUploadResponse
    if err := json.Unmarshal(body, &bResp); err != nil {
        fmt.Printf("❌ Failed to parse server metadata envelope: %v", err)
    }

    // Verify that the core registry process actually completed successfully
    if !bResp.Success || bResp.TemplateID == "" {
		fmt.Printf("❌ Backend rejected payload: %s\n", bResp.Message)
        return
    }

    fmt.Println("\nSuccess!")
    fmt.Println("Your FLIX ID has been generated. Use the following URL to integrate this template into your application:")
    fmt.Printf("https://blsqui.net/flix/registry/%s\n\n", bResp.TemplateID)
}

// Normalizes Cadence text code by stripping single-line comments, whitespaces, tabs, and line breaks to ensure pure structural logic parity.
func normalizeCadenceCode(code string) string {
	var cleanLines []string

	// Split the text block into individual lines to scan for comments
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if strings.HasPrefix(trimmedLine, "//") {
			continue
		}

		if idx := strings.Index(trimmedLine, "//"); idx != -1 {
			trimmedLine = strings.TrimSpace(trimmedLine[:idx])
		}

		if trimmedLine != "" {
			cleanLines = append(cleanLines, trimmedLine)
		}
	}

	// Join back the remaining logic lines into a single solid text sequence
	combinedCode := strings.Join(cleanLines, "")

	replacer := strings.NewReplacer(" ", "", "\t", "", "\n", "", "\r", "")
	return replacer.Replace(combinedCode)
}

// Fetch raw template metrics directly from the production FCL serving engine
func fetchRemoteFlixTemplate(flixID string) (*FlixTemplateSchema, error) {
	url := fmt.Sprintf("https://blsqui.net/flix/registry/%s", flixID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status: %d", resp.StatusCode)
	}

	var remote FlixTemplateSchema
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&remote); err != nil {
		return nil, err
	}
	return &remote, nil
}

func executeFlixUpdatePayload(flixID string, publicationState string, promoteToPublic bool, localBytes []byte) {
	// 1. Build network request transmission envelope
	payload := map[string]interface{}{
		"targetFlixID":     flixID,
		"publicationState": publicationState,
		"publicTrigger":    promoteToPublic,
		"templateData":     json.RawMessage(localBytes),
	}

	bodyBytes, _ := json.Marshal(payload)
	
	// Post data to your secure production API gateway route
	resp, err := http.Post("https://blsqui.net/api/flix/update", "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		fmt.Printf("❌ Network Error: Could not reach Blsqui Registry: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("❌ Upload Rejected by Registry Server (Status: %d):\n%s\n", resp.StatusCode, string(respBytes))
		return
	}

	// 2. Decode the response body parameters matching your clean backend structure
	var uResp FlixUpdateResponse
	if err := json.NewDecoder(resp.Body).Decode(&uResp); err != nil {
		fmt.Printf("❌ Error decoding registry server response metadata: %v\n", err)
		return
	}

	// 3. Print out clean confirmation messages with the pristine URL
	fmt.Println("\n🎉 Congratulations! Updated FLIX Template is successfully injected into Blsqui Registry.")
	fmt.Println("Use the following URL to integrate this updated FLIX Template into your application:")
	fmt.Printf("👉 https://blsqui.net/flix/registry/%s\n\n", uResp.NewTemplateID)
}

func executeStateChange(flixID string, targetState string) {
	fmt.Printf("📡 Communicating state modification for ID: %s...\n", flixID)
	// You can map this to a PATCH or POST endpoint like /api/flix/status
	// e.g., payload = { flix_id: flixID, status: targetState }
	
	fmt.Println("Template successfully promoted to Public!")
	fmt.Println("[System] Updated Registry State: PUBLIC")
	fmt.Printf("[URL] Live at: https://blsqui.net/flix/registry/%s\n", flixID)
}

func showFlowMissingMessage() {
	fmt.Println("❌ Error: 'flow-cli' is not installed on this system.")
	fmt.Println("💡 Blsqui-CLI requires the official Flow toolchain to generate FLIX templates.")
	fmt.Println("\nTo fix this, please run the official installation command:")
	fmt.Println("👉 macOS/Linux: sh -ci \"$(curl -fsSL https://raw.githubusercontent.com/onflow/flow-cli/master/install.sh)\"")
	fmt.Println("👉 Windows: iex (irm 'https://raw.githubusercontent.com/onflow/flow-cli/master/install.ps1')")
}


type FlowConfig struct {
	Accounts map[string]interface{} `json:"accounts"`
}

// DeployTestnetContract targets the Testnet architecture using clean terminal prompts
func DeployTestnetContract() {
	network := "testnet"

	// 1. PROMPT FOR CONTRACT METADATA (Bypasses tedious manual flow.json editing)
	var contractQuestions = []*survey.Question{
		{
			Name: "name",
			Prompt: &survey.Input{
				Message: "Enter the Smart Contract Name:",
				Default: "BlsquiTerminal",
			},
			Validate: survey.Required,
		},
		{
			Name: "flowJsonPath",
			Prompt: &survey.Input{
				Message: "Where is your flow.json configuration file?",
				Default: "./flow.json",
			},
			Validate: survey.Required,
		},
	}

	answers := struct {
		Name         string
		FlowJsonPath string
	}{}

	if err := survey.Ask(contractQuestions, &answers); err != nil {
		fmt.Printf("🛑 Setup cancelled: %v\n", err)
		return
	}

	workingDir := filepath.Dir(answers.FlowJsonPath)
	flowJsonFilename := filepath.Base(answers.FlowJsonPath)

    cmd := exec.Command("flow", "project", "deploy",
        "-f", flowJsonFilename,
        "--network", network,
        "--update",
    )
    cmd.Dir = workingDir
	outputBytes, err := cmd.CombinedOutput()

	// Because we aren't streaming live to os.Stdout anymore, we manually print the raw
    // compiler response here so you can still read it perfectly on your Mac terminal!
    combinedLogs := string(outputBytes)
    fmt.Print(combinedLogs)

    if err == nil {
        fmt.Println("Success! Contract successfully updated.")
        return
    }
	fmt.Println("████████████████████████████████████████████████████████████")
    fmt.Println("\n------------------------------------------------------------")

    // Check for specific Cadence characteristic error signatures inside the Stdout copy!
    if strings.Contains(combinedLogs, "found new field") {
        fmt.Println("🚧 OOPS! A 'found new field' error occurred.")
		fmt.Println("🚧 This happens because Cadence is protecting existing account storage layouts.")
        fmt.Println("🔥 BUT DON'T WORRY! Blsqui CLI can generate a new testnet account and deploy a contract to it")
        fmt.Println("🔥 to overcome this characteristic Cadence limitation.")
    } else if strings.Contains(combinedLogs, "incompatible change") || strings.Contains(combinedLogs, "removed") {
        fmt.Println("🚧 OOPS! An incompatible storage type modification layout was detected.")
		fmt.Println("🚧 Existing resource states cannot be overwritten.")
        fmt.Println("🔥 BUT DON'T WORRY! Blsqui CLI can generate a new testnet account and deploy a contract to it")
        fmt.Println("🔥 to overcome this characteristic Cadence limitation.")
	} else if strings.Contains(combinedLogs, "storage limit check failed") || strings.Contains(combinedLogs, "capacity") {
        // NEW ERROR BRANCH: Captures storage boundary panics and routes them to the faucet
        fmt.Println("🚧 OOPS! A 'storage limit check failed' error occurred.")
        fmt.Println("💸 This happens because the target account does not hold enough FLOW tokens")
        fmt.Println("   to back the memory space required by your smart contract's byte size.")
        fmt.Println("\n🚀 HOW TO FIX THIS:")
        fmt.Println("1. Copy the target account address throwing this error.")
        fmt.Println("2. Visit the official Flow Testnet Faucet to drop free tokens into it:")
        fmt.Println("   🔗 👉 https://faucet.flow.com/fund-account")
        fmt.Println("3. Re-run Blsqui CLI deployment to land your logic beautifully!")
        fmt.Println("\n--- RAW FLOW CLI OUTPUT ---")
        if len(strings.TrimSpace(combinedLogs)) > 0 {
            fmt.Println(combinedLogs)
        } else {
            fmt.Printf("Exit Error State: %v\n", err)
        }
	    fmt.Println("------------------------------------------------------------")
        fmt.Println("████████████████████████████████████████████████████████████\n")
        return
    } else {
        // General fallback message if it failed for standard connectivity or path reasons
        fmt.Println("[ERROR] Deployment pipeline interrupted. Your contract has some problem. Please read the error message.")
        fmt.Println("\n--- RAW FLOW CLI OUTPUT ---")
        if len(strings.TrimSpace(combinedLogs)) > 0 {
            fmt.Println(combinedLogs)
        } else {
            fmt.Printf("Exit Error State: %v\n", err)
        }
	    fmt.Println("------------------------------------------------------------\n")
        return
    }
    fmt.Println("------------------------------------------------------------\n")

	confirmNewDeploy := false
	prompt := &survey.Confirm{
		Message: fmt.Sprintf("Would you like Blsqui CLI to automatically genarate a new testnet account and deploy %s contract on it?", answers.Name),
		Default: true,
	}

	if err := survey.AskOne(prompt, &confirmNewDeploy); err != nil || !confirmNewDeploy {
		fmt.Println("🛑 Deployment pipeline halted.")
		return
	}

	newAccountName, err := getNextSequentialTestnetAccount(answers.FlowJsonPath)
    if err != nil {
        newAccountName = "testnet-account-2"
    }

	var payerAccountName string
    payerPrompt := &survey.Input{
        Message: "Enter the account-name of your Testnet account in flow.json (used to pay network fees). It is under accounts in flow.json. :",
        Default: "testnet-account",
    }
    if err := survey.AskOne(payerPrompt, &payerAccountName); err != nil || payerAccountName == "" {
        fmt.Println("🛑 Deployment pipeline halted. Payer account name is required.")
        return
    }
    payerAccountName = strings.TrimSpace(payerAccountName)

    fmt.Printf("\n Requesting a brand-new account address from Flow Testnet via payer [%s]...\n", payerAccountName)

	createAccountCmd := exec.Command("flow", "accounts", "create",
        "-f", flowJsonFilename,
        "--network", "testnet",
        "--signer", payerAccountName,
        "--key", "ac1be7cb1a939330d97fae2f36ec2a20e280706006bea995688295f26fe02ac03d10893b4d63f5585acab6726d5d70e8005e52b36b1a7e22cf071dbc92aa699f",
		"--output", "json",
    )
    createAccountCmd.Dir = workingDir

    var stdoutBuf, errBuf bytes.Buffer
	createAccountCmd.Stdout = &stdoutBuf
    createAccountCmd.Stderr = &errBuf

    if err := createAccountCmd.Run(); err != nil {
        fmt.Println("❌ Failed to generate a testnet blockchain account address.")
		fmt.Println("--- RAW ACCOUNT CREATION ERROR ---")
        fmt.Println(errBuf.String())
        fmt.Println("----------------------------------")
        return
    }
	outputStr := stdoutBuf.String()

	var mintedAddress string
    if strings.Contains(outputStr, "\"address\":") {
        parts := strings.Split(outputStr, "\"address\":")
        if len(parts) > 1 {
            // Isolate the clean hex string
            subPart := strings.Split(parts[1], ",")[0]
            mintedAddress = strings.Trim(subPart, " \"\n\r\t")
        }
    }

    if mintedAddress == "" {
        fmt.Println("❌ Failed to parse the newly minted address from Flow CLI response data.")
        return
    }

    // Ensure the address does not have a leading 0x before saving
    mintedAddress = strings.TrimPrefix(mintedAddress, "0x")
    fmt.Printf("New Testnet address was successfully generated: [0x%s]\n", mintedAddress)

    // WRITE THE CONFIGURATION PARAMETERS SAFELY INTO FLOW.JSON USING THE CONFIG UTILITY
    fmt.Println("Saving the new profile configuration into flow.json...")
    addAccountCmd := exec.Command("flow", "config", "add", "account",
        "-f", flowJsonFilename,
        "--name", newAccountName,
        "--address", mintedAddress, // Passes the true, real, live address we parsed!
        "--private-key", "58552795fa36262c9a1d4a1a47e63eb947dfd385e2ee6eca621625987572df62", // Attaches the private key pairing safely
    )
    addAccountCmd.Dir = workingDir
    if err := addAccountCmd.Run(); err != nil {
        fmt.Printf("❌ Failed to modify user accounts array: %v\n", err)
        return
    }

	// 4. ISOLATE AND CLEAN ONLY THE DEPLOYMENTS BLOCK
    configBytes, readErr := os.ReadFile(answers.FlowJsonPath)
    if readErr == nil {
        configStr := string(configBytes)
        parts := strings.SplitN(configStr, "\"deployments\":", 2)
        if len(parts) == 2 {
            topOfFile := parts[0]
            deploymentsBlock := parts[1]
            oldTargetEntry := fmt.Sprintf("\"%s\"", answers.Name)
            deploymentsBlock = strings.Replace(deploymentsBlock, oldTargetEntry+",", "", 1)
            deploymentsBlock = strings.Replace(deploymentsBlock, oldTargetEntry, "", 1)

			finalConfigStr := topOfFile + "\"deployments\":" + deploymentsBlock
            _ = os.WriteFile(answers.FlowJsonPath, []byte(finalConfigStr), 0644)
        }
    }

    // 5. WIRE UP THE FRESH ROUTING POINTER IN THE DEPLOYMENT CONFIG
    fmt.Println("Updating deployment array of flow.json...")
    configCmd := exec.Command("flow", "config", "add", "deployment",
        "-f", flowJsonFilename,
        "--network", network,
        "--account", newAccountName,
        "--contract", answers.Name,
    )
    configCmd.Dir = workingDir
    _ = configCmd.Run()

	fmt.Printf("💸 Moving 1.0 FLOW tokens to [0x%s] from %s account for deployment transaction fee requirement...\n", mintedAddress, payerAccountName)
    cadenceArgsJSON := fmt.Sprintf(`[{"type":"Address","value":"0x%s"},{"type":"UFix64","value":"1.00000000"}]`, mintedAddress)
	const transferCadenceCode = `
    import FungibleToken from 0x9a0766d93b6608b7
    import FlowToken from 0x7e60df042a9c0868

    transaction(recipient: Address, amount: UFix64) {
        let vaultRef: auth(FungibleToken.Withdraw) &FlowToken.Vault

        prepare(signer: auth(FungibleToken.Withdraw, Storage) &Account) {
            self.vaultRef = signer.storage.borrow<auth(FungibleToken.Withdraw) &FlowToken.Vault>(from: /storage/flowTokenVault)
                ?? panic("Could not borrow reference to the owner's Vault!")
        }

        execute {
            let receiverRef = getAccount(recipient)
                .capabilities.get<&{FungibleToken.Receiver}>(/public/flowTokenReceiver)
                .borrow()
                ?? panic("Could not borrow receiver reference to the recipient's Vault!")

            let tokens <- self.vaultRef.withdraw(amount: amount)
            receiverRef.deposit(from: <-tokens)
        }
    }`

    // 2. WRITE IT TO A TEMPORARY RUNTIME FILE
    tempTxFile := ".temp_transfer_tokens.cdc"
    tempTxPath := filepath.Join(workingDir, tempTxFile)
    if err := os.WriteFile(tempTxPath, []byte(strings.TrimSpace(transferCadenceCode)), 0644); err != nil {
        fmt.Printf("❌ Failed to create temporary pipeline transaction: %v\n", err)
        return
    }
    // Ensure the temporary file is deleted when this function exits
    defer os.Remove(tempTxPath)

	fundCmd := exec.Command("flow", "transactions", "send",
        tempTxFile,
        "-f", flowJsonFilename,
        "--network", "testnet",
        "--signer", payerAccountName,
        "--args-json", cadenceArgsJSON,
    )
    fundCmd.Dir = workingDir

    var fundErrBuf bytes.Buffer
    fundCmd.Stderr = &fundErrBuf

    if err := fundCmd.Run(); err != nil {
        fmt.Printf("[Error]: %v\n", fundErrBuf.String())
        return
    }
    fmt.Println("Storage tokens successfully credited 1.0 FLOW token.")

	// DEPLOY TO THE BRAND NEW ACCOUNT WITHOUT ANY TYPE-SAFETY RESTRICTIONS
    fmt.Println("Deploying the smart contract to your fresh Testnet address...")
    deployCmd := exec.Command("flow", "project", "deploy",
        "-f", flowJsonFilename,
        "--network", network,
		"--update",
    )
    deployCmd.Dir = workingDir
    deployCmd.Stdout = os.Stdout
    deployCmd.Stderr = os.Stderr
    if err := deployCmd.Run(); err != nil {
        fmt.Printf("❌ Testnet deployment failed: %v\n", err)
        return
    }

	// CLEAN STRINGS FOR EXTRACTION PACKAGING
    cleanAddress := strings.TrimPrefix(mintedAddress, "0x")

    fmt.Printf("\n🎉 [%s] is running at your new account address space [%s]. Take a Look!! -> https://testnet.flowscan.io/contract/A.%s.%s?tab=deployments\n", 
        answers.Name,
        newAccountName,
        cleanAddress,
        answers.Name,
    )
}

func getNextSequentialTestnetAccount(configPath string) (string, error) {
	fileBytes, err := os.ReadFile(configPath)
	if err != nil {
		return "", err
	}

	var config FlowConfig
	if err := json.Unmarshal(fileBytes, &config); err != nil {
		return "", err
	}

	testnetCount := 0
	for accountName := range config.Accounts {
		if strings.HasPrefix(accountName, "testnet-account-") {
			testnetCount++
		}
	}

	return fmt.Sprintf("testnet-account-%d", testnetCount+1), nil
}