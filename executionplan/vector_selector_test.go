package executionplan

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql"
	"github.com/stretchr/testify/require"
)

func TestSelector(t *testing.T) {
	testCases := []struct {
		name     string
		load     string
		start    time.Time
		end      time.Time
		interval time.Duration
		expected []promql.Vector
	}{
		{
			name: "timestamps match with step",
			load: `load 30s
              bar 0 1 10 100 1000`,
			start:    time.Unix(0, 0),
			end:      time.Unix(120, 0),
			interval: 60 * time.Second,
			expected: []promql.Vector{
				[]promql.Sample{
					{Metric: labels.FromStrings("__name__", "bar"), Point: promql.Point{T: 0, V: 0}},
				},
				[]promql.Sample{
					{Metric: labels.FromStrings("__name__", "bar"), Point: promql.Point{T: 60000, V: 10}},
				},
				[]promql.Sample{
					{Metric: labels.FromStrings("__name__", "bar"), Point: promql.Point{T: 120000, V: 1000}},
				},
			},
		},
		{
			name: "timestamps before step",
			load: `load 29s
              bar 0 1 10 100 1000`,
			start:    time.Unix(0, 0),
			end:      time.Unix(120, 0),
			interval: 60 * time.Second,
			expected: []promql.Vector{
				[]promql.Sample{
					{Metric: labels.FromStrings("__name__", "bar"), Point: promql.Point{T: 0, V: 0}},
				},
				[]promql.Sample{
					{Metric: labels.FromStrings("__name__", "bar"), Point: promql.Point{T: 60000, V: 10}},
				},
				[]promql.Sample{
					{Metric: labels.FromStrings("__name__", "bar"), Point: promql.Point{T: 120000, V: 1000}},
				},
			},
		},
		{
			name: "timestamps after step",
			load: `load 31s
              bar 0 1 10 100 1000`,
			start:    time.Unix(0, 0),
			end:      time.Unix(120, 0),
			interval: 60 * time.Second,
			expected: []promql.Vector{
				[]promql.Sample{
					{Metric: labels.FromStrings("__name__", "bar"), Point: promql.Point{T: 0, V: 0}},
				},
				[]promql.Sample{
					{Metric: labels.FromStrings("__name__", "bar"), Point: promql.Point{T: 60000, V: 1}},
				},
				[]promql.Sample{
					{Metric: labels.FromStrings("__name__", "bar"), Point: promql.Point{T: 120000, V: 100}},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			test, err := promql.NewTest(t, tc.load)
			require.NoError(t, err)
			defer test.Close()

			err = test.Run()
			require.NoError(t, err)

			nameMatcher, err := labels.NewMatcher(labels.MatchEqual, labels.MetricName, "bar")
			require.NoError(t, err)
			matchers := []*labels.Matcher{nameMatcher}

			series := newSeriesFilter(test.Storage(), tc.start, tc.end, matchers)
			selector := NewVectorSelector(series, nil, tc.start, tc.end, tc.interval, 0, 1)
			result := make([]promql.Vector, 0)
			for {
				r, err := selector.Next(context.Background())
				require.NoError(t, err)
				if r == nil {
					break
				}
				result = append(result, r)
			}

			require.Equal(t, tc.expected, result)
		})
	}
}
