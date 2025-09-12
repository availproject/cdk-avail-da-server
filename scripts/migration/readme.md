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
#L1 RPC
RPC_URL=https://sepolia.infura.io/v3/<API_KEY>

# DAC RPC
DAC_URL=https://test.cdk.dac/rpc/

# Turbo DA
TURBO_DA_URL=https://turing.turbo-api.availproject.org
API_KEY=

# Rollup contract address
CONTRACT_ADDRESS=0x123456789abcdef...

# Migration l1 block range
START_BLOCK=5000000
END_BLOCK=5000100

# Script retry attempts count
MAX_ATTEMPTS=5
MAX_TIMEOUT_MINS=10

# S3 configuration
S3_BUCKET=
S3_REGION=
S3_ACCESS_KEY=
S3_SECRET_KEY=
S3_OBJECT_PREFIX=
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
