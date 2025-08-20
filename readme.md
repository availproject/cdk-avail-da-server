# CDK Erigon DA Server

A simple cdk-avail-da-server for serving off-chain data.  
It attempts to fetch data from an Avail DA and can falls back to an AWS S3 bucket if the data was unable to found from Avail DA

---

## Features

- JSON-RPC endpoint: `sync_getOffChainData`
- Retrieves data from:
  - Avail DA (on-chain)
  - AWS S3 bucket (off-chain fallback)
- Health check endpoint (`/health`)
- Configurable via `.env` file
- Built with Go's standard logger for simplicity

---

## Requirements

- **Go** 1.23+
- **Docker** & **Docker Compose** (optional, for containerized runs)
- AWS S3 credentials with read permissions
- A running L1 RPC endpoint for contract calls

---

## Environment Variables

Create a `.env` file in the project root:

```env
# L1 configuration
L1_RPC_URL=
ATTESTATION_CONTRACT_ADDRESS=
IS_BRIDGE_ENABLED=

# Avail configuration
AVAIL_RPC_URL=

# S3 configuration
S3_BUCKET=
S3_REGION=
S3_ACCESS_KEY=
S3_SECRET_KEY=
S3_OBJECT_PREFIX=
```

## Running the server

### Locally

```shell
go mod tidy
go run server.go
```

### Docker

```shell
docker compose up
```

The server will starts on <http://localhost:8000>

## API

JSON-RPC: Get Off-Chain Data

### Request body

```json
{
  "jsonrpc": "2.0",
  "method": "sync_getOffChainData",
  "params": ["0xHASH_HERE"],
  "id": 1
}
```

Example curl:

```shell
curl -X POST http://localhost:8080/rpc \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"sync_getOffChainData","params":["0xHASH_HERE"],"id":1}'
```

### Response

Success

```json
{
  "jsonrpc": "2.0",
  "result": "base64_encoded_data_here",
  "id": 1
}
```

Failed

```json
{
  "jsonrpc": "2.0",
  "error": "failed to retrieve off-chain data",
  "id": 1
}
```
