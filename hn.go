package main

import (
	"context"
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/sourcegraph/conc/pool"
	"github.com/tidwall/gjson"
)

const (
	baseURL = "https://hacker-news.firebaseio.com/v0"
	topURI  = "/topstories.json"
	itemURI = "/item/{id}.json"
	topN    = 10
)

type Item struct {
	ID        int64
	Title     string
	URL       string
	RawURL    string
	Score     int64
	Timestamp int64
}

func topNHN(ctx context.Context, client *resty.Client, topN int) ([]Item, error) {
	res, err := client.R().Get(topURI)
	if err != nil {
		return nil, err
	}
	obj := gjson.ParseBytes(res.Body())
	itemIDs := make([]int64, 0, topN)
	var i int
	obj.ForEach(func(key, value gjson.Result) bool {
		itemIDs = append(itemIDs, value.Int())
		i++
		return i < topN
	})
	wp := pool.NewWithResults[Item]().WithMaxGoroutines(3).WithContext(ctx)
	for i := range itemIDs {
		itemID := itemIDs[i]
		wp.Go(func(ctx context.Context) (Item, error) {
			res, err := client.R().
				SetContext(ctx).
				SetPathParam("id", fmt.Sprintf("%d", itemID)).
				Get(itemURI)
			if err != nil {
				return Item{}, err
			}
			obj := gjson.ParseBytes(res.Body())
			return Item{
				ID:        itemID,
				Title:     obj.Get("title").String(),
				URL:       fmt.Sprintf("https://news.ycombinator.com/item?id=%d", itemID),
				RawURL:    obj.Get("url").String(),
				Score:     obj.Get("score").Int(),
				Timestamp: obj.Get("time").Int(),
			}, err
		})
	}
	items, err := wp.Wait()
	return items, err
}
