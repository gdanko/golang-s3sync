package s3sync

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gdanko/golang-s3sync/pkg/s3diff"

	"fmt"
)

type Syncer struct {
	ACL          string
	Debug        bool
	Delete       bool
	Destination  string
	Differ       *s3diff.Differ
	Dryrun       bool
	MaxThreads   int
	Profile      string
	Region       string
	Source       string
	SourceBucket string
	S3           *s3.S3
	Verify       bool
}

var (
	err   error
	input *s3.ListBucketsInput
	sess  *session.Session
)

func (s *Syncer) Sync() error {
	err = s.validate()
	if err != nil {
		return err
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(s.Region),
		// Credentials: credentials.NewSharedCredentials("", s.Profile),
	})

	if err != nil {
		return err
	}

	s.S3 = s3.New(sess)
	input = &s3.ListBucketsInput{}

	// Make sure we can connect with the provided credentials
	_, err = s.S3.ListBuckets(input)
	if err != nil {
		// https://docs.aws.amazon.com/sdk-for-go/api/service/s3/#S3.ListBuckets
		return fmt.Errorf("aws was not able to validate the provided access credentials")
	}

	s.init()
	s.Differ.Diff()
	s.Differ.GenerateSyncList()

	return nil
}

func (s *Syncer) init() error {
	s.Differ = &s3diff.Differ{
		Source:      s.Source,
		Destination: s.Destination,
		S3:          s.S3,
		Delete:      s.Delete,
		Debug:       s.Debug,
	}

	err = s.Differ.DetermineTypes()
	if err != nil {
		return err
	}

	if s.Differ.SourceType == "s3" {
		s.SourceBucket = s.Differ.SourceBucket
	}

	return nil
}
