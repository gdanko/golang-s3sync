package s3diff

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/thoas/go-funk"
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

type SyncItem struct {
	Action      string
	Source      string
	Bucket      string
	Destination string
	Key         string
	MD5         string
	Message     string
	Path        string
	Size        int64
}

type Differ struct {
	Common                 map[string]FileInfo
	Debug                  bool
	Delete                 bool
	Destination            string
	DestinationBucket      string
	DestinationList        map[string]FileInfo
	DestinationMD5Mismatch map[string]FileInfo
	DestinationOnly        map[string]FileInfo
	DestinationPath        string
	DestinationRoot        string
	DestinationType        string
	S3                     *s3.S3
	Source                 string
	SourceBucket           string
	SourceList             map[string]FileInfo
	SourceMD5Mismatch      map[string]FileInfo
	SourceOnly             map[string]FileInfo
	SourcePath             string
	SourceType             string
	SyncList               map[string]SyncItem
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
	var (
		name string
		obj  FileInfo
	)
	// Put this in a "New" func which returns s3diff.Differ?
	d.Common = make(map[string]FileInfo)
	d.DestinationList = make(map[string]FileInfo)
	d.DestinationMD5Mismatch = make(map[string]FileInfo)
	d.DestinationOnly = make(map[string]FileInfo)
	d.SourceList = make(map[string]FileInfo)
	d.SourceMD5Mismatch = make(map[string]FileInfo)
	d.SourceOnly = make(map[string]FileInfo)
	d.SyncList = make(map[string]SyncItem)
	d.buildFileLists()

	for name, obj = range d.SourceList {
		if funk.Contains(d.DestinationList, name) {
			if d.DestinationList[name].MD5 == d.SourceList[name].MD5 {
				d.Common[name] = obj
			} else {
				d.SourceMD5Mismatch[name] = obj
			}
		} else {
			d.SourceOnly[name] = obj
		}
	}

	for name, obj = range d.DestinationList {
		if funk.Contains(d.SourceList, name) {
			if d.SourceList[name].MD5 == d.DestinationList[name].MD5 {
				d.Common[name] = obj
			} else {
				d.DestinationMD5Mismatch[name] = obj
			}
		} else {
			d.DestinationOnly[name] = obj
		}
	}
}

func (d *Differ) buildFileLists() {
	fmt.Println("building file list...")
	switch d.SourceType {
	case "local":
		d.getLocalFiles(d.SourcePath, &d.SourceList)
	case "s3":
		d.getS3Files(d.SourcePath, d.SourceBucket, &d.SourceList)
	}

	switch d.DestinationType {
	case "local":
		d.getLocalFiles(d.DestinationPath, &d.DestinationList)
	case "s3":
		d.getS3Files(d.DestinationPath, d.DestinationBucket, &d.DestinationList)
	}
	fmt.Println("done")
}

func (d *Differ) getLocalFiles(path string, fileList *map[string]FileInfo) {
	err = filepath.Walk(path, func(item string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		info, err = validatePath(item)
		if err == nil {
			if info.IsDir() == false {
				md5sum, err = md5checksum(item)
				if err != nil {
					panic(err) // Handle this error soon
				}

				p1 := item
				p2 := filepath.Dir(path)
				key, _ := filepath.Rel(p2, p1)
				strippedKey := strings.Join(strings.Split(key, string(os.PathSeparator))[1:], string(os.PathSeparator))

				(*fileList)[strippedKey] = FileInfo{
					Key:       strippedKey,
					Directory: info.IsDir(),
					Path:      item,
					Dirname:   filepath.Dir(path),
					Filename:  filepath.Base(path),
					Size:      info.Size(),
					MD5:       md5sum,
				}
			}
		}

		return nil
	})
}

func (d *Differ) getS3Files(path string, bucket string, fileList *map[string]FileInfo) {
	prefix := fmt.Sprintf("%s/", path)
	resp, err := d.S3.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: &bucket,
		Prefix: &prefix,
	})

	if err != nil {
		panic(err) // Handle this error soon
	}

	for _, fileObj := range resp.Contents {
		key := *fileObj.Key
		if !strings.HasSuffix(key, string(os.PathSeparator)) && int64(*fileObj.Size) != 0 {
			key := filepath.Join(filepath.Dir(key), filepath.Base(key))
			strippedKey := strings.Join(strings.Split(key, string(os.PathSeparator))[1:], string(os.PathSeparator))

			(*fileList)[strippedKey] = FileInfo{
				Key:      key,
				Dirname:  filepath.Dir(key),
				Filename: filepath.Base(key),
				Size:     int64(*fileObj.Size),
				MD5:      strings.ReplaceAll(*fileObj.ETag, "\"", ""),
			}
		}
	}
}

func (d *Differ) GenerateSyncList() {
	var (
		name       string
		obj        FileInfo
		sourceFile string
		syncItem   SyncItem
		toSync     map[string]FileInfo
	)

	toSync = mergeFileLists(d.SourceOnly, d.SourceMD5Mismatch)

	for name, obj = range toSync {
		if d.SourceType == "s3" && d.DestinationType == "s3" {
			sourceFile = filepath.Join(
				filepath.Dir(d.Source),
				obj.Key,
			)
			syncItem = d.getSyncItem(sourceFile)
			syncItem.Action = "copy"
			syncItem.Message = fmt.Sprintf("%s: %s to %s", syncItem.Action, syncItem.Source, syncItem.Destination)

		} else if d.SourceType == "s3" && d.DestinationType == "local" {
			// this is not ideal
			sourceFile = "s3://" + filepath.Join(d.SourceBucket, obj.Key)
			syncItem = d.getSyncItem(sourceFile)
			syncItem.Action = "download"
			syncItem.Message = fmt.Sprintf("%s: %s to %s", syncItem.Action, syncItem.Source, syncItem.Destination)

		} else if d.SourceType == "local" && d.DestinationType == "s3" {
			sourceFile = obj.Path
			syncItem = d.getSyncItem(sourceFile)
			syncItem.Action = "upload"
			syncItem.Message = fmt.Sprintf("%s: %s to %s", syncItem.Action, syncItem.Source, syncItem.Destination)
		}

		if obj.MD5 != "" {
			syncItem.MD5 = obj.MD5
		}
		syncItem.Size = obj.Size

		d.SyncList[name] = syncItem
	}

	if d.Delete == true {
		for name, obj = range d.DestinationOnly {
			if d.DestinationType == "s3" {
				d.SyncList[name] = SyncItem{
					Action:  "delete",
					Message: fmt.Sprintf("delete: s3://%s/%s", d.DestinationBucket, obj.Key),
					Bucket:  d.DestinationBucket,
					Key:     obj.Key,
				}
			} else if d.DestinationType == "local" {
				d.SyncList[name] = SyncItem{
					Action:  "delete",
					Message: fmt.Sprintf("delete: %s", obj.Path),
					Path:    obj.Path,
				}
			}
		}
	}
}

func (d *Differ) getSyncItem(sourceFile string) SyncItem {
	relativePath, _ := filepath.Rel(d.Source, sourceFile)
	return SyncItem{
		Source: sourceFile,
		// Destination: filepath.Join(d.Destination, relativePath),
		// filepath.Join converts s3:// to s3:/
		// will fix later
		Destination: d.Destination + "/" + relativePath,
	}
}
