package service

import (
	"errors"
	"log"
	"os"
	"strconv"

	"github.com/availproject/cdk-erigon-da-service/da"
	"github.com/ethereum/go-ethereum/common"
)

func GetOffChainData(a *da.AvailBackend, s *da.S3Backend, hash string) ([]byte, error) {
	log.Printf("Processing hash: %s", hash)

	hexHash := common.HexToHash(hash)

	isBridgeEnabled, err := strconv.ParseBool(os.Getenv("IS_BRIDGE_ENABLED"))
	if err != nil {
		log.Printf("Invalid boolean value for IS_BRIDGE_ENABLED: %v", err)
		return nil, err
	}

	if isBridgeEnabled {
		data, err := a.GetDataFromAvail(hexHash)
		if err != nil {
			log.Printf("Failed to get data from Avail, falling back to S3: %v", err)
		} else {
			log.Println("Successfully retrieved data from Avail")
			return data, nil
		}
	}

	log.Println("Retrieving off-chain data from S3")
	data, err := s.GetDataFromS3(hexHash)
	if err != nil {
		log.Printf("Failed to retrieve off-chain data from S3: %v", err)
		return nil, errors.New("failed to retrieve the data from off-chain DA")
	}

	log.Println("Successfully retrieved off-chain data")
	return data, nil
}
