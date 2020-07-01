package s3sync

import "fmt"

type Syncer struct {
	Destination string
	MaxThreads  int
	Profile     string
	Region      string
	Source      string
}

func (s Syncer) Sync() error {
	fmt.Println(s.Profile)
	return nil
}
