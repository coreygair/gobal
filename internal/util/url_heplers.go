package util

import "net/url"

// Parses a list of urls from strings, ignoring errors.
func UnsafeParseURLList(urls []string) []*url.URL {
	parsedURLs := make([]*url.URL, len(urls))
	for i, u := range urls {
		p, _ := url.Parse(u)
		parsedURLs[i] = p
	}
	return parsedURLs
}

// Parses a list of urls from strings.
func ParseURLList(urls []string) ([]*url.URL, error) {
	parsedURLs := make([]*url.URL, len(urls))
	for i, u := range urls {
		p, e := url.Parse(u)

		if e != nil {
			return nil, e
		}

		parsedURLs[i] = p
	}
	return parsedURLs, nil
}
