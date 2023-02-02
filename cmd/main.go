package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func parseFeedIntoArticles(r io.Reader) ([]string, error) {
	document, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to retrive document: %w", err)
	}

	var articles []string

	document.Find("item").Each(func(i int, s *goquery.Selection) {
		// retrieve title text and trim cdata tag
		title := strings.TrimSuffix(strings.TrimPrefix(
			s.Find("title").Text(), "<![CDATA["), "]]>")

		articles = append(articles, title)
	})

	return articles, nil
}

func getArticlesFromFeed(ctx context.Context, feed string) ([]string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, feed, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	defer response.Body.Close()

	articles, err := parseFeedIntoArticles(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	return articles, nil
}

func checkForUpdates(storage map[string]struct{}, feed string, callback func(article string)) error {
	articles, err := getArticlesFromFeed(context.Background(), feed)
	if err != nil {
		return fmt.Errorf("failed to retrive articles from feed: %w", err)
	}

	// processing articles in reverse order for newest to be first
	for i := len(articles) - 1; i >= 0; i-- {
		article := articles[i]

		// if article already exists in storage then ignore it
		if _, ok := storage[article]; ok {
			// logger.Printf("article '%s' already exists in storage", article)
			continue
		}

		storage[article] = struct{}{}
		callback(article)
	}

	return nil
}

func main() {
	const (
		feed    = "https://habr.com/ru/rss/all/"
		timeout = 30 * time.Second
	)

	var storage = map[string]struct{}{}

	for {
		callback := func(article string) {
			log.Printf("%s\n", article)
		}

		if err := checkForUpdates(storage, feed, callback); err != nil {
			log.Printf("check for updates failed: %v", err)
		}

		time.Sleep(timeout)
	}
}
