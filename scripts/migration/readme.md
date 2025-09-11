# CDK Data Availability Migration Tool

This tool extracts batch data from L1 contracts, fetches corresponding payloads from the DAC, and uploads them to **Avail Turbo DA** and an **S3 fallback bucket**.

---

## Features

- Iterates over a block range (`START_BLOCK` → `END_BLOCK`).
- Reads `sequenceBatchesValidium` transactions from L1.
- Fetches the referenced data from the DAC.
- Posts the data to Avail Turbo DA.
- Uploads the data to an S3 bucket as fallback.
- Structured logs with clear block-by-block separation.

---

## Requirements

- **Go** 1.23+
- Access to:
  - Ethereum L1 RPC (Sepolia/Mainnet)
  - DAC RPC service
  - Turbo DA API (with API key)
  - AWS S3 bucket for fallback storage

---

## Environment Variables

Create a `.env` file inside `scripts/migration/`:

```
⚠️ Note: Make sure END_BLOCK >= START_BLOCK, otherwise initialization will fail.
```

```env
# Block range
START_BLOCK=9165751
END_BLOCK=9165800

# L1
L1_RPC_URL=https://sepolia.infura.io/v3/<PROJECT_ID>
L1_CONTRACT_ADDRESS=0xYourContractAddressHere

# DAC
DAC_RPC_URL=https://dac.test.availproject.org

# Turbo DA
TURBO_DA_URL=https://staging.turbo-api.availproject.org/v1/submit_raw_data
TURBO_DA_API_KEY=your-api-key-here

# S3
S3_BUCKET=polygon-cdk-test
S3_REGION=us-east-1
S3_ACCESS_KEY=your-access-key
S3_SECRET_KEY=your-secret-key
S3_OBJECT_PREFIX=

# Retry tuning
MAX_ATTEMPTS=5

```

## Running

```shell
cd scripts/migration
go run main.go
```

## Logs

```
══════════════════════════════════════
🟦 Processing Block 9165751
══════════════════════════════════════
Tx: 0xf08809b2...
Method: sequenceBatchesValidium
🔍 Found 1 batch hashes
  ➡️ Batch 0 [Hash: 0xb37c4fdd...]
     ✅ DAC fetch success (size=544 bytes)
     ✅ DA upload success
```
