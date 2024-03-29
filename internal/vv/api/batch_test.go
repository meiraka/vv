package api

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/meiraka/vv/internal/log"
)

func TestImgBatch(t *testing.T) {
	t.Run("update", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()
		c1 := make(chan struct{}, 1)
		c2 := make(chan struct{}, 1)
		cov1 := newCoverFunc(func(map[string][]string) ([]string, bool) { return []string{""}, true },
			func(ctx context.Context, song map[string][]string) error {
				<-c1
				return nil
			}, func(context.Context, map[string][]string, string) error { return errors.New("must not be called") })
		cov2 := newCoverFunc(func(map[string][]string) ([]string, bool) { return []string{""}, true },
			func(ctx context.Context, song map[string][]string) error {
				<-c2
				return nil
			}, func(context.Context, map[string][]string, string) error { return errors.New("must not be called") })

		batch := newImgBatch([]ImageProvider{cov1, cov2}, log.NewTestLogger(t))
		if err := batch.Update([]map[string][]string{{"file": {"/foo/bar"}}}); err != nil {
			t.Errorf("batch.Update() = %v; want %v", err, nil)
		}
		if err := batch.Update([]map[string][]string{{"file": {"/foo/bar"}}}); err != errAlreadyUpdating {
			t.Errorf("batch.Update() = %v; want %v", err, errAlreadyUpdating)
		}
		if err := batch.Rescan([]map[string][]string{{"file": {"/foo/bar"}}}); err != errAlreadyUpdating {
			t.Errorf("batch.Rescan() = %v; want %v", err, errAlreadyUpdating)
		}
		testEvent(ctx, t, batch.Event(), true, true)
		c1 <- struct{}{}
		c2 <- struct{}{}
		testEvent(ctx, t, batch.Event(), false, true)
		if len(c1) != 0 {
			t.Errorf("cov1.Update is not called: %d", len(c1))
		}
		if len(c2) != 1 {
			t.Errorf("cov2.Update called: %d", len(c2))
		}
		if err := batch.Shutdown(ctx); err != nil {
			t.Errorf("got first batch.Shutdown() = %v; want nil", err)
		}
		testEvent(ctx, t, batch.Event(), false, false)
		if err := batch.Shutdown(ctx); err != nil {
			t.Errorf("got second batch.Shutdown() = %v; want nil", err)
		}
		if err := batch.Update([]map[string][]string{{"file": {"/foo/bar"}}}); err != ErrAlreadyShutdown {
			t.Errorf("shutdown batch.Update() = %v; want %v", err, ErrAlreadyShutdown)
		}
	})
	t.Run("rescan", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()
		c1 := make(chan struct{}, 1)
		c2 := make(chan struct{}, 1)
		cov1 := newCoverFunc(func(map[string][]string) ([]string, bool) { return []string{""}, true },
			func(context.Context, map[string][]string) error { return errors.New("must not be called") },
			func(ctx context.Context, song map[string][]string, _ string) error {
				<-c1
				return nil
			})
		cov2 := newCoverFunc(func(map[string][]string) ([]string, bool) { return []string{""}, true },
			func(context.Context, map[string][]string) error { return errors.New("must not be called") },
			func(ctx context.Context, song map[string][]string, _ string) error {
				<-c2
				return nil
			})

		batch := newImgBatch([]ImageProvider{cov1, cov2}, log.NewTestLogger(t))
		if err := batch.Rescan([]map[string][]string{{"file": {"/foo/bar"}}}); err != nil {
			t.Errorf("batch.Rescan() = %v; want %v", err, nil)
		}
		if err := batch.Rescan([]map[string][]string{{"file": {"/foo/bar"}}}); err != errAlreadyUpdating {
			t.Errorf("batch.Rescan() = %v; want %v", err, errAlreadyUpdating)
		}
		if err := batch.Update([]map[string][]string{{"file": {"/foo/bar"}}}); err != errAlreadyUpdating {
			t.Errorf("batch.Update() = %v; want %v", err, errAlreadyUpdating)
		}
		testEvent(ctx, t, batch.Event(), true, true)
		c1 <- struct{}{}
		c2 <- struct{}{}
		testEvent(ctx, t, batch.Event(), false, true)
		if len(c1) != 0 {
			t.Errorf("cov1.Rescan is not called: %d", len(c1))
		}
		if len(c2) != 1 {
			t.Errorf("cov2.Rescan called: %d", len(c2))
		}
		if err := batch.Shutdown(ctx); err != nil {
			t.Errorf("got first batch.Shutdown() = %v; want nil", err)
		}
		testEvent(ctx, t, batch.Event(), false, false)
		if err := batch.Shutdown(ctx); err != nil {
			t.Errorf("got second batch.Shutdown() = %v; want nil", err)
		}
		if err := batch.Rescan([]map[string][]string{{"file": {"/foo/bar"}}}); err != ErrAlreadyShutdown {
			t.Errorf("shutdown batch.Rescan() = %v; want %v", err, ErrAlreadyShutdown)
		}
	})

	t.Run("shutdown at update", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()
		cov := newCoverFunc(func(map[string][]string) ([]string, bool) { return []string{""}, true },
			func(ctx context.Context, song map[string][]string) error {
				<-ctx.Done()
				return nil
			}, func(context.Context, map[string][]string, string) error { return errors.New("must not be called") })
		batch := newImgBatch([]ImageProvider{cov}, log.NewTestLogger(t))
		if err := batch.Update([]map[string][]string{{"file": {"/foo/bar"}}}); err != nil {
			t.Errorf("batch.Update() = %v; want nil", err)
		}
		sctx, scancel := context.WithTimeout(ctx, time.Millisecond)
		defer scancel()
		if err := batch.Shutdown(sctx); err != nil {
			t.Errorf("got batch.Shutdown() = %v; want nil", err)
		}
	})
	t.Run("shutdown timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()
		cov := newCoverFunc(func(map[string][]string) ([]string, bool) { return []string{""}, true },
			func(context.Context, map[string][]string) error {
				<-ctx.Done()
				return nil
			}, func(context.Context, map[string][]string, string) error { return errors.New("must not be called") })
		batch := newImgBatch([]ImageProvider{cov}, log.NewTestLogger(t))
		if err := batch.Update([]map[string][]string{{"file": {"/foo/bar"}}}); err != nil {
			t.Errorf("batch.Update() = err; want nil")
		}
		testEvent(ctx, t, batch.Event(), true, true)
		sctx, scancel := context.WithTimeout(ctx, time.Millisecond)
		defer scancel()
		if err := batch.Shutdown(sctx); err != context.DeadlineExceeded {
			t.Errorf("got batch.Shutdown() = %v; want %v", err, context.DeadlineExceeded)
		}
	})
	t.Run("empty", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()
		batch := newImgBatch([]ImageProvider{}, log.NewTestLogger(t))
		if err := batch.Update([]map[string][]string{{"file": {"/foo/bar"}}}); err != nil {
			t.Errorf("batch.Update() = %v; want nil", err)
		}
		testEvent(ctx, t, batch.Event(), true, true)
		testEvent(ctx, t, batch.Event(), false, true)
		if urls, ok := batch.GetURLs(map[string][]string{"file": {"/foo/bar"}}); len(urls) != 0 || !ok {
			t.Errorf("GetURLs got %v, %v; want nil, true", urls, ok)
		}
		if err := batch.Shutdown(ctx); err != nil {
			t.Errorf("got batch.Shutdown() = %v; want nil", err)
		}
		testEvent(ctx, t, batch.Event(), false, false)
	})
}

