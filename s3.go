package main

import (
	"bytes"
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

func getS3Client() (s3.Client, error) {
	sdkConfig, err := config.LoadDefaultConfig(context.TODO())

	s3Client := *s3.NewFromConfig(sdkConfig)

	return s3Client, err
}

// BucketBasics encapsulates the Amazon Simple Storage Service (Amazon S3) actions used in the examples. It contains S3Client, an Amazon S3 service client that is used to perform bucket and object actions.
type BucketBasics struct {
	S3Client *s3.Client
}

// UploadFile reads from a file and puts the data into an object in a bucket.
func (basics BucketBasics) UploadFile(image *bytes.Reader, fileExt string, contentType string) (string, error) {

	fileName := uuid.NewString() + "." + fileExt
	log.Println("fileName:", fileName)

	_, err := basics.S3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String("assets/" + fileName),
		Body:        image,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		log.Printf("Couldn't upload file %v to %v. Here's why: %v\n",
			fileName, bucketName, err)
	}

	return fileName, err
}

// DeleteObject deletes a single object from a bucket and returns the deleted object key.
func (basics BucketBasics) DeleteObject(objectKey string) (string, error) {
	_, err := basics.S3Client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String("assets/" + objectKey),
	})

	if err != nil {
		log.Printf("Couldn't delete object %v from bucket %v. Here's why: %v\n", objectKey, bucketName, err)
		return "", err
	}

	log.Printf("DeleteObject finished with %v.\n", objectKey)

	return objectKey, err
}
