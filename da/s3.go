package da

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ethereum/go-ethereum/common"
)

type S3Backend struct {
	s3Client     *s3.Client
	bucket       string
	objectPrefix string
}

func NewS3Backend(bucket, region, accessKey, secretKey, objectPrefix string) (*S3Backend, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		log.Printf("Failed to load AWS config, err:%v", err)
		return nil, err
	}
	s3Client := s3.NewFromConfig(cfg)

	return &S3Backend{
		s3Client:     s3Client,
		bucket:       bucket,
		objectPrefix: objectPrefix,
	}, nil
}

func (s *S3Backend) GetDataFromS3(hash common.Hash) ([]byte, error) {
	start := time.Now()
	log.Printf("Fetching data from S3, hash:%v", hash.Hex())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := s.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		log.Printf("Bucket check failed, bucket:%v, err:%v", s.bucket, err)
		return nil, fmt.Errorf("bucket check failed: %w", err)
	}

	out, err := s.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.objectPrefix + hash.String()),
	})
	if err != nil {
		log.Printf("Failed to get object from S3, key:%v, err:%v", s.objectPrefix+hash.String(), err)
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer out.Body.Close()

	data, err := io.ReadAll(out.Body)
	if err != nil {
		log.Printf("Failed to read object body, err:%v", err)
		return nil, fmt.Errorf("failed to read object body: %w", err)
	}

	log.Printf("Successfully retrieved data from S3, bucket:%s, key:%s, size:%d, duration:%v", s.bucket, s.objectPrefix+hash.String(),
		len(data),
		time.Since(start),
	)
	return data, nil
}
