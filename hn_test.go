package main

import (
	"context"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
)

func TestTop10HN(t *testing.T) {
	items, err := topNHN(context.TODO(), resty.New().SetBaseURL(baseURL), topN)
	require.NoError(t, err)
	require.Len(t, items, 10)
	t.Log(items)
}
