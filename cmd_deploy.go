package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
)

type FlowConfig struct {
	Accounts map[string]interface{} `json:"accounts"`
}

func DeployTestnetContract() {
	network := "testnet"

	var contractQuestions = []*survey.Question{
		{
			Name: "name",
			Prompt: &survey.Input{
				Message: "Enter the Smart Contract Name:",
				Default: "GeneralGame",
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

	combinedLogs := string(outputBytes)
	fmt.Print(combinedLogs)

	if err == nil {
		fmt.Println("Success! Contract successfully updated.")
		return
	}
	fmt.Println("████████████████████████████████████████████████████████████")
	fmt.Println("\n------------------------------------------------------------")

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
			subPart := strings.Split(parts[1], ",")[0]
			mintedAddress = strings.Trim(subPart, " \"\n\r\t")
		}
	}

	if mintedAddress == "" {
		fmt.Println("❌ Failed to parse the newly minted address from Flow CLI response data.")
		return
	}

	mintedAddress = strings.TrimPrefix(mintedAddress, "0x")
	fmt.Printf("New Testnet address was successfully generated: [0x%s]\n", mintedAddress)

	fmt.Println("Saving the new profile configuration into flow.json...")
	addAccountCmd := exec.Command("flow", "config", "add", "account",
		"-f", flowJsonFilename,
		"--name", newAccountName,
		"--address", mintedAddress,
		"--private-key", "58552795fa36262c9a1d4a1a47e63eb947dfd385e2ee6eca621625987572df62",
	)
	addAccountCmd.Dir = workingDir
	if err := addAccountCmd.Run(); err != nil {
		fmt.Printf("❌ Failed to modify user accounts array: %v\n", err)
		return
	}

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

	tempTxFile := ".temp_transfer_tokens.cdc"
	tempTxPath := filepath.Join(workingDir, tempTxFile)
	if err := os.WriteFile(tempTxPath, []byte(strings.TrimSpace(transferCadenceCode)), 0644); err != nil {
		fmt.Printf("❌ Failed to create temporary pipeline transaction: %v\n", err)
		return
	}
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