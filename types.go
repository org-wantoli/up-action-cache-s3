package main

const (
	ActionPut    = "put"
	ActionGet    = "get"
	ActionDelete = "delete"
)

type Action struct {
	Operation string
	Bucket    string
	S3Class   string
	Key       string
	Artifacts string
}
