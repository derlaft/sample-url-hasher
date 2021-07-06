package urlhasher

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const DefaultFetchTimeout = time.Second * 60

type OnDone func(ctx context.Context, url string, hash []byte, err error)

type Hasher struct {
	// Parallel indicates the maximum amount of concurrent workers allowed
	Parallel int
	// FetchTimeout sets a timeout for each URL to be fetched
	// If not set, defaults to DefaultFetchTimeout
	FetchTimeout time.Duration
	OnDone       OnDone
}

func (h *Hasher) doHTTPRequest(ctx context.Context, reqURL string) ([]byte, error) {

	// create a http request
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("could not create http request: %v", err)
	}

	// execute it with default client
	// to potentially benefit from keeping some idle connections open
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not perform HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// TODO: do we need to abort on non-200 responses?

	hash := md5.New()

	// copy response body directly to hasher
	// and thus efficiently hashing large payloads without reading them into memory
	_, err = io.Copy(hash, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could hash HTTP response: %v", err)
	}

	return hash.Sum(nil), nil
}

func (h *Hasher) fetchIndividualURL(ctx context.Context, url string) {

	// wrap context with one implementing timeout
	ctx, cancel := context.WithTimeout(ctx, h.FetchTimeout)
	defer cancel()

	// do the request
	hash, err := h.doHTTPRequest(ctx, url)

	// notify the client
	h.OnDone(ctx, url, hash, err)
}

func (h *Hasher) worker(ctx context.Context, stop func(), urls <-chan string) {
	for {
		select {
		case <-ctx.Done():
			// context cancelled - return
			return
		case url, ok := <-urls:
			if !ok {
				// input channel is closed - nothing to do anymore
				return
			}

			// data received, process
			h.fetchIndividualURL(ctx, url)
		}
	}
}

// Start queries each provided URL and hashes the bodies
func (h *Hasher) Start(ctx context.Context, urls []string) error {

	// make sure all the workers are closed
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if h.FetchTimeout == 0 {
		h.FetchTimeout = DefaultFetchTimeout
	}

	// Spawn h.Parallel workers, feed them via a channel.
	// I would recommend not doing anything I am doing here
	// and just use golang.org/x/sync/semaphore instead.
	// Hopefully it's going to be a part of stdlib soon
	var inputChannel = make(chan string)

	go func() {
		// drain channel on abort
		<-ctx.Done()
		for range inputChannel {
		}
	}()

	// keep track of the number of workers we spawned
	var wg sync.WaitGroup
	wg.Add(h.Parallel)

	// start workers
	for i := 0; i < h.Parallel; i++ {
		go func() {
			defer wg.Done()
			h.worker(ctx, cancel, inputChannel)
		}()
	}

	// feed workers
	for _, url := range urls {
		inputChannel <- url
	}
	close(inputChannel)

	// wait for all the workers to be closed
	wg.Wait()

	// ideally, errors returned from OnDone callbacks should somehow be returned here
	// however, that is out of scope of this task
	return ctx.Err()
}
