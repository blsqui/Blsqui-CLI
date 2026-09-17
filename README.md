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

> 💡 **Tip:** To update an existing installation to the latest version, run the exact same command.

### macOS & Linux
```bash
sh -ci "$(curl -fsSL https://raw.githubusercontent.com/blsqui/blsqui-cli/master/install.sh)"
```

### Windows (PowerShell)
```bash
irm https://raw.githubusercontent.com/blsqui/blsqui-cli/master/install.ps1 | iex
```

## 📋 Prerequisites

### 1. Install Flow CLI
- macOS & Linux
```bash
sh -ci "$(curl -fsSL https://raw.githubusercontent.com/onflow/flow-cli/master/install.sh)"
```

- Windows (PowerShell)
```bash
iex (irm 'https://raw.githubusercontent.com/onflow/flow-cli/master/install.ps1')
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

# Blsqui FLIX Auto-Audit Specifications

To protect users against unintended token drains and asset losses, all FLIX templates submitted through `blsqui-cli` undergo automated static verification. Templates failing any of the following checks are rejected with HTTP `422 Unprocessable Entity`.

---

### Rules & Error Codes

| Error Code | Target | Rule Description |
| :--- | :--- | :--- |
| `FLIX_AUDIT_FT_AMOUNT_NOT_HARDCODED` | `cadence_script.cdc:<line>` | Any `.withdraw(amount: ...)` call protected by `auth(FungibleToken.Withdraw)` must use a **hardcoded numeric literal** (e.g. `1.0`), never a variable or parameter. |
| `FLIX_AUDIT_METADATA_MISMATCH` | `metadata.description` | 1. If an amount is withdrawn, both the exact numeric amount and the token symbol (e.g., `1.0 FLOW` or `1 FLOW`) must appear in all language descriptions.<br>2. If the Cadence code contains a `destroy` keyword, all description translations must include `*destroy`. |
| `FLIX_AUDIT_METADATA_MISMATCH` | `metadata.sdk:detail-body` | If `sdk:detail-body` is defined, it must include the numeric amount specified in the withdrawal. |

---

### Valid Sample

```cadence
// Cadence
let vaultRef = signer.storage.borrow<auth(FungibleToken.Withdraw) &FlowToken.Vault>(
    from: /storage/flowTokenVault
) ?? panic("Could not borrow reference to owner Vault")

self.sentVault <- vaultRef.withdraw(amount: 1.0)
```
```json
// metadata.json
{
  "messages": [
    {
      "key": "description",
      "i18n": [
        { "tag": "en-US", "translation": "Insert 1.0 FLOW coins to enter the match." },
        { "tag": "ja-JP", "translation": "対戦に参加するため 1.0 FLOW を支払います。" }
      ]
    }
  ]
}
```

## 📖 More Documentation & Guides

- Developer Guide & Tutorials: https://blsqui.net/developer-guide/blsqui-cli
- Unreal Engine 5 SDK: BlsquiSDK-UnrealPublic



## 📄 License
MIT License - see [LICENSE](LICENSE) for details.

