# s3sync
s3sync is a library and CLI designed to sync s3. It can do:
* s3 <> s3
* s3 <> local
* local <> s3

## Installation
Coming soon!

## Features
s3sync provides an almost rysnc-style way of syncing with s3.
* Dryrun mode to show you what will be done.
* Delete files from the source if they don't exist in the destination.
* Verify the files after copying.
* Include multiple patterns. (COMING SOON)
* Exclude multiple patterns. (COMING SOON)

## Options
* Source - The source, either a local path or s3://bucket/path.
* Destination - The destination, either a local path or s3://bucket/path.
* MaxThreads - The number of threads to use while performing copies. Defaults to 12.
* Profile: The AWS profile.
* Region: The AWS region.
* Delete: Delete files from the destination that do not exist in the source.
* Verify: Perform an md5 checksum validation.
* Debug: Enable debug mode (does nothing now)
* Dryrun: Show what would be done without making changes.

## Use as a Library
Using s3sync as a library is very easy. You create an instance of the s3sync.Syncer struct and initiate the sync. For example:
```
import "github.com/gdanko/golang-s3sync/pkg/s3sync"

var syncer s3sync.Syncer

syncer = s3sync.Syncer{
	Source:      "s3://my-bucket/foo",
	Destination: "/usr/local/foo",
	MaxThreads:  15,
	Profile:     "default",
	Region:      "us-east-1",
	Delete:      true,
	Verify:      true,
	Debug:       false,
	Dryrun:      false,
}

err = syncer.Sync()
if err != nil {
	// handle the error
}
```

## CLI Use
The CLI is a wrapper for the library. The help looks like this:
```
[gdanko@gdanko-mac ~]$ s3sync -h
Usage:
  s3sync [OPTIONS]

Application Options:
  -s, --source=      The source, either absolute local path or s3://<bucket>/<path>
  -d, --destination= The destination, either absolute local path or s3://<bucket>/<path>
  -i, --include=     COMING SOON! Include <pattern>. Can be used more than once.
  -e, --exclude=     COMING SOON! Exclude <pattern>. Can be used more than once.
  -m, --max-threads= The maximum number of threads to use while copying. (default: 12)
  -p, --profile=     The AWS profile to use. (default: default)
  -r, --region=      The AWS region to use.
      --delete       Delete files on the destination side that do not exist on the source.
  -v, --verify       Verify the files after copying.
      --debug        Display debug output.
  -n, --dryrun       Show what would be done but change nothing.

Help Options:
  -h, --help         Show this help message
  ```
  
  # TODO
  * Implement includes and excludes.
  * Do not panic on copy errors, but instead log them at the end.
  * Implement the verification.
  * Hide more Aram jabs in the code.
  