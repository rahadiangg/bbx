package api

import (
	"context"
	"net/url"
	"strconv"
)

// Page is Bamboo's standard paginated envelope.
type Page[T any] struct {
	Results    []T `json:"results"`
	Size       int `json:"size"`
	MaxResult  int `json:"max-result"`
	StartIndex int `json:"start-index"`
	// Some endpoints nest the array under a different key. Those endpoints provide
	// their own typed envelopes and don't use this struct.
}

// PageOpts holds pagination + expansion options shared across list endpoints.
type PageOpts struct {
	StartIndex int
	MaxResults int    // default 25 when 0
	Expand     string // optional comma-separated expansion hints
	Extra      url.Values
}

func (p PageOpts) Values() url.Values {
	v := url.Values{}
	if p.Extra != nil {
		for k, vs := range p.Extra {
			v[k] = vs
		}
	}
	if p.StartIndex > 0 {
		v.Set("start-index", strconv.Itoa(p.StartIndex))
	}
	if p.MaxResults > 0 {
		v.Set("max-results", strconv.Itoa(p.MaxResults))
	}
	if p.Expand != "" {
		v.Set("expand", p.Expand)
	}
	return v
}

// Iterate calls `fetch` repeatedly with incrementing start-index until either
// a page is empty or fewer results than max-results are returned. `limit` caps
// the total items returned (0 = unlimited).
func Iterate[T any](ctx context.Context, opts PageOpts, limit int, fetch func(context.Context, PageOpts) (Page[T], error)) ([]T, error) {
	var all []T
	if opts.MaxResults == 0 {
		opts.MaxResults = 25
	}
	for {
		page, err := fetch(ctx, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, page.Results...)
		if limit > 0 && len(all) >= limit {
			return all[:limit], nil
		}
		if len(page.Results) == 0 || len(page.Results) < opts.MaxResults {
			return all, nil
		}
		opts.StartIndex += len(page.Results)
	}
}
