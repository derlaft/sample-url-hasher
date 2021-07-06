package urlhasher

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
)

func TestSimplePayload(t *testing.T) {

	// idea of the test: make a very simple successfull case

	const (
		samplePayload     = "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
		samplePayloadHash = "6330d6a09e56387e4dd59502418fa642"
	)

	// create a sample http server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, samplePayload)
	}))
	defer ts.Close()

	var executed int

	hasher := Hasher{
		Parallel: 1,
		OnDone: func(ctx context.Context, url string, hash []byte, err error) {
			if err != nil {
				t.Fatalf("Unexpected error in OnDone: %v", err)
			}

			if url != ts.URL {
				t.Fatalf("Unexpected URL in callback: %v", url)
			}

			gotHash := fmt.Sprintf("%x", hash)
			if gotHash != samplePayloadHash {
				t.Fatalf("Unexpected payload hash: %v (expected %v)", gotHash, samplePayloadHash)
			}

			executed++
		},
	}

	err := hasher.Start(context.Background(), []string{ts.URL})
	if err != nil {
		t.Fatalf("Unexpected error in Start: %v", err)
	}

	if executed != 1 {
		t.Fatalf("Unexpected number of executions: %v (expected 1)", executed)
	}
}

func TestSimpleError(t *testing.T) {

	// idea of the test: make a very simple error case

	const samplePayload = "https://www.youtube.com/watch?v=dQw4w9WgXcQ"

	// create a sample http server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// emulate broken connection by setting Content-Length to an invalid value
		w.Header().Set("Content-Length", strconv.Itoa(len(samplePayload)+42))

		fmt.Fprintln(w, samplePayload)
	}))
	defer ts.Close()

	var executed int

	hasher := Hasher{
		Parallel: 1,
		OnDone: func(ctx context.Context, url string, hash []byte, err error) {
			if err == nil {
				t.Fatalf("Expected error in OnDone, got nil")
			}

			if url != ts.URL {
				t.Fatalf("Unexpected URL in callback: %v", url)
			}

			if hash != nil {
				t.Fatalf("Expected nil hash, got %x", hash)
			}

			executed++
		},
	}

	err := hasher.Start(context.Background(), []string{ts.URL})
	if err != nil {
		t.Fatalf("Unexpected error in Start: %v", err)
	}

	if executed != 1 {
		t.Fatalf("Unexpected number of executions: %v (expected 1)", executed)
	}
}

func TestConcurrency(t *testing.T) {

	// idea of the test: try to trigger race detector when testing with -race

	const (
		numberOfURLs = 10240
		parallel     = 128
	)

	var (
		generatedPayloads     = map[string]int{}
		generatedPayloadsSync sync.Mutex
	)

	// create a sample http server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var payload = make([]byte, 2048)
		_, err := rand.Read(payload)
		if err != nil {
			t.Fatalf("rand.Read failed: %v", err)
		}

		payloadHash := fmt.Sprintf("%x", md5.Sum(payload))

		generatedPayloadsSync.Lock()
		generatedPayloads[payloadHash]++
		generatedPayloadsSync.Unlock()

		_, err = w.Write(payload)
		if err != nil {
			t.Fatalf("w.Write failed: %v", err)
		}
	}))
	defer ts.Close()

	hasher := Hasher{
		Parallel: parallel,
		OnDone: func(ctx context.Context, url string, hash []byte, err error) {
			if err != nil {
				t.Fatalf("Unexpected error in OnDone: %v", err)
			}

			gotHash := fmt.Sprintf("%x", hash)

			generatedPayloadsSync.Lock()
			generatedPayloads[gotHash]--
			generatedPayloadsSync.Unlock()
		},
	}

	var urls = make([]string, 0, numberOfURLs)
	for i := 0; i < numberOfURLs; i++ {
		urls = append(urls, fmt.Sprintf("%v/?number=%v", ts.URL, i))
	}

	err := hasher.Start(context.Background(), urls)
	if err != nil {
		t.Fatalf("Unexpected error in Start: %v", err)
	}

	if len(generatedPayloads) != numberOfURLs {
		t.Fatalf("Unexpected number of executions: %v (expected %v)", len(generatedPayloads), numberOfURLs)
	}

	for hash, dt := range generatedPayloads {
		if dt < 0 {
			t.Fatalf("Hash not returned from the server, but reported: %v", hash)
		} else if dt > 0 {
			t.Fatalf("Hash returned from the server, but not reported: %v", hash)
		}
	}
}
