package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"

	"github.com/availproject/cdk-avail-da-server/scripts/migration/pkg/da"
	"github.com/availproject/cdk-avail-da-server/scripts/migration/pkg/dac"
	"github.com/availproject/cdk-avail-da-server/scripts/migration/pkg/l1"
)

type MigrationService struct {
	ctx          context.Context
	cancel       context.CancelFunc
	client       *ethclient.Client
	DABackend    *da.DABackend
	startBlock   *big.Int
	endBlock     *big.Int
	contractAbi  abi.ABI
	contractAddr common.Address
	dacURL       string
	maxAttempts  int
}

func main() {

	m, err := initialize()
	if err != nil {
		log.Fatalf("Failed to initialize migration service: %v", err)
	}
	defer m.cancel()

	// Iterate over blocks, query batch hashes, fetch from DAC, and upload to S3
	for block := new(big.Int).Set(m.startBlock); block.Cmp(m.endBlock) <= 0; block.Add(block, big.NewInt(1)) {
		log.Printf("\n═══════════════════════════════════════════")
		log.Printf("🟦 Processing Block %d", block.Uint64())
		log.Printf("═══════════════════════════════════════════")
		hashes, err := l1.QueryBatchHashesFromL1ByBlockNumber(m.ctx, m.client, m.contractAbi, m.contractAddr, block)
		if err != nil {
			log.Printf("Error querying batch hashes from L1 for block %d: %v", block.Uint64(), err)
			continue
		}
		if len(hashes) == 0 {
			log.Printf("ℹ️  No batch hashes found")
			continue
		}

		log.Printf("🔍 Found %d batch hashes", len(hashes))
		for i, h := range hashes {
			log.Printf("  ➡️ Batch %d [Hash: %s]", i, h.Hex())
			var batchData []byte
			var err error
			// Fetch from DAC with retries
			err = retry(m.ctx, m.maxAttempts, 1*time.Second, func() error {
				var e error
				batchData, e = dac.GetDataFromDACByHash(m.ctx, m.dacURL, h)
				if e != nil {
					log.Printf("    ❌ DAC fetch failed: %v", e)
					return e
				}
				log.Printf("    ✅ DAC fetch success (size=%d bytes)", len(batchData))
				return nil
			})
			if err != nil {
				log.Printf("    ⛔ Skipping batch (could not fetch from DAC)")
				continue
			}
			// Upload to S3 with retries
			err = retry(m.ctx, m.maxAttempts, 1*time.Second, func() error {
				e := m.DABackend.PostDataToDA(m.ctx, h, batchData)
				if e != nil {
					log.Printf("    ❌ DA upload failed: %v", e)
					return e
				}
				log.Printf("    ✅ DA upload success")
				return nil
			})
			if err != nil {
				log.Printf("Failed to upload batch hash %s after retries: %v", h.Hex(), err)
			}
		}

	}
}

func initialize() (MigrationService, error) {
	// Load .env file
	if err := godotenv.Load(".env"); err != nil {
		log.Println("No .env file found, falling back to system env")
	}

	// Read and validate environment variables
	rpcURL := os.Getenv("RPC_URL")
	dacURL := os.Getenv("DAC_URL")
	contractAddr := common.HexToAddress(os.Getenv("CONTRACT_ADDRESS"))
	startBlock := new(big.Int)
	startBlock.SetString(os.Getenv("START_BLOCK"), 10)
	endBlock := new(big.Int)
	endBlock.SetString(os.Getenv("END_BLOCK"), 10)

	if rpcURL == "" || contractAddr == (common.Address{}) || startBlock.Cmp(big.NewInt(0)) == 0 || endBlock.Cmp(big.NewInt(0)) == 0 {
		log.Println(rpcURL, contractAddr, startBlock, endBlock)
		return MigrationService{}, fmt.Errorf("please set RPC_URL, CONTRACT_ADDRESS, START_BLOCK, and END_BLOCK environment variables")
	}

	if startBlock.Cmp(endBlock) > 0 {
		log.Println(endBlock, startBlock)
		return MigrationService{}, fmt.Errorf("END_BLOCK must be greater than to START_BLOCK")
	}

	bucket := os.Getenv("S3_BUCKET")
	region := os.Getenv("S3_REGION")
	accessKey := os.Getenv("S3_ACCESS_KEY")
	secretKey := os.Getenv("S3_SECRET_KEY")
	objectPrefix := os.Getenv("S3_OBJECT_PREFIX")

	if bucket == "" || region == "" || accessKey == "" || secretKey == "" {
		return MigrationService{}, fmt.Errorf("missing required S3 configuration")
	}

	turboDAURL := os.Getenv("TURBO_DA_URL")
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" && turboDAURL == "" {
		return MigrationService{}, fmt.Errorf("please set API_KEY and TURBO_DA_URL environment variables")
	}

	// Read MAX_ATTEMPTS env variable and parse to int
	maxAttempts := 5
	if maxAttemptsStr := os.Getenv("MAX_ATTEMPTS"); maxAttemptsStr != "" {
		if val, err := strconv.Atoi(maxAttemptsStr); err == nil && val > 0 {
			maxAttempts = val
		}
	}
	maxTimeOutMins := 10
	if maxTimeOutMinsStr := os.Getenv("MAX_TIMEOUT_MINS"); maxTimeOutMinsStr != "" {
		if val, err := strconv.Atoi(maxTimeOutMinsStr); err == nil && val > 0 {
			maxTimeOutMins = val
		}
	}

	// Initialization
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(maxTimeOutMins)*time.Minute)

	da, err := da.NewDABackend(bucket, region, accessKey, secretKey, objectPrefix, turboDAURL, apiKey)
	if err != nil {
		cancel()
		return MigrationService{}, fmt.Errorf("failed to initialize DA backend: %w", err)
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		cancel()
		return MigrationService{}, err
	}

	// Load ABI
	contractAbi, err := abi.JSON(strings.NewReader(l1.PolygonValidiumEtrogABI))
	if err != nil {
		cancel()
		return MigrationService{}, err
	}

	return MigrationService{
		ctx:          ctx,
		cancel:       cancel,
		client:       client,
		DABackend:    da,
		startBlock:   startBlock,
		endBlock:     endBlock,
		contractAbi:  contractAbi,
		contractAddr: contractAddr,
		dacURL:       dacURL,
		maxAttempts:  maxAttempts,
	}, nil
}

// retry executes fn with exponential backoff and jitter
func retry(ctx context.Context, maxAttempts int, baseDelay time.Duration, fn func() error) error {
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}

		// Compute exponential backoff: baseDelay * 2^(attempt-1)
		backoff := baseDelay << (attempt - 1)

		// Add jitter: up to ±20%
		jitter := time.Duration(rand.Int63n(int64(backoff) / 5))
		if rand.Intn(2) == 0 {
			backoff -= jitter
		} else {
			backoff += jitter
		}

		// Don’t wait if this was the last attempt
		if attempt == maxAttempts {
			break
		}

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return err
}
