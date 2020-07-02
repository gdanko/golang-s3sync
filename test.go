package main

import (
	"github.com/gdanko/golang-s3sync/pkg/s3sync"
)

func main() {
	var s s3sync.Syncer

	s = s3sync.Syncer{
		Profile: "default",
		// Source:  "s3://gdanko-test1/foo",
		Source:      "/usr/local/bin",
		Destination: "s3://gdanko-test2/bar",
		// Destination: "/usr/local/bin",
		Region: "us-west-2",
	}

	err := s.Sync()

	if err != nil {
		panic(err)
	}
}
