package da

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ethereum/go-ethereum/common"
)

type DABackend struct {
	s3Client     *s3.Client
	bucket       string
	objectPrefix string
	turboDAURL   string
	apiKey       string
}

func NewDABackend(bucket, region, accessKey, secretKey, objectPrefix, turboDAURL, apiKey string) (*DABackend, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		log.Printf("Failed to load AWS config for bucket %s in region %s, err: %v", bucket, region, err)
		return nil, err
	}
	s3Client := s3.NewFromConfig(cfg)

	return &DABackend{
		s3Client:     s3Client,
		turboDAURL:   turboDAURL,
		apiKey:       apiKey,
		bucket:       bucket,
		objectPrefix: objectPrefix,
	}, nil
}

func encodeKey(hash common.Hash) string {
	return hash.Hex()[2:] // strip 0x
}

func (s *DABackend) PostDataToDA(ctx context.Context, hash common.Hash, data []byte) error {
	// First post to Turbo DA
	_, err := PostDataToTurboDA(ctx, s.turboDAURL, s.apiKey, data)
	if err != nil {
		log.Printf("Failed to post data to Turbo DA for hash %s: %v", hash.Hex(), err)
		return err
	}
	// Then upload to S3
	err = PostDataToS3(ctx, s.s3Client, s.objectPrefix, s.bucket, hash, data)
	if err != nil {
		log.Printf("Failed to upload data to S3 for hash %s: %v", hash.Hex(), err)
		return err
	}
	return nil
}

func PostDataToS3(ctx context.Context, s3Client *s3.Client, objectPrefix string, bucket string, hash common.Hash, data []byte) error {
	start := time.Now()
	key := objectPrefix + encodeKey(hash)
	log.Printf("Uploading data to S3, bucket:%s, key:%s, hash:%s, size:%d bytes", bucket, key, hash.Hex(), len(data))

	// PutObject API call
	_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		log.Printf("Failed to upload object to S3, bucket:%s, key:%s, hash:%s, err:%v", bucket, key, hash.Hex(), err)
		return fmt.Errorf("failed to upload object to S3: %w", err)
	}

	log.Printf("Successfully uploaded data to S3, bucket:%s, key:%s, hash:%s, size:%d bytes, duration:%v",
		bucket, key, hash.Hex(), len(data), time.Since(start),
	)
	return nil
}

func PostDataToTurboDA(ctx context.Context, url string, apiKey string, data []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url+"/v1/submit_raw_data", bytes.NewReader(data))
	if err != nil {
		log.Printf("Failed to create request to Turbo DA: %v", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("x-api-key", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Failed to post data to Turbo DA: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response from Turbo DA: %v", err)
		return nil, err
	}

	log.Printf("Successfully posted data to Turbo DA")
	return respData, nil
}
