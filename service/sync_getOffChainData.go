package service

import (
	"errors"
	"log"

	"github.com/availproject/cdk-avail-da-server/da"
	"github.com/ethereum/go-ethereum/common"
)

func GetOffChainData(a *da.AvailBackend, s *da.S3Backend, hash string) ([]byte, error) {
	log.Printf("Getting off-chain data for hash: %s", hash)

	hexHash := common.HexToHash(hash)

	// Disabled support for L1 recovery thru Avail chain
	// if a.IsBridgeEnabled() {
	// 	data, err := a.GetDataFromAvail(hexHash)
	// 	if err != nil {
	// 		log.Printf("Failed to get data from Avail, falling back to S3: %v", err)
	// 	} else {
	// 		log.Println("Successfully retrieved data from Avail")
	// 		return data, nil
	// 	}
	// } else {
	// 	log.Printf("Avail Bridge is not enabled for the cdk chain, checking on S3")
	// }

	log.Println("Retrieving off-chain data from S3")
	data, err := s.GetDataFromS3(hexHash)
	if err != nil {
		log.Printf("Failed to retrieve off-chain data from S3: %v", err)
		return nil, errors.New("failed to retrieve the data from off-chain DA")
	}

	log.Println("Successfully retrieved off-chain data")
	return data, nil
}
