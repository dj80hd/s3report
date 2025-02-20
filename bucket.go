package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Analysis contains summary data about the objects in the bucket
type Analysis struct {
	Name               string
	CreationDate       time.Time
	LastModified       time.Time
	TotalSize          int64
	DisplayObjectCount int
	TotalCount         int
	SizePerOwnerID     map[string]int64
	Objects            []string //TODO: Use []*s3.Object ?
	Error              error
}

// newAnalysis is Analysis constuctor
func newAnalysis(b *Bucket, displayObjectCount int) *Analysis {
	a := Analysis{
		Name:               b.Name,
		DisplayObjectCount: displayObjectCount,
		CreationDate:       b.CreationDate,
		SizePerOwnerID:     make(map[string]int64, 0),
		Objects:            make([]string, 0),
	}
	return &a
}

// JSON format of Analysis
func (a *Analysis) JSON() string {
	j, err := json.Marshal(*a)
	if err != nil {
		return err.Error()
	}
	return string(j)
}

// String format of Analysis
func (a *Analysis) String() string {
	var out strings.Builder
	fmt.Fprintf(&out, "Name: %s\nObjectCount: %d\nTotalSize: %s\nCreationDate: %s\nLastModified: %s\n",
		a.Name,
		a.TotalCount,
		byteCountToHuman(a.TotalSize),
		a.CreationDate.Format(time.RFC3339),
		a.LastModified.Format(time.RFC3339))
	fmt.Fprintf(&out, "Objects:\n")
	for _, o := range a.Objects {
		fmt.Fprintf(&out, " * %s\n", o)
	}
	fmt.Fprintf(&out, "TotalSizePerAccount:\n")
	for owner, size := range a.SizePerOwnerID {
		fmt.Fprintf(&out, " * %s/%s %s\n", byteCountToHuman(size), byteCountToHuman(a.TotalSize), owner)
	}
	return out.String()
}

// processObject is called for every object in the bucket
func (a *Analysis) processObject(object *s3.Object) {
	a.TotalCount++

	a.TotalSize = a.TotalSize + *object.Size

	a.SizePerOwnerID[*object.Owner.ID] += *object.Size

	if a.LastModified.Before(*object.LastModified) {
		a.LastModified = *object.LastModified
	}

	objectString := fmt.Sprintf("%s %s %s",
		(*object.LastModified).Format(time.RFC3339),
		byteCountToHuman(*object.Size),
		*object.Key)

	a.Objects = append(a.Objects, objectString) //FIXME: horrible names!
}

// Bucket represents an s3 bucket in the s3report
type Bucket struct {
	Name            string
	CreationDate    time.Time
	Analysis        *Analysis
	AnalysisChannel chan *Analysis //TODO: This is wonky
}

// Analyze uses the handleListObjectsOutput callback to start collecting data
func (b *Bucket) Analyze(ch chan *Analysis, displayObjectCount int) {
	b.Analysis = newAnalysis(b, displayObjectCount)
	b.AnalysisChannel = ch

	svc, err := s3Service(b.Name)
	if err != nil {
		b.analysisError(err)
		return
	}

	params := &s3.ListObjectsInput{Bucket: aws.String(b.Name)}
	if err := svc.ListObjectsPages(params, b.handleListObjectOutput); err != nil {
		b.analysisError(err)
		return
	}

}

// analysisError marks the analysis as error and sends it to the AnalysisChannel
func (b *Bucket) analysisError(err error) {
	b.Analysis.Error = err
	b.AnalysisChannel <- b.Analysis
}

// handleListObjectOutput is the callback for a list objects operation; aws s3 api is limited to 1000 objects at a time
func (b *Bucket) handleListObjectOutput(page *s3.ListObjectsOutput, lastPage bool) bool {
	for _, object := range page.Contents {
		b.Analysis.processObject(object)
	}
	if lastPage {
		b.completeAnalysis()
	}
	return !lastPage
}

// completeAnalysis finalizes the Analysis object and sends it on the AnalysisChannel
// TODO: make this idempotent to avoid sending the analysis twice
func (b *Bucket) completeAnalysis() {
	a := b.Analysis

	//limit to n objects TODO: This is kinda wonky
	sort.Strings(a.Objects)
	a.Objects = onlyN(a.Objects, a.DisplayObjectCount)

	b.AnalysisChannel <- a
}

// GetBuckets gets all the buckets in the current account
func GetBuckets(include, exclude string) ([]*Bucket, error) {
	buckets, err := getAllBuckets()
	if err != nil {
		return nil, err
	}
	buckets = filterBuckets(buckets, include, exclude)
	return buckets, nil
}

// onlyN returns last n (n positive) or first (n negative) strings from a slice of strings
func onlyN(a []string, n int) []string {
	n = -n
	if n > 0 {
		if n > len(a) {
			n = len(a)
		}
		a = a[:n]
	} else {
		if -n > len(a) {
			n = -len(a)
		}
		a = a[len(a)+n:]
	}
	return a
}

// byteCountToHuman converts a byte count e.g 12345678 to a human readable form e.g. 12.3MB
func byteCountToHuman(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(b)/float64(div), "kMGTPE"[exp])
}

// s3Service gets a suitable s3Service to access a bucket, or all buckets (bucketName="")
func s3Service(bucketName string) (*s3.S3, error) {
	s, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	svc := s3.New(s, &aws.Config{})
	if bucketName == "" {
		return svc, nil
	}

	//If bucket has a LocationConstraint, use config to access through that region specifically
	result, err := svc.GetBucketLocation(&s3.GetBucketLocationInput{Bucket: &bucketName})
	if err != nil {
		return nil, err
	}
	if result.LocationConstraint != nil {
		s, err := session.NewSession()
		if err != nil {
			return nil, err
		}
		svc = s3.New(s, &aws.Config{Region: result.LocationConstraint})
	}

	return svc, nil
}

// filterBuckets filters a list of buckets using the include and exclude substrings
func filterBuckets(buckets []*Bucket, include, exclude string) []*Bucket {
	filtered := make([]*Bucket, 0)
	for _, b := range buckets {
		if strings.Contains(b.Name, include) && (len(exclude) == 0 || !strings.Contains(b.Name, exclude)) {
			filtered = append(filtered, b)
		}
	}
	return filtered
}

// getAllBuckets gets all buckets for the current account
func getAllBuckets() ([]*Bucket, error) {
	buckets := make([]*Bucket, 0)

	svc, err := s3Service("")
	if err != nil {
		return nil, err
	}

	resp, err := svc.ListBuckets(&s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}

	for _, b := range resp.Buckets {
		bucket := &Bucket{Name: *b.Name, CreationDate: *b.CreationDate}
		buckets = append(buckets, bucket)
	}
	return buckets, nil
}
