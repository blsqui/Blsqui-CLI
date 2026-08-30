# Blsqui CLI

The official developer command-line tool for **Blsqui** — streamlining Cadence smart contract deployments, FLIX (Flow Interaction Template) v1.1 generation, and registry synchronization for web and game engine integrations (Unreal Engine 5, Unity, Web).

---

## ⚡ Key Features

- **Automated FLIX v1.1 Generation:** Compiles Cadence transactions (`.cdc`) and metadata into standardized FLIX templates.
- **Domain & IP Authorization:** Enforces domain restrictions and challenge-response security rules for game transactions.
- **SDK Nonce Preflight Checks:** Verifies `nonce: String` parameters for replay protection and game engine transaction synchronization.
- **Blsqui Registry Sync:** Direct template upload, metadata update, and audit status verification via `api.blsqui.net`.
- **Testnet Smart Contract Deployment:** Interactive deployment and migration workflows for Cadence contracts.

---

## 📦 Installation

### macOS & Linux
```bash
sh -ci "$(curl -fsSL [https://raw.githubusercontent.com/blsqui/blsqui-cli/master/install.sh](https://raw.githubusercontent.com/blsqui/blsqui-cli/master/install.sh))"
```

### Windows (PowerShell)
```bash
irm [https://raw.githubusercontent.com/blsqui/blsqui-cli/master/install.ps1](https://raw.githubusercontent.com/blsqui/blsqui-cli/master/install.ps1) | iex
```

## 📋 Prerequisites

### 1. Install Flow CLI
- macOS & Linux
```bash
sh -ci "$(curl -fsSL [https://raw.githubusercontent.com/onflow/flow-cli/master/install.sh](https://raw.githubusercontent.com/onflow/flow-cli/master/install.sh))"
```

- Windows (PowerShell)
```bash
iex (irm '[https://raw.githubusercontent.com/onflow/flow-cli/master/install.ps1](https://raw.githubusercontent.com/onflow/flow-cli/master/install.ps1)')
```

### 2. Initialize Flow Project

In your project root directory, run:
```bash
flow init
```

1. Enter your project name when prompted.
2. Select Custom project (select standard Flow contract dependencies).
3. Select FlowToken and FungibleToken for standard contract dependencies.

## Usage

Launch the interactive developer interface:
```bash
blsqui
```

### Interactive Menu Workflows

| Action | Description |
| --- | --- |
| `🚀 Deploy Contract` | Auto-deploy and migrate Cadence smart contracts to Testnet |
| `📦 Upload FLIX` | Generate and register a new FLIX v1.1 template to the Blsqui registry |
| `🔄 Update Template` | Sync IP branding, localized metadata, or updated Cadence code |
| `🔍 Verify Status` | Inspect audit progress and publish templates to the public registry |
| `🚪 Exit` | Close the developer tool |

## Command-Line Flags
```bash
# Display CLI version
blsqui -v
blsqui -version

# Display help output
blsqui --help
```

## 📖 Documentation & Guides

- Developer Guide & Tutorials: https://blsqui.net/developer-guide/blsqui-cli
- Unreal Engine 5 SDK: BlsquiSDK-UnrealPublic



## 📄 License
MIT License - see [LICENSE](LICENSE) for details.

