package api

import (
	"context"
	"errors"
	"strconv"
	"testing"
)

func TestPageOptsValues(t *testing.T) {
	t.Parallel()
	o := PageOpts{StartIndex: 10, MaxResults: 50, Expand: "branches"}
	v := o.Values()
	if v.Get("start-index") != "10" {
		t.Errorf("start-index = %q", v.Get("start-index"))
	}
	if v.Get("max-results") != "50" {
		t.Errorf("max-results = %q", v.Get("max-results"))
	}
	if v.Get("expand") != "branches" {
		t.Errorf("expand = %q", v.Get("expand"))
	}
}

func TestPageOptsValuesZerosOmitted(t *testing.T) {
	t.Parallel()
	v := PageOpts{}.Values()
	if v.Get("start-index") != "" || v.Get("max-results") != "" || v.Get("expand") != "" {
		t.Fatalf("unexpected values: %v", v)
	}
}

func TestPageOptsExtraIncluded(t *testing.T) {
	t.Parallel()
	o := PageOpts{Extra: map[string][]string{"includeAllStates": {"true"}}}
	v := o.Values()
	if v.Get("includeAllStates") != "true" {
		t.Fatalf("got %v", v)
	}
}

func TestIterateStopsOnShortPage(t *testing.T) {
	t.Parallel()
	calls := 0
	items, err := Iterate[int](context.Background(), PageOpts{MaxResults: 5}, 0,
		func(_ context.Context, o PageOpts) (Page[int], error) {
			calls++
			switch o.StartIndex {
			case 0:
				return Page[int]{Results: []int{1, 2, 3, 4, 5}}, nil
			case 5:
				return Page[int]{Results: []int{6, 7}}, nil // short page -> stop
			}
			t.Fatalf("unexpected start-index %d", o.StartIndex)
			return Page[int]{}, nil
		})
	if err != nil {
		t.Fatalf("Iterate: %v", err)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
	if got := strconv.Itoa(len(items)); got != "7" {
		t.Errorf("len = %s, items = %v", got, items)
	}
}

func TestIterateStopsOnEmpty(t *testing.T) {
	t.Parallel()
	calls := 0
	items, err := Iterate[int](context.Background(), PageOpts{MaxResults: 3}, 0,
		func(_ context.Context, o PageOpts) (Page[int], error) {
			calls++
			if o.StartIndex == 0 {
				return Page[int]{Results: []int{1, 2, 3}}, nil
			}
			return Page[int]{Results: nil}, nil
		})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || len(items) != 3 {
		t.Fatalf("calls=%d items=%v", calls, items)
	}
}

func TestIterateLimit(t *testing.T) {
	t.Parallel()
	items, err := Iterate[int](context.Background(), PageOpts{MaxResults: 5}, 4,
		func(_ context.Context, o PageOpts) (Page[int], error) {
			return Page[int]{Results: []int{o.StartIndex, o.StartIndex + 1, o.StartIndex + 2, o.StartIndex + 3, o.StartIndex + 4}}, nil
		})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 4 {
		t.Fatalf("len = %d, want 4", len(items))
	}
}

func TestIterateDefaultMaxResults(t *testing.T) {
	t.Parallel()
	var seenMax int
	_, _ = Iterate[int](context.Background(), PageOpts{}, 0,
		func(_ context.Context, o PageOpts) (Page[int], error) {
			seenMax = o.MaxResults
			return Page[int]{}, nil
		})
	if seenMax != 25 {
		t.Fatalf("default MaxResults = %d, want 25", seenMax)
	}
}

func TestIteratePropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	_, err := Iterate[int](context.Background(), PageOpts{MaxResults: 5}, 0,
		func(_ context.Context, _ PageOpts) (Page[int], error) {
			return Page[int]{}, want
		})
	if !errors.Is(err, want) {
		t.Fatalf("err = %v", err)
	}
}
