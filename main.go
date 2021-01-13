package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func fail(msg string) {
	fmt.Fprintf(os.Stderr, msg)
	os.Exit(1)
}

func main() {
	// parse our arguments
	displayObjectCount := flag.Int("count", -5,
		"Number objects to show for each bucket. 5 = five newest, -5 = five oldest.")

	timeout := flag.Int("timeout", 600,
		"Number of seconds to wait for all analysis to complete.")

	include := flag.String("include", "",
		"only include buckets whose name includes this string. Default is all buckets")

	exclude := flag.String("exclude", "",
		"exclude buckets whose name includes this string. Default is no buckets.")

	jsonOutput := flag.Bool("json", false, "json output")

	flag.Parse()

	//Get list of buckets filtered by user options.
	buckets, err := GetBuckets(*include, *exclude)
	if err != nil {
		fail("Could not get buckets: " + err.Error())
	}

	if len(buckets) == 0 {
		fail("No buckets Found")
	}

	//Calculate analysis of each bucket in parallel
	ch := make(chan *Analysis, 10)
	for _, b := range buckets {
		go b.Analyze(ch, *displayObjectCount)
	}

	//Output results as they arrive.
	for i := 0; i < len(buckets); i++ {
		select {
		case analysis := <-ch:
			if *jsonOutput {
				fmt.Printf("%s\n", analysis.JSON())
			} else {
				fmt.Printf("%s\n", analysis)
			}
		case <-time.After(time.Duration(*timeout) * time.Second):
			fmt.Println("timeout")
			os.Exit(1)
		}
	}
}
