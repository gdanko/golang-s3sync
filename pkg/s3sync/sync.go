package s3sync

import (
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gdanko/golang-s3sync/pkg/s3diff"

	"fmt"
)

// Syncer holds information about how to sync
type Syncer struct {
	ACL         string
	Debug       bool
	Delete      bool
	Destination string
	Differ      *s3diff.Differ
	Downloader  *s3manager.Downloader
	Dryrun      bool
	MaxThreads  int
	Profile     string
	Region      string
	Source      string
	S3          *s3.S3
	Uploader    *s3manager.Uploader
	Verify      bool
}

// SyncOuput will hold the output information for each synced item
type SyncOutput struct {
	Message string
	Status  string
}

var (
	err   error
	input *s3.ListBucketsInput
	sess  *session.Session
)

// Sync initializes the Differ, triggers the diff, and performs the sync
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
	s.Uploader = s3manager.NewUploaderWithClient(s.S3)
	s.Downloader = s3manager.NewDownloaderWithClient(s.S3)

	input = &s3.ListBucketsInput{}

	// Make sure we can connect with the provided credentials
	_, err = s.S3.ListBuckets(input)
	if err != nil {
		// https://docs.aws.amazon.com/sdk-for-go/api/service/s3/#S3.ListBuckets
		return fmt.Errorf("aws was not able to validate the provided access credentials")
	}

	s.init()
	// prettyPrint(s.Differ.SyncList, true)
	s.syncFiles()

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

	s.Differ.Diff()
	s.Differ.GenerateSyncList()

	return nil
}

func (s *Syncer) syncFiles() error {
	var (
		// name string
		job       s3diff.SyncItem
		jobs      chan s3diff.SyncItem
		obj       s3diff.SyncItem
		results   chan string
		totalJobs int
	)

	if len(s.Differ.SyncList) > 0 {
		actions := map[string][]s3diff.SyncItem{
			"copy":     []s3diff.SyncItem{},
			"download": []s3diff.SyncItem{},
			"upload":   []s3diff.SyncItem{},
			"delete":   []s3diff.SyncItem{},
		}

		for _, obj = range s.Differ.SyncList {
			action := obj.Action
			actions[action] = append(actions[action], obj)
		}

		if len(actions["copy"]) > 0 {
			totalJobs = len(actions["copy"])
			jobs = make(chan s3diff.SyncItem, totalJobs)
			results = make(chan string, totalJobs)

			for w := 1; w <= s.MaxThreads; w++ {
				go s.s3ToS3(w, jobs, results)
			}

			for _, job = range actions["copy"] {
				jobs <- job
			}
			close(jobs)

			for _, _ = range actions["copy"] {
				<-results
			}
		}

		if len(actions["download"]) > 0 {
			totalJobs = len(actions["download"])
			jobs = make(chan s3diff.SyncItem, totalJobs)
			results = make(chan string, totalJobs)

			for w := 1; w <= s.MaxThreads; w++ {
				go s.s3ToLocal(w, jobs, results)
			}

			for _, job = range actions["download"] {
				jobs <- job
			}
			close(jobs)

			for _, _ = range actions["download"] {
				<-results
			}
		}

		if len(actions["upload"]) > 0 {
			totalJobs = len(actions["upload"])
			jobs = make(chan s3diff.SyncItem, totalJobs)
			results = make(chan string, totalJobs)

			for w := 1; w <= s.MaxThreads; w++ {
				go s.localToS3(w, jobs, results)
			}

			for _, job = range actions["upload"] {
				jobs <- job
			}
			close(jobs)

			for _, _ = range actions["upload"] {
				<-results
			}
		}

		if len(actions["delete"]) > 0 {
			s.deleteFiles(actions["delete"])
		}

	} else {
		fmt.Println("sync status: OK")
	}
	return nil
}

