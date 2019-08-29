package bucket

import (
	"github.com/aws/aws-sdk-go/service/s3"
	"reflect"
	"testing"
	"time"
)

func defaultBucket() *Bucket {
	return &Bucket{
		Name:            "Test",
		Analysis:        defaultAnalysis(),
		AnalysisChannel: make(chan *Analysis, 1),
	}
}

func defaultAnalysis() *Analysis {
	a := newAnalysis(&Bucket{}, 0)
	return a
}

func mockListObjectsOutput(key, ownerid string, size int64) *s3.ListObjectsOutput {
	listObjectsOutput := &s3.ListObjectsOutput{}
	timestamp := time.Now()
	owner := s3.Owner{ID: &ownerid}
	listObjectsOutput.SetContents(
		[]*s3.Object{
			&s3.Object{LastModified: &timestamp, Owner: &owner, Key: &key, Size: &size},
		})
	return listObjectsOutput
}

func TestDefaultAnalysisJSON(t *testing.T) {
	a := defaultAnalysis()

	expected := `{"Name":"","CreationDate":"0001-01-01T00:00:00Z","LastModified":"0001-01-01T00:00:00Z","TotalSize":0,"DisplayObjectCount":0,"TotalCount":0,"SizePerOwnerID":{},"Objects":[],"Error":null}`

	if a.JSON() != expected {
		t.Errorf("Invalid Analysis JSON:%s is not %s", a.JSON(), expected)

	}
}

func TestDefaultAnalysisString(t *testing.T) {
	a := defaultAnalysis()

	expected := "Name: \nObjectCount: 0\nTotalSize: 0B\nCreationDate: 0001-01-01T00:00:00Z\nLastModified: 0001-01-01T00:00:00Z\nObjects:\nTotalSizePerAccount:\n"

	if a.String() != expected {
		t.Errorf("Invalid default Analysis String():%s", a.String())
	}
}

func TestAnalysisCallback(t *testing.T) {

	b := defaultBucket()
	a := b.Analysis

	b.handleListObjectOutput(mockListObjectsOutput("key1", "ownerid1", 1024), false)
	b.handleListObjectOutput(mockListObjectsOutput("key2", "ownerid2", 1024), true)
	if a.TotalSize != 2048 {
		t.Errorf("handleListObjectOutput callback error got TotalSize %d", a.TotalSize)
	}
	if a != <-b.AnalysisChannel {
		t.Errorf("Analysis not received on channel")
	}
}

func TestOnlyN(t *testing.T) {
	s := make([]string, 0)
	s = append(s, "a", "b", "c", "d", "e", "f")

	last3 := []string{"d", "e", "f"}
	first3 := []string{"a", "b", "c"}
	none := []string{}

	if !reflect.DeepEqual(last3, onlyN(s, 3)) {
		t.Errorf("Last 3: %s", onlyN(s, 3))
	}
	if !reflect.DeepEqual(first3, onlyN(s, -3)) {
		t.Errorf("First 3 is %s", onlyN(s, -3))
	}
	if !reflect.DeepEqual(s, onlyN(s, -10)) {
		t.Errorf("first 10: %s", onlyN(s, -10))
	}
	if !reflect.DeepEqual(s, onlyN(s, 10)) {
		t.Errorf("last 10: %s", onlyN(s, 10))
	}
	if !reflect.DeepEqual(none, onlyN(s, 0)) {
		t.Errorf("This should be empty: %s", onlyN(s, 0))
	}
}

func TestByteCountToHuman(t *testing.T) {
	byteCount := int64(12345678)
	expected := "12.3MB"
	got := byteCountToHuman(byteCount)
	if expected != got {
		t.Errorf("Expected %s but got %s", expected, got)
	}
	byteCount = int64(1)
	expected = "1B"
	got = byteCountToHuman(byteCount)
	if expected != got {
		t.Errorf("Expected %s but got %s", expected, got)
	}
}

func TestFilterBuckets(t *testing.T) {
	buckets := []*Bucket{&Bucket{Name: "foobar", CreationDate: time.Now()}}

	filteredBuckets := filterBuckets(buckets, "foo", "")
	if 1 != len(filteredBuckets) {
		t.Errorf("failed positive include")
	}

	filteredBuckets = filterBuckets(buckets, "bad", "")
	if 0 != len(filteredBuckets) {
		t.Errorf("failed negative include")
	}

	filteredBuckets = filterBuckets(buckets, "", "foo")
	if 0 != len(filteredBuckets) {
		t.Errorf("failed positive exclude")
	}

	filteredBuckets = filterBuckets(buckets, "", "not there")
	if 1 != len(filteredBuckets) {
		t.Errorf("failed negative exclude")
	}
}
