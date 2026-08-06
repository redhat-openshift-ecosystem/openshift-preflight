package engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/cache"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/static"
	"github.com/google/go-containerregistry/pkg/v1/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// fakeLayer wraps a real, static v1.Layer so that Digest/DiffID/Size/MediaType
// behave normally, while allowing Uncompressed to be made to fail a
// configurable number of times (and/or block) before delegating to the
// wrapped layer's real behavior.
type fakeLayer struct {
	v1.Layer

	mu      sync.Mutex
	calls   int
	failN   int
	blockFn func()
}

func newFakeLayer(content []byte, failN int) *fakeLayer {
	return &fakeLayer{
		Layer: static.NewLayer(content, types.DockerLayer),
		failN: failN,
	}
}

func (f *fakeLayer) Uncompressed() (io.ReadCloser, error) {
	f.mu.Lock()
	f.calls++
	call := f.calls
	block := f.blockFn
	f.mu.Unlock()

	if block != nil {
		block()
	}

	if call <= f.failN {
		return nil, fmt.Errorf("simulated failure on attempt %d", call)
	}
	return f.Layer.Uncompressed()
}

func (f *fakeLayer) Calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// erroringLayersImage wraps a real v1.Image but fails when its layers are
// listed, simulating a malformed or unreadable manifest.
type erroringLayersImage struct {
	v1.Image
}

func (erroringLayersImage) Layers() ([]v1.Layer, error) {
	return nil, errors.New("simulated layers listing failure")
}

// diffIDErrLayer wraps a real v1.Layer but fails to report its DiffID,
// simulating a layer with corrupt or unreadable metadata.
type diffIDErrLayer struct {
	v1.Layer
}

func (diffIDErrLayer) DiffID() (v1.Hash, error) {
	return v1.Hash{}, errors.New("simulated diffID failure")
}

// flakyReader yields a fixed amount of data successfully and then fails,
// simulating a connection that drops partway through streaming a layer's
// body after the request itself already succeeded.
type flakyReader struct {
	data []byte
	pos  int
}

func (r *flakyReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, errors.New("simulated connection reset mid-stream")
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *flakyReader) Close() error { return nil }

// midStreamFailLayer wraps a real v1.Layer, but its Uncompressed reader opens
// successfully and then fails partway through being read.
type midStreamFailLayer struct {
	v1.Layer
}

func (midStreamFailLayer) Uncompressed() (io.ReadCloser, error) {
	return &flakyReader{data: []byte("partial-layer-content")}, nil
}

// corruptThenCleanLayer wraps a real, static v1.Layer -- which provides the
// correct Digest/DiffID/Size/MediaType for goodContent -- but serves
// different, non-matching bytes from Uncompressed for the first corruptN
// calls before falling back to the real, correct content. This simulates a
// cache entry that was corrupted (or left truncated by an earlier interrupted
// download) before eventually being cleared and re-fetched cleanly.
type corruptThenCleanLayer struct {
	v1.Layer

	mu             sync.Mutex
	calls          int
	corruptN       int
	corruptContent []byte
}

func newCorruptThenCleanLayer(goodContent, corruptContent []byte, corruptN int) *corruptThenCleanLayer {
	return &corruptThenCleanLayer{
		Layer:          static.NewLayer(goodContent, types.DockerLayer),
		corruptN:       corruptN,
		corruptContent: corruptContent,
	}
}

func (l *corruptThenCleanLayer) Uncompressed() (io.ReadCloser, error) {
	l.mu.Lock()
	l.calls++
	call := l.calls
	l.mu.Unlock()

	if call <= l.corruptN {
		return io.NopCloser(bytes.NewReader(l.corruptContent)), nil
	}
	return l.Layer.Uncompressed()
}

func (l *corruptThenCleanLayer) Calls() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.calls
}

// truncatingRemoteLayer wraps a real, static v1.Layer -- which provides the
// correct Digest/DiffID/Size/MediaType for goodContent -- but simulates a
// remote layer whose network stream is interrupted partway through the very
// first read (as if a connection dropped mid-download), yielding only
// partial bytes before failing. Subsequent reads succeed with the full,
// correct content, as a genuine retried fetch would.
type truncatingRemoteLayer struct {
	v1.Layer

	mu      sync.Mutex
	calls   int
	partial []byte
}

func newTruncatingRemoteLayer(goodContent, partial []byte) *truncatingRemoteLayer {
	return &truncatingRemoteLayer{
		Layer:   static.NewLayer(goodContent, types.DockerLayer),
		partial: partial,
	}
}

func (l *truncatingRemoteLayer) Uncompressed() (io.ReadCloser, error) {
	l.mu.Lock()
	l.calls++
	call := l.calls
	l.mu.Unlock()

	if call == 1 {
		return &flakyReader{data: l.partial}, nil
	}
	return l.Layer.Uncompressed()
}

