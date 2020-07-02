package s3diff

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/kylelemons/godebug/pretty"
)

type FileInfo struct {
	Directory bool
	Dirname   string
	Filename  string
	Key       string
	MD5       string
	Path      string
	Size      int64
}

type Differ struct {
	Debug             bool
	Delete            bool
	Destination       string
	DestinationBucket string
	DestinationList   []FileInfo
	DestinationPath   string
	DestinationRoot   string
	DestinationType   string
	S3                *s3.S3
	Source            string
	SourceBucket      string
	SourceList        []FileInfo
	SourcePath        string
	SourceType        string
}

var (
	err    error
	md5sum string
)

func (d *Differ) DetermineTypes() error {
	var (
		err error
		u   *url.URL
	)
	// Source
	u, err = url.Parse(d.Source)
	if err != nil {
		return err
	}

	if u.Scheme == "s3" {
		d.SourceType = "s3"
		d.SourceBucket = u.Hostname()
		d.SourcePath = strings.TrimLeft(u.Path, string(os.PathSeparator))
	} else {
		d.SourceType = "local"
		d.SourcePath = d.Source
	}

	// Destination
	u, err = url.Parse(d.Destination)
	if err != nil {
		return err
	}

	if u.Scheme == "s3" {
		d.DestinationType = "s3"
		d.DestinationBucket = u.Hostname()
		d.DestinationPath = strings.TrimLeft(u.Path, string(os.PathSeparator))
	} else {
		d.DestinationType = "local"
		d.DestinationPath = d.Destination
		if filepath.IsAbs(d.Destination) == false {
			d.DestinationPath, err = filepath.Abs(d.Destination)
			if err != nil {
				return err
			}

		}
		d.DestinationRoot = filepath.Dir(d.DestinationPath)
	}

	return nil
}

func (d *Differ) Diff() {
	d.buildFileList()
}

func (d *Differ) buildFileList() {
	fmt.Println("building file list...")
	switch d.SourceType {
	case "local":
		d.getLocalFiles(d.SourcePath)
	case "s3":
		d.getS3Files(d.SourcePath, d.SourceBucket)
	}

	switch d.DestinationType {
	case "local":
		d.getLocalFiles(d.DestinationPath)
	case "s3":
		d.getS3Files(d.DestinationPath, d.DestinationBucket)
	}
}

func (d *Differ) getLocalFiles(path string) {
	err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() == false {
			md5sum, err = md5checksum(path)
			if err != nil {
				panic(err) // Handle this error soon
			}

			d.SourceList = append(d.SourceList, FileInfo{
				Directory: info.IsDir(),
				Path:      path,
				Dirname:   filepath.Dir(path),
				Filename:  filepath.Base(path),
				Size:      info.Size(),
				MD5:       md5sum,
			})
		}

		return nil
	})
}

func (d *Differ) getS3Files(path string, bucket string) {
	prefix := fmt.Sprintf("%s/", path)
	resp, err := d.S3.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: &bucket,
		Prefix: &prefix,
	})

	if err != nil {
		panic(err) // Handle this error soon
	}

	for _, fileObj := range resp.Contents {
		// prettyPrint(fileObj, false)
		key := *fileObj.Key
		if !strings.HasSuffix(key, string(os.PathSeparator)) && int64(*fileObj.Size) != 0 {
			d.DestinationList = append(d.DestinationList, FileInfo{
				Dirname:  filepath.Dir(key),
				Filename: filepath.Base(key),
				Key:      filepath.Join(filepath.Dir(key), filepath.Base(key)),
				Size:     int64(*fileObj.Size),
				MD5:      strings.ReplaceAll(*fileObj.ETag, "\"", ""),
			})
		}
	}
	prettyPrint(d.DestinationList, true)
}

func md5checksum(path string) (checksum string, err error) {
	hasher := md5.New()
	s, err := ioutil.ReadFile(path)
	hasher.Write(s)
	if err != nil {
		return "", err
	}
	checksum = hex.EncodeToString(hasher.Sum(nil))

	return checksum, nil
}

func prettyPrint(item interface{}, exit bool) {
	pretty.Print(item)
	if exit == true {
		os.Exit(0)
	}
}