func testEvent(ctx context.Context, t *testing.T, e <-chan bool, want bool, ok bool) {
	t.Helper()
	select {
	case <-ctx.Done():
		t.Errorf("got no Batch.Event(): %v", ctx.Err())
	case got, o := <-e:
		if got != want || o != ok {
			t.Errorf("got Batch.Event() = %v, %v; %v, %v", got, o, want, ok)
		}
	}
}

type coverFunc struct {
	getURLs func(map[string][]string) ([]string, bool)
	update  func(context.Context, map[string][]string) error
	rescan  func(context.Context, map[string][]string, string) error
}

func newCoverFunc(getURLs func(map[string][]string) ([]string, bool), update func(context.Context, map[string][]string) error,
	rescan func(context.Context, map[string][]string, string) error) *coverFunc {
	return &coverFunc{
		getURLs: getURLs,
		update:  update,
		rescan:  rescan,
	}
}

func (c *coverFunc) GetURLs(s map[string][]string) ([]string, bool) {
	return c.getURLs(s)
}

func (c *coverFunc) Update(ctx context.Context, song map[string][]string) error {
	return c.update(ctx, song)
}

func (c *coverFunc) Rescan(ctx context.Context, song map[string][]string, id string) error {
	return c.rescan(ctx, song, id)
}