func (s *Syncer) s3ToS3(id int, jobs <-chan s3diff.SyncItem, results chan<- string) {
	var (
		destination       *url.URL
		destinationBucket string
		destinationKey    string
		err               error
		job               s3diff.SyncItem
		mimeType          string
		mt                *mimetype.MIME
		source            *url.URL
		sourceBucket      string
		sourceFile        string
		sourceKey         string
	)

	for job = range jobs {
		if s.Dryrun == true {
			dryrun(job.Message)
			results <- "Done!"
		} else {
			fmt.Println(job.Message)
			source, err = url.Parse(job.Source)
			if err != nil {
				panic(err)
			}
			sourceBucket = source.Hostname()
			sourceKey = strings.TrimLeft(source.Path, string(os.PathSeparator))
			sourceFile = sourceBucket + string(os.PathSeparator) + sourceKey

			destination, err = url.Parse(job.Destination)
			if err != nil {
				panic(err)
			}
			destinationBucket = destination.Hostname()
			destinationKey = strings.TrimLeft(destination.Path, string(os.PathSeparator))

			mt, err = mimetype.DetectFile(job.Source)
			if err != nil {
				mimeType = "application/octet-stream"
			} else {
				mimeType = mt.String()
			}

			_, err = s.S3.CopyObject(&s3.CopyObjectInput{
				ACL:         &s.ACL,
				Bucket:      &destinationBucket,
				ContentType: &mimeType,
				CopySource:  &sourceFile,
				Key:         &destinationKey,
			})
		}
	}
}

func (s *Syncer) s3ToLocal(id int, jobs <-chan s3diff.SyncItem, results chan<- string) {
	var (
		destinationDirectory string
		err                  error
		job                  s3diff.SyncItem
		source               *url.URL
		sourceBucket         string
		sourceKey            string
	)
	for job = range jobs {
		if s.Dryrun == true {
			dryrun(job.Message)
			results <- "Done!"
		} else {
			fmt.Println(job.Message)
			source, err = url.Parse(job.Source)
			if err != nil {
				panic(err)
			}
			sourceBucket = source.Hostname()
			sourceKey = strings.TrimLeft(source.Path, string(os.PathSeparator))
			destinationDirectory = filepath.Dir(job.Destination)

			err = os.MkdirAll(destinationDirectory, 0755)
			if err != nil {
				panic(err) // We should log failed files and not kill the entire job
			}

			f, err := os.Create(job.Destination)
			if err != nil {
				panic(err) // We should log failed files and not kill the entire job
			}

			_, err = s.Downloader.Download(f, &s3.GetObjectInput{
				Bucket: &sourceBucket,
				Key:    &sourceKey,
			})

			if err != nil {
				fmt.Printf("failed to download %s to %s: %s\n", job.Source, job.Destination, err)
			}
		}
		results <- "Done!"
	}
}

func (s *Syncer) localToS3(id int, jobs <-chan s3diff.SyncItem, results chan<- string) {
	var (
		body              io.Reader
		destination       *url.URL
		destinationBucket string
		destinationKey    string
		err               error
		job               s3diff.SyncItem
		mimeType          string
		mt                *mimetype.MIME
		// resp        *s3manager.UploadOutput
	)
	for job = range jobs {
		if s.Dryrun == true {
			dryrun(job.Message)
			results <- "Done!"
		} else {
			fmt.Println(job.Message)
			destination, err = url.Parse(job.Destination)
			if err != nil {
				panic(err)
			}
			destinationBucket = destination.Hostname()
			destinationKey = strings.TrimLeft(destination.Path, string(os.PathSeparator))

			mt, err = mimetype.DetectFile(job.Source)
			if err != nil {
				mimeType = "application/octet-stream"
			} else {
				mimeType = mt.String()
			}

			body, err = os.Open(job.Source)
			if err != nil {
				panic(err)
			}

			_, err = s.Uploader.Upload(&s3manager.UploadInput{
				ACL:         &s.ACL,
				Body:        body,
				Bucket:      &destinationBucket,
				ContentType: &mimeType,
				Key:         &destinationKey,
			})

			if err != nil {
				fmt.Printf("failed to copy %s to %s\n", job.Source, job.Destination)
			}
		}
		results <- "Done!"
	}
}

func (s *Syncer) deleteFiles(fileList []s3diff.SyncItem) {
	var (
		job s3diff.SyncItem
	)

	for _, job = range fileList {
		if s.Dryrun == true {
			dryrun(job.Message)
		} else {
			fmt.Println(job.Message)
		}
	}
}
