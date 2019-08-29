# Description
s3report analyzes s3 buckets and reports the results.

Specifically it reports the following information:

* Bucket Names
* Creation date of the bucket
* Last Modified date per bucket
* Number of files per bucket
* Total Size of files per bucket, per account
* A list of N number of files that are either the most recently or least recently modified in the bucket or account

And supports the following features:

* Human readable byte sizes
* Ability to specify the type (newest/oldest) and number N as parameters
* Ability to filter buckets in and out of the results
* Optional JSON output

The solution is written in 3 languages: [golang](main.go), [Bash](s3report.sh) and [Ruby](s3report.rb).
to model the best practices of each command line tool language.

## Requirements
* [go](https://golang.org/doc/install)

## Installation
```
make && cp s3report /usr/local/bin
```
## Usage

```
Usage of ./s3report:
  -count int
    	Number objects to show for each bucket. 5 means five newest, -5 means five oldest. (default -5)
  -exclude string
    	exclude buckets whose name includes this string. Default is include all buckets.
  -include string
    	only include buckets whose name includes this string. (default is include all buckets)
  -json
    	json output
  -timeout int
    	Number of seconds to wait for all analysis to complete. (default 600)

```

## Usage Examples

Get help:
```
s3report --help
```

Report on all buckets whose name includes 'foo'

```
s3report --include foo 
```
Output:
```
Name: foo-bucket
ObjectCount: 208
TotalSize: 220.1GB
CreationDate: 2018-03-30T21:13:24Z
LastModified: 2018-04-14T00:12:07Z
Objects:
 * 2018-03-31T00:00:03Z 1.8kB 1.tgz
 * 2018-03-31T00:00:03Z 1.8kB 2.tgx
 * 2018-03-31T00:00:03Z 330B 3.tgz
TotalSizePerAccount:
 * 220.1GB/220.1GB 41e9e2e433166ae95388d7bf0589299200200200292929299191111111d3d1af

...
```

Report on all buckets *except* thoee whose names include 'foo'

```
s3report --exclude foo 
```
Output:
```
Name: bar-bucket
ObjectCount: 3355
TotalSize: 782.4GB
CreationDate: 2017-05-21T21:08:47Z
LastModified: 2018-04-14T00:17:32Z
Objects:
 * 2017-05-22T20:45:52Z 280.1kB 1.tgz
 * 2017-05-22T20:45:52Z 429B 2.tgz
 * 2017-05-22T20:45:52Z 61.9MB 3.tgz
TotalSizePerAccount:
 * 782.4GB/782.4GB 41e9e2e433166ae96424784738473847384783743847384738748222222221af
```

## Other Versions
For demonstration purposes of performance and maintainability, ruby and bash versions of s3report are contained in this repo.  

### Ruby
Requirements: [ruby](https://www.ruby-lang.org/en/documentation/installation/) 2.x
```
gem install bundler 
bundle install
ruby s3report.rb --inlucde foo
```
### Bash
Requirements: bash 4.x, [jq](https://stedolan.github.io/jq/download/) 1.x, [aws-cli](https://aws.amazon.com/cli/) 1.x

bash version is limited to 100 lines per [Google's shell script style guide](https://google.github.io/styleguide/shell.xml).  

The --json feature is not suppored.
```
./s3report.sh --include foo 
```

## TODO
* Testing for ruby version
* Better Testing/Mocking for golang version with github.com/stretchr/testify
