package main

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func newSession(region string) *session.Session {
	return session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))
}

func PutObject(region, bucket, key, s3Class string) error {
	sess := newSession(region)
	uploader := s3manager.NewUploader(sess)

	f, err := os.Open(key)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket:       aws.String(bucket),
		Key:          aws.String(key),
		Body:         f,
		StorageClass: aws.String(s3Class),
	})
	return err
}

func GetObject(region, bucket, key string) (int64, error) {
	sess := newSession(region)
	downloader := s3manager.NewDownloader(sess)

	f, err := os.Create(key)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	n, err := downloader.Download(f, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return n, err
}

func DeleteObject(region, bucket, key string) error {
	sess := newSession(region)
	svc := s3.New(sess)

	_, err := svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return err
}

func ObjectExists(region, bucket, key string) (bool, error) {
	sess := newSession(region)
	svc := s3.New(sess)

	_, err := svc.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFound" {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
