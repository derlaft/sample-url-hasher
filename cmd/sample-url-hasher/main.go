package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/derlaft/sample-url-hasher/urlhasher"
)

func main() {

	parallel := flag.Int("parallel", 10, "Maximum number of parallel workers")

	flag.Parse()

	if *parallel <= 0 {
		flag.Usage()
		os.Exit(1)
	}

	// create a hasher
	h := &urlhasher.Hasher{
		Parallel: *parallel,
		OnDone: func(ctx context.Context, url string, hash []byte, err error) {
			if err != nil {
				// error handling: write errors to stderr
				log.Printf("Error, could not fetch url (%v): %v", url, err)
				return
			}

			// write output to stdout
			fmt.Printf("%v %x\n", url, hash)
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// listen to the system signal: allow ^C
	go abortOnSignal(ctx, cancel)

	err := h.Start(ctx, getURLs())
	if err != nil {
		log.Printf("hasher.Start error: %v", err)
		os.Exit(1)
	}
}

// getURLs extracts URLs from flag arguments
// also adds http:// schema if necessary
func getURLs() []string {

	var output = make([]string, 0, flag.NArg())
	for _, arg := range flag.Args() {

		parsedURL, err := url.Parse(arg)
		if err != nil {
			log.Printf("Skipping invalid url %v: %v", arg, err)
			continue
		}

		// add http:// to requests to satisfy examples
		if parsedURL.Scheme == "" {
			parsedURL.Scheme = "http"
		}

		output = append(output, parsedURL.String())
	}

	return output
}

// abortOnSignal calls given cancel func on SIGINT
func abortOnSignal(ctx context.Context, onSignal func()) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// wait for both ctx.Done() and sigs
	select {
	case <-sigs:
	case <-ctx.Done():
	}

	onSignal()

}
