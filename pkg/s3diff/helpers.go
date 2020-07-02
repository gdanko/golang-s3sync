package s3diff

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/kylelemons/godebug/pretty"
)

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

func pathExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	} else {
		return true
	}
}

func validatePath(path string) (os.FileInfo, error) {
	var (
		err    error
		info   os.FileInfo
		target string
	)
	target, err = os.Readlink(path)
	if err == nil {
		if pathExists(target) == true {
			info, err = os.Stat(target)
			if err != nil {
				panic(err)
			}
		} else {
			return nil, fmt.Errorf("the symlink %s has a target %s but the target does not exist", path, target)
		}
	} else {
		info, err = os.Stat(path)
		if err != nil {
			panic(err)
		}
	}

	return info, nil
}

func mergeFileLists(l1, l2 map[string]FileInfo) map[string]FileInfo {
	for k, v := range l2 {
		l1[k] = v
	}

	return l1
}

func prettyPrint(item interface{}, exit bool) {
	pretty.Print(item)
	if exit == true {
		os.Exit(0)
	}
}