func (l *truncatingRemoteLayer) Calls() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.calls
}

// fakeCache is a minimal, in-memory cache.Cache used to observe which layers
// get deleted from the cache after a failed pull attempt, without touching
// disk. Get always reports a cache miss, since these tests exercise
// pullLayers/pullLayerWithRetry directly against plain (non-cache-wrapped)
// layers; only Delete tracking matters here.
type fakeCache struct {
	mu        sync.Mutex
	deleted   []v1.Hash
	deleteErr error
}

func (c *fakeCache) Put(l v1.Layer) (v1.Layer, error) { return l, nil }

func (c *fakeCache) Get(v1.Hash) (v1.Layer, error) { return nil, cache.ErrNotFound }

func (c *fakeCache) Delete(h v1.Hash) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deleted = append(c.deleted, h)
	return c.deleteErr
}

func (c *fakeCache) Deleted() []v1.Hash {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.deleted)
}

var _ = Describe("pullLayers", func() {
	var origDelay time.Duration

	BeforeEach(func() {
		origDelay = pullLayerRetryBaseDelay
		pullLayerRetryBaseDelay = time.Millisecond
		DeferCleanup(func() { pullLayerRetryBaseDelay = origDelay })
	})

	It("succeeds on the first attempt when the layer downloads cleanly", func() {
		layer := newFakeLayer([]byte("hello"), 0)
		img, err := mutate.AppendLayers(empty.Image, layer)
		Expect(err).ToNot(HaveOccurred())

		Expect(pullLayers(context.Background(), img, &fakeCache{})).To(Succeed())
		Expect(layer.Calls()).To(Equal(1))
	})

	It("retries a layer that fails transiently and eventually succeeds", func() {
		layer := newFakeLayer([]byte("hello"), pullLayerMaxAttempts-1)
		img, err := mutate.AppendLayers(empty.Image, layer)
		Expect(err).ToNot(HaveOccurred())

		Expect(pullLayers(context.Background(), img, &fakeCache{})).To(Succeed())
		Expect(layer.Calls()).To(Equal(pullLayerMaxAttempts))
	})

	It("returns a wrapped error once all attempts are exhausted", func() {
		layer := newFakeLayer([]byte("hello"), pullLayerMaxAttempts+5)
		img, err := mutate.AppendLayers(empty.Image, layer)
		Expect(err).ToNot(HaveOccurred())

		err = pullLayers(context.Background(), img, &fakeCache{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to pull layer"))
		Expect(layer.Calls()).To(Equal(pullLayerMaxAttempts))
	})

	It("bounds concurrency across many layers", func() {
		const numLayers = pullLayerConcurrency * 3

		var (
			mu         sync.Mutex
			active     int
			maxActive  int
			releaseAll = make(chan struct{})
		)

		layers := make([]v1.Layer, 0, numLayers)
		for i := 0; i < numLayers; i++ {
			l := newFakeLayer([]byte(fmt.Sprintf("layer-%d", i)), 0)
			l.blockFn = func() {
				mu.Lock()
				active++
				if active > maxActive {
					maxActive = active
				}
				mu.Unlock()

				<-releaseAll

				mu.Lock()
				active--
				mu.Unlock()
			}
			layers = append(layers, l)
		}

		img, err := mutate.AppendLayers(empty.Image, layers...)
		Expect(err).ToNot(HaveOccurred())

		done := make(chan error, 1)
		go func() {
			done <- pullLayers(context.Background(), img, &fakeCache{})
		}()

		// Wait until the pool has filled up to its configured limit.
		Eventually(func() int {
			mu.Lock()
			defer mu.Unlock()
			return active
		}).Should(Equal(pullLayerConcurrency))

		// It should never exceed that limit while more layers remain queued.
		Consistently(func() int {
			mu.Lock()
			defer mu.Unlock()
			return maxActive
		}).Should(BeNumerically("<=", pullLayerConcurrency))

		close(releaseAll)
		Eventually(done).Should(Receive(BeNil()))
		Expect(maxActive).To(Equal(pullLayerConcurrency))
	})

	It("stops promptly without retrying when the context is already cancelled", func() {
		layer := newFakeLayer([]byte("hello"), pullLayerMaxAttempts+5)
		img, err := mutate.AppendLayers(empty.Image, layer)
		Expect(err).ToNot(HaveOccurred())

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err = pullLayers(ctx, img, &fakeCache{})
		Expect(err).To(MatchError(context.Canceled))
		// The context is checked before the first attempt, so the layer
		// should never actually be read -- distinguishing prompt
		// cancellation from exhausting retries against a failing layer.
		Expect(layer.Calls()).To(Equal(0))
	})

	It("returns a wrapped error when listing an image's layers fails", func() {
		err := pullLayers(context.Background(), erroringLayersImage{Image: empty.Image}, &fakeCache{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to list image layers"))
	})

	It("returns an error when a layer's stream fails partway through", func() {
		layer := midStreamFailLayer{Layer: static.NewLayer([]byte("hello"), types.DockerLayer)}
		diffID, err := layer.DiffID()
		Expect(err).ToNot(HaveOccurred())

		err = pullLayerOnce(layer, diffID)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("simulated connection reset mid-stream"))
	})

	It("returns a wrapped error when a layer's DiffID cannot be determined", func() {
		layer := diffIDErrLayer{Layer: static.NewLayer([]byte("hello"), types.DockerLayer)}
		err := pullLayerWithRetry(context.Background(), logr.Discard(), layer, &fakeCache{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to determine layer diff id"))
	})

	It("stops waiting and returns promptly when the context is cancelled during backoff", func() {
		pullLayerRetryBaseDelay = 200 * time.Millisecond
		layer := newFakeLayer([]byte("hello"), pullLayerMaxAttempts+5)

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(20 * time.Millisecond)
			cancel()
		}()

		err := pullLayerWithRetry(ctx, logr.Discard(), layer, &fakeCache{})
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, context.Canceled)).To(BeTrue())
		Expect(layer.Calls()).To(Equal(1))
	})

	It("clears the cached entry and retries when a layer's content does not match its digest", func() {
		good := []byte("the-real-uncorrupted-layer-content")
		bad := []byte("corrupted-bytes-of-a-different-length")
		layer := newCorruptThenCleanLayer(good, bad, 1)

		diffID, err := layer.DiffID()
		Expect(err).ToNot(HaveOccurred())

		fc := &fakeCache{}
		Expect(pullLayerWithRetry(context.Background(), logr.Discard(), layer, fc)).To(Succeed())
		Expect(layer.Calls()).To(Equal(2))
		Expect(fc.Deleted()).To(ConsistOf(diffID))
	})

	It("returns a wrapped mismatch error and clears the cache on every attempt when corruption persists", func() {
		good := []byte("the-real-uncorrupted-layer-content")
		bad := []byte("always-corrupted")
		layer := newCorruptThenCleanLayer(good, bad, pullLayerMaxAttempts+5)

		diffID, err := layer.DiffID()
		Expect(err).ToNot(HaveOccurred())

		fc := &fakeCache{}
		err = pullLayerWithRetry(context.Background(), logr.Discard(), layer, fc)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, errLayerContentMismatch)).To(BeTrue())

		deleted := fc.Deleted()
		Expect(deleted).To(HaveLen(pullLayerMaxAttempts))
		for _, h := range deleted {
			Expect(h).To(Equal(diffID))
		}
	})

	It("logs but does not fail the retry when clearing the cache entry itself errors", func() {
		layer := newFakeLayer([]byte("hello"), 1)
		fc := &fakeCache{deleteErr: errors.New("simulated disk error clearing cache entry")}

		Expect(pullLayerWithRetry(context.Background(), logr.Discard(), layer, fc)).To(Succeed())
		Expect(fc.Deleted()).To(HaveLen(1))
	})

	It("clears a truncated on-disk cache entry left by an interrupted download and re-fetches cleanly", func() {
		tmpDir, err := os.MkdirTemp("", "pull-layers-cache-test-*")
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(os.RemoveAll, tmpDir)

		// Large enough that "half of it" is a meaningfully truncated,
		// non-matching prefix rather than a coincidentally-valid tarball.
		good := []byte(strings.Repeat("real-layer-content-", 1000))
		partial := good[:len(good)/2]

		remoteLayer := newTruncatingRemoteLayer(good, partial)
		diffID, err := remoteLayer.DiffID()
		Expect(err).ToNot(HaveOccurred())

		realCache := cache.NewFilesystemCache(tmpDir)
		img, err := mutate.AppendLayers(empty.Image, remoteLayer)
		Expect(err).ToNot(HaveOccurred())
		cachedImg := cache.Image(img, realCache)

		// Without pullLayers explicitly deleting the cache entry after the
		// first, interrupted attempt, go-containerregistry's own filesystem
		// cache would happily hand back that truncated file on the next
		// read (a short plain-file read isn't an io.ErrUnexpectedEOF), so
		// this only succeeds if the corrupted entry was actually cleared.
		Expect(pullLayers(context.Background(), cachedImg, realCache)).To(Succeed())
		Expect(remoteLayer.Calls()).To(Equal(2))

		cachedLayer, err := realCache.Get(diffID)
		Expect(err).ToNot(HaveOccurred())
		rc, err := cachedLayer.Uncompressed()
		Expect(err).ToNot(HaveOccurred())
		defer rc.Close()

		gotBytes, err := io.ReadAll(rc)
		Expect(err).ToNot(HaveOccurred())
		Expect(gotBytes).To(Equal(good))
	})
})
