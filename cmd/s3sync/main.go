package main

import (
	"fmt"
	"os"

	"github.com/gdanko/golang-s3sync/pkg/s3sync"
	flags "github.com/jessevdk/go-flags"
)

type Options struct {
	Source      string   `short:"s" long:"source" description:"The source, either absolute local path or s3://<bucket>/<path>" required:"true"`
	Destination string   `short:"d" long:"destination" description:"The destination, either absolute local path or s3://<bucket>/<path>" required:"true"`
	Include     []string `short:"i" long:"include" description:"COMING SOON! Include <pattern>. Can be used more than once."`
	Exclude     []string `short:"e" long:"exclude" description:"COMING SOON! Exclude <pattern>. Can be used more than once."`
	MaxThreads  int      `short:"m" long:"max-threads" description:"The maximum number of threads to use while copying." default:"12"`
	Profile     string   `short:"p" long:"profile" description:"The AWS profile to use." required:"true" default:"default"`
	Region      string   `short:"r" long:"region" description:"The AWS region to use." required:"true"`
	Delete      bool     `long:"delete" description:"Delete files on the destination side that do not exist on the source."`
	Verify      bool     `short:"v" long:"verify" description:"Verify the files after copying."`
	Debug       bool     `long:"debug" description:"Display debug output."`
	Dryrun      bool     `short:"n" long:"dryrun" description:"Show what would be done but change nothing."`
	Aram        bool     `short:"a" long:"aram" description:"Tell me about Aram." hidden:"true"`
}

func main() {
	var (
		err      error
		flagsErr *flags.Error
		ok       bool
		opts     Options
		syncer   s3sync.Syncer
	)

	// Parse the options
	parser1 := flags.NewParser(&opts, flags.Default)
	if _, err = parser1.Parse(); err != nil {
		if flagsErr, ok = err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	if opts.Aram {
		fmt.Println("That Aram is a real bully.")
	}

	if opts.MaxThreads < 1 {
		fmt.Println("--max-threads cannot be less than 1.")
		os.Exit(1)
	}

	// Validate region

	syncer = s3sync.Syncer{
		Source:      opts.Source,
		Destination: opts.Destination,
		MaxThreads:  opts.MaxThreads,
		Profile:     opts.Profile,
		Region:      opts.Region,
		Delete:      opts.Delete,
		Verify:      opts.Verify,
		Debug:       opts.Debug,
		Dryrun:      opts.Dryrun,
	}

	err = syncer.Sync()
	if err != nil {
		fmt.Println(err)
	}
}
