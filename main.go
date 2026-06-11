package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	action := Action{
		Operation: os.Getenv("ACTION"),
		Bucket:    os.Getenv("BUCKET"),
		S3Class:   os.Getenv("S3_CLASS"),
		Key:       os.Getenv("KEY"),
		Artifacts: os.Getenv("ARTIFACTS"),
	}

	region := os.Getenv("AWS_REGION")

	if action.S3Class == "" {
		action.S3Class = "STANDARD"
	}

	switch action.Operation {
	case ActionPut:
		runPut(action, region)
	case ActionGet:
		runGet(action, region)
	case ActionDelete:
		runDelete(action, region)
	default:
		log.Fatalf("Invalid action: %s", action.Operation)
	}
}

func runPut(a Action, region string) {
	if a.Artifacts == "" {
		log.Fatal("artifacts input is required for put operation")
	}

	zipFile := a.Key + ".zip"
	defer os.Remove(zipFile)

	if err := Zip(a.Artifacts, zipFile); err != nil {
		log.Fatalf("Failed to create zip: %v", err)
	}

	if err := PutObject(region, a.Bucket, zipFile, a.S3Class); err != nil {
		log.Fatalf("Failed to upload to S3: %v", err)
	}

	log.Print("Cache saved successfully")
}

func runGet(a Action, region string) {
	zipFile := a.Key + ".zip"

	exists, err := ObjectExists(region, a.Bucket, zipFile)
	if err != nil {
		log.Fatalf("Failed to check cache existence: %v", err)
	}

	if !exists {
		log.Printf("No caches found for the following key: %s", zipFile)
		return
	}

	log.Printf("Cache hit for the following key: %s", zipFile)

	size, err := GetObject(region, a.Bucket, zipFile)
	if err != nil {
		log.Fatalf("Failed to download from S3: %v", err)
	}

	if err := Unzip(zipFile); err != nil {
		log.Fatalf("Failed to extract zip: %v", err)
	}

	os.Remove(zipFile)

	log.Printf("Cache downloaded successfully, containing %d bytes", size)

	if err := setCacheHit(); err != nil {
		log.Fatalf("Failed to set cache-hit output: %v", err)
	}
}

func runDelete(a Action, region string) {
	zipFile := a.Key + ".zip"

	if err := DeleteObject(region, a.Bucket, zipFile); err != nil {
		log.Fatalf("Failed to delete from S3: %v", err)
	}

	log.Print("Cache purged successfully")
}

func setCacheHit() error {
	githubOutput := os.Getenv("GITHUB_OUTPUT")
	if githubOutput == "" {
		return nil
	}

	f, err := os.OpenFile(githubOutput, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open GITHUB_OUTPUT: %w", err)
	}
	defer f.Close()

	_, err = fmt.Fprintln(f, "CACHE_HIT=true")
	return err
}
