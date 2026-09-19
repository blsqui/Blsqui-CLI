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

type FlixServerResponse struct {
    Success       bool   `json:"success"`
    Verified      bool   `json:"verified"`
    TemplateID    string `json:"template_id"`
    NewTemplateID string `json:"new_template_id"`
}

type FlixUpdateEnvelope struct {
    TargetFlixID     string          `json:"targetFlixID"`
    PublicationState string          `json:"publicationState"`
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

type BlsquiAuditError struct {
    Code    string `json:"code"`
    Target  string `json:"target"`
    Message string `json:"message"`
}

type BlsquiErrorResponse struct {
    Success bool               `json:"success"`
    Error   string             `json:"error"`
    Errors  []BlsquiAuditError `json:"errors"`
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
        Default: "./cadence/metadata/transfer_10_flow_metadata.json",
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
        fmt.Printf("❌ Failed to read server execution stream: %v\n", err)
        return
    }

    // Intercept HTTP errors (e.g., 422 Unprocessable Entity for audit failures, 400 Bad Request)
    if resp.StatusCode != http.StatusOK {
        handleAuditErrorResponse(resp.StatusCode, body)
        return
    }

    var bResp FlixServerResponse
    if err := json.Unmarshal(body, &bResp); err != nil {
        fmt.Printf("❌ Failed to parse server metadata envelope: %v\n", err)
        return
    }

	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    fmt.Println("🎉 FLIX Template Registered Successfully!")
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

    fmt.Printf("\n📋 FLIX Template ID (For SDK & Web):\n")
    fmt.Printf("   \033[1;36m%s\033[0m\n", bResp.TemplateID)

    fmt.Printf("\n🔗 Direct Registry Endpoint:\n")
    fmt.Printf("   https://api.blsqui.net/flix/registry/%s\n", bResp.TemplateID)

    fmt.Println("\n💡 Tip:")
    fmt.Println("   Copy the FLIX Template ID above into your Unreal Engine / Unity / Godot SDKs")
    fmt.Println("   or pass it directly into your web integration.")
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
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
    url := fmt.Sprintf("https://api.blsqui.net/flix/registry/%s", flixID)
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

func executeFlixUpdatePayload(flixID string, publicationState string, localBytes []byte) {
    payload := map[string]interface{}{
        "targetFlixID":     flixID,
        "publicationState": publicationState,
        "templateData":     json.RawMessage(localBytes),
    }

    bodyBytes, _ := json.Marshal(payload)
    resp, err := http.Post("https://api.blsqui.net/api/flix/update", "application/json", bytes.NewBuffer(bodyBytes))
    if err != nil {
        fmt.Printf("❌ Network Error: Could not reach Blsqui Registry: %v\n", err)
        return
    }
    defer resp.Body.Close()

    respBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        fmt.Printf("❌ Failed to read server execution stream: %v\n", err)
        return
    }

    if resp.StatusCode != http.StatusOK {
        handleAuditErrorResponse(resp.StatusCode, respBytes)
        return
    }

    var uResp FlixServerResponse
    if err := json.Unmarshal(respBytes, &uResp); err != nil {
        fmt.Printf("❌ Error decoding registry server response metadata: %v\n", err)
        return
    }

    fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    fmt.Println("🎉 FLIX Template Updated & Synchronized Successfully!")
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

    fmt.Printf("\n📋 FLIX Template ID (For SDK & Web):\n")
    fmt.Printf("   \033[1;36m%s\033[0m\n", uResp.TemplateID)

    fmt.Printf("\n🔗 Direct Registry Endpoint:\n")
    fmt.Printf("   https://api.blsqui.net/flix/registry/%s\n", uResp.TemplateID)

    if publicationState == "PUBLISH_LATER" {
        fmt.Println("\n🔒 Status: Staged Privately (on_public: false)")
        fmt.Println("   The template passed automated audit checks and is saved.")
        fmt.Println("   Promote it to the public registry when you are ready to release.")
    } else {
        fmt.Println("\n✅ Status: Live & Active (on_public: true)")
        fmt.Println("   Automated audit verified. Updates are live immediately.")
    }
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}

func handleAuditErrorResponse(statusCode int, bodyBytes []byte) bool {
    if statusCode == http.StatusOK {
        return false
    }

    var errResp BlsquiErrorResponse
    if err := json.Unmarshal(bodyBytes, &errResp); err == nil && len(errResp.Errors) > 0 {
        fmt.Println("\n❌ FLIX Audit Violations Detected (HTTP Status 422):")
        fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        for _, e := range errResp.Errors {
            fmt.Printf(" • [%s] at \033[1;33m%s\033[0m:\n   %s\n", e.Code, e.Target, e.Message)
        }
        fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Println("\n💡 How to Resolve Audit Violations:")
        fmt.Println(" 1. [FLIX_AUDIT_MISSING_REQUIRED_FIELD]")
        fmt.Println("    → Ensure 'title', 'icon', and 'description' exist under messages with non-empty translations.")
        fmt.Println(" 2. [FLIX_AUDIT_FT_AMOUNT_NOT_HARDCODED]")
        fmt.Println("    → Hardcode the numeric amount inside `.withdraw(amount: ...)` (e.g. `1.0`), dynamic variables are disallowed.")
        fmt.Println(" 3. [FLIX_AUDIT_METADATA_MISMATCH (description)]")
        fmt.Println("    → Include both numeric amount and token ticker (e.g. '1.0 FLOW') in description translations.")
        fmt.Println("    → If Cadence uses `destroy`, include '*destroy' in description translations.")
        fmt.Println(" 4. [FLIX_AUDIT_METADATA_MISMATCH (sdk:detail-body)]")
        fmt.Println("    → If 'sdk:detail-body' is provided, ensure the withdrawal amount (e.g. '1' or '1.0') is stated using standard half-width digits.")
        fmt.Println()
        return true
    }

    fmt.Printf("❌ Upload Rejected by Registry (Status: %d):\n%s\n", statusCode, string(bodyBytes))
    return true
}