package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/cache"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/log"
)

// errLayerContentMismatch indicates that a layer's content, once fully read,
// does not hash to the digest the layer itself reports. This can happen when
// a previously-cached copy of the layer on disk is corrupted or was left
// truncated by an interrupted download (see pullLayerWithRetry).
var errLayerContentMismatch = errors.New("layer content does not match expected digest")

const (
	// pullLayerMaxAttempts is the number of times to attempt to fully download a
	// single layer before giving up. go-containerregistry's own retry support
	// (remote.WithRetryBackoff, etc.) only retries the initial HTTP round trip; it
	// does not retry a read that fails partway through streaming a layer's body.
	// This is the layer of retry that covers that gap.
	pullLayerMaxAttempts = 5
	// pullLayerConcurrency bounds how many layers are pulled at the same time,
	// mirroring crane's own default job concurrency.
	pullLayerConcurrency = 4
	// pullLayerRetryMaxDelay prevents a transient registry failure from causing
	// an unbounded retry delay if the retry budget is increased later.
	pullLayerRetryMaxDelay = 30 * time.Second
	// pullLayerRetryJitter spreads concurrent retries over a small time window.
	pullLayerRetryJitter = 0.2
)

// pullLayerRetryBaseDelay is the initial backoff delay between failed
// attempts to pull a single layer. It's a var (rather than a const) so tests
// can shrink it.
var pullLayerRetryBaseDelay = 2 * time.Second

// pullLayerWait is replaced in tests so they do not need to wait.
var pullLayerWait = time.After

// pullLayers eagerly downloads the full, uncompressed content of every layer in
// img, verifying it against the layer's own digest. Reading through
// cache.Image's lazy layer wrapper (see the caller in ExecuteChecks) is what
// actually persists each layer to the on-disk filesystem cache, keyed by the
// layer's DiffID -- the same value untar's use of mutate.Extract looks up
// later. As a result, once pullLayers returns successfully, extraction can
// proceed entirely against local disk with no further registry interaction,
// and any layer that failed to download -- whether due to a transient network
// error or a corrupted/truncated cache entry -- has already been retried here.
// layerCache must be the same Cache instance img's layers were wrapped with
// (via cache.Image), so a bad on-disk entry can be cleared before retrying.
func pullLayers(ctx context.Context, img v1.Image, layerCache cache.Cache) error {
	logger := logr.FromContextOrDiscard(ctx)

	layers, err := img.Layers()
	if err != nil {
		return fmt.Errorf("failed to list image layers: %w", err)
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(pullLayerConcurrency)

	for _, layer := range layers {
		g.Go(func() error {
			return pullLayerWithRetry(gctx, logger, layer, layerCache)
		})
	}

	return g.Wait()
}

// pullLayerWithRetry fully downloads a single layer's uncompressed content,
// retrying with a backoff if an attempt fails partway through. layerCache is
// the same Cache the layer's image was wrapped with; on any failed attempt --
// a network error or a detected content/digest mismatch -- the corresponding
// on-disk cache entry is cleared before retrying. This matters even for plain
// network errors: cache.Image's cache-miss path tees the remote response
// straight to disk as it's read, so an interrupted download can leave a
// truncated file cached. Left alone, a later read of that file can complete
// without error (a short read isn't necessarily a read *error*), so a retry
// could otherwise silently "succeed" against corrupt, incomplete data.
func pullLayerWithRetry(ctx context.Context, logger logr.Logger, layer v1.Layer, layerCache cache.Cache) error {
	diffID, err := layer.DiffID()
	if err != nil {
		return fmt.Errorf("failed to determine layer diff id: %w", err)
	}

	retryBackoff := wait.Backoff{
		Duration: pullLayerRetryBaseDelay,
		Factor:   2,
		Jitter:   pullLayerRetryJitter,
		Steps:    pullLayerMaxAttempts - 1,
		Cap:      pullLayerRetryMaxDelay,
	}

	var lastErr error
	for attempt := 1; attempt <= pullLayerMaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := pullLayerOnce(layer, diffID); err != nil {
			lastErr = err

			// cache.Image reuses any layer returned by cache.Get
			// (https://github.com/google/go-containerregistry/blob/v0.22.1/pkg/v1/cache/cache.go#L107-L126).
			// The filesystem cache deletes a file only when LayerFromFile reports
			// io.ErrUnexpectedEOF (https://github.com/google/go-containerregistry/blob/v0.22.1/pkg/v1/cache/fs.go#L120-L141),
			// but LayerFromFile only probes compression headers before returning a
			// layer (https://github.com/google/go-containerregistry/blob/v0.22.1/pkg/v1/tarball/layer.go#L170-L235).
			// A truncated raw layer can therefore be a cache hit; delete it so the
			// next cache miss reaches fs.layer.Uncompressed, which uses os.Create
			// to overwrite the file (https://github.com/google/go-containerregistry/blob/v0.22.1/pkg/v1/cache/fs.go#L83-L95).
			if delErr := layerCache.Delete(diffID); delErr != nil && !errors.Is(delErr, cache.ErrNotFound) {
				logger.V(log.DBG).Info("failed to clear cached layer after failed pull attempt",
					"diffID", diffID.String(), "reason", delErr.Error())
				return fmt.Errorf("failed to clear cached layer %s after failed pull attempt: %w", diffID, errors.Join(err, delErr))
			}

			reason := "failed to pull layer"
			if errors.Is(err, errLayerContentMismatch) {
				reason = "cached layer content did not match expected digest; cleared cache entry"
			}

			if attempt == pullLayerMaxAttempts {
				logger.V(log.DBG).Info(reason+", exhausted all attempts",
					"diffID", diffID.String(), "attempt", attempt, "maxAttempts", pullLayerMaxAttempts, "reason", err.Error())
				break
			}

			logger.V(log.DBG).Info(reason+", will retry",
				"diffID", diffID.String(), "attempt", attempt, "maxAttempts", pullLayerMaxAttempts, "reason", err.Error())

			delay := min(retryBackoff.Step(), pullLayerRetryMaxDelay)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-pullLayerWait(delay):
			}
			continue
		}

		return nil
	}

	return fmt.Errorf("failed to pull layer %s after %d attempts: %w", diffID, pullLayerMaxAttempts, lastErr)
}

// pullLayerOnce makes a single attempt to fully read a layer's uncompressed
// content, verifying that it hashes to the layer's own reported diffID.
func pullLayerOnce(layer v1.Layer, diffID v1.Hash) error {
	rc, err := layer.Uncompressed()
	if err != nil {
		return err
	}
	defer rc.Close()

	got, bytesRead, err := v1.SHA256(rc)
	if err != nil {
		return err
	}

	if got != diffID {
		return fmt.Errorf("%w: read %d bytes, computed %s, expected %s", errLayerContentMismatch, bytesRead, got, diffID)
	}

	return nil
}
