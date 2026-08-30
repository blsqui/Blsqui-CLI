package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

func showFlowMissingMessage() {
	fmt.Println("❌ Error: 'flow-cli' is not installed on this system.")
	fmt.Println("💡 Blsqui-CLI requires the official Flow toolchain to generate FLIX templates.")
	fmt.Println("\nTo fix this, please run the official installation command:")
	fmt.Println("👉 macOS/Linux: sh -ci \"$(curl -fsSL https://raw.githubusercontent.com/onflow/flow-cli/master/install.sh)\"")
	fmt.Println("👉 Windows: iex (irm 'https://raw.githubusercontent.com/onflow/flow-cli/master/install.ps1')")
}

func generateAndProcessFlixTemplate() (templatePath string, localBytes []byte, success bool) {
	// Cadence File Path with instant validation
	var cadencePath string
	pathPrompt := &survey.Input{
		Message: "Where is your Cadence (.cdc) file?",
		Default: "./cadence/transactions/transfer_10_flow.cdc",
	}
	survey.AskOne(pathPrompt, &cadencePath, survey.WithValidator(func(val interface{}) error {
		str, _ := val.(string)
		clean := strings.TrimSpace(str)
		if _, err := os.Stat(clean); os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", clean)
		}
		return nil
	}))
	cadencePath = strings.TrimSpace(cadencePath)

	// Metadata JSON Path with instant validation
	var metaJsonPath string
	metaPathPrompt := &survey.Input{
		Message: "Where is your Metadata (.json) file?",
		Default: "./cadence/metadata/metadata.json",
	}
	survey.AskOne(metaPathPrompt, &metaJsonPath, survey.WithValidator(func(val interface{}) error {
		str, _ := val.(string)
		clean := strings.TrimSpace(str)
		if _, err := os.Stat(clean); os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", clean)
		}
		return nil
	}))
	metaJsonPath = strings.TrimSpace(metaJsonPath)

	// flow.json Path with instant validation
	var flowJsonPath string
	flowJsonPrompt := &survey.Input{
		Message: "Where is your flow.json configuration file?",
		Default: "./flow.json",
	}
	survey.AskOne(flowJsonPrompt, &flowJsonPath, survey.WithValidator(func(val interface{}) error {
		str, _ := val.(string)
		clean := strings.TrimSpace(str)
		if _, err := os.Stat(clean); os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", clean)
		}
		return nil
	}))
	flowJsonPath = strings.TrimSpace(flowJsonPath)

	// Resolve absolute system paths
	absCadencePath, err1 := filepath.Abs(cadencePath)
	absFlowJsonPath, err2 := filepath.Abs(flowJsonPath)
	absMetaJsonPath, err3 := filepath.Abs(metaJsonPath)

	if err1 != nil || err2 != nil || err3 != nil {
		fmt.Println("\n❌ Filepath Resolution Failure: Unable to calculate absolute system paths.")
		return "", nil, false
	}

	fmt.Println("\n⚙️  Running Flow CLI FLIX compiler...")

	cmd := exec.Command(
		"flow", "flix", "generate", absCadencePath,
		"-f", absFlowJsonPath,
		"--pre-fill", absMetaJsonPath,
		"--network", "testnet",
		"--network", "mainnet",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		rawStr := string(output)

		fmt.Println("\n" + strings.Repeat("━", 64))
		fmt.Println("❌ Flow CLI FLIX Generation Failed")
		fmt.Println(strings.Repeat("━", 64))

		// Extract the specific error line (bypassing version/security warnings)
		var specificError string
		lines := strings.Split(rawStr, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "❌ Command Error:") || strings.Contains(trimmed, "could not read") || strings.Contains(trimmed, "cannot find") {
				specificError = trimmed
				break
			}
		}

		if specificError != "" {
			fmt.Println("\n" + specificError)
			fmt.Println("⬆️ ⬆️ ⬆️ ⬆️ ⬆️  [ ROOT CAUSE OF FAILURE ]")
			fmt.Println("👉 Please check that the file path or contract definition is accessible.\n")
		} else {
			fmt.Printf("\nDetailed Diagnostics:\n%s\n", rawStr)
			fmt.Println("⬆️ ⬆️ ⬆️ ⬆️ ⬆️  [ FLOW CLI OUTPUT ]\n")
		}

		fmt.Println(strings.Repeat("━", 64))
		return "", nil, false
	}

	// Structural extraction of clean FLIX JSON
	firstBracketIdx := bytes.IndexByte(output, '{')
	lastBracketIdx := bytes.LastIndexByte(output, '}')

	if firstBracketIdx == -1 || lastBracketIdx == -1 || firstBracketIdx >= lastBracketIdx {
		fmt.Println("\n❌ Flow CLI executed, but the stream data contains no recognizable JSON object contents.")
		fmt.Printf("📋 Raw Output Stream Captured:\n%s\n", string(output))
		return "", nil, false
	}

	localBytes = bytes.TrimSpace(output[firstBracketIdx : lastBracketIdx+1])
	calculatedTemplatePath := strings.Replace(cadencePath, ".cdc", ".template.json", 1)
	err = os.WriteFile(calculatedTemplatePath, localBytes, 0644)
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not save template file to %s: %v\n", calculatedTemplatePath, err)
	} else {
		fmt.Printf("\n✅ Template successfully created at: %s\n", calculatedTemplatePath)
	}

	return calculatedTemplatePath, localBytes, true
}

func uploadTemplateToBackend(filePath string) {
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		fileData = []byte(`{"data": "mock_template_payload"}`)
	}

	url := "https://api.blsqui.net/api/flix/register"
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ Failed to read server execution stream: %v", err)
	}

	var bResp BlsquiUploadResponse
	if err := json.Unmarshal(body, &bResp); err != nil {
		fmt.Printf("❌ Failed to parse server metadata envelope: %v", err)
	}

	if !bResp.Success || bResp.TemplateID == "" {
		fmt.Printf("❌ Backend rejected payload: %s\n", bResp.Message)
		return
	}

	fmt.Println("\nSuccess!")
	fmt.Println("Your FLIX ID has been generated. Use the following URL to integrate this template into your application:")
	fmt.Printf("https://blsqui.net/flix/registry/%s\n\n", bResp.TemplateID)
}

func normalizeCadenceCode(code string) string {
	var cleanLines []string
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
	combinedCode := strings.Join(cleanLines, "")
	replacer := strings.NewReplacer(" ", "", "\t", "", "\n", "", "\r", "")
	return replacer.Replace(combinedCode)
}

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
	payload := map[string]interface{}{
		"targetFlixID":     flixID,
		"publicationState": publicationState,
		"publicTrigger":    promoteToPublic,
		"templateData":     json.RawMessage(localBytes),
	}

	bodyBytes, _ := json.Marshal(payload)
	resp, err := http.Post("https://api.blsqui.net/api/flix/update", "application/json", bytes.NewBuffer(bodyBytes))
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

	var uResp FlixUpdateResponse
	if err := json.NewDecoder(resp.Body).Decode(&uResp); err != nil {
		fmt.Printf("❌ Error decoding registry server response metadata: %v\n", err)
		return
	}

	fmt.Println("\n🎉 Congratulations! Updated FLIX Template is successfully injected into Blsqui Registry.")
	fmt.Println("Use the following URL to integrate this updated FLIX Template into your application:")
	fmt.Printf("👉 https://blsqui.net/flix/registry/%s\n\n", uResp.NewTemplateID)
}