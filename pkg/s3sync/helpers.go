package s3sync

import (
	"fmt"
	"os"

	"github.com/kylelemons/godebug/pretty"
)

func (s *Syncer) validate() error {
	// errorList := []string
	if s.ACL == "" {
		s.ACL = "private"
	}

	if s.Destination == "" {
		return fmt.Errorf("the Destination option is required")
	}

	if s.MaxThreads == 0 {
		s.MaxThreads = 12
	}

	if s.Profile == "" {
		s.Profile = "default"
	}

	if s.Region == "" {
		return fmt.Errorf("the Region option is required")
	}

	if s.Source == "" {
		return fmt.Errorf("the Source option is required")
	}

	return nil
}

func prettyPrint(item interface{}, exit bool) {
	pretty.Print(item)
	if exit == true {
		os.Exit(0)
	}
}
