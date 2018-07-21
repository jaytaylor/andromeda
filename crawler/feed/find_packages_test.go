package feed

import (
	"reflect"
	"testing"
)

func TestFindPackages(t *testing.T) {
	testCases := []struct {
		input    string
		expected []string
	}{
		{
			input:    `jaytaylor.com/andromeda`,
			expected: []string{"jaytaylor.com/andromeda"},
		},
		{
			input:    `.j container/heap jaytaylor.com/andromeda container/ring j.`,
			expected: []string{"jaytaylor.com/andromeda"},
		},
		{
			input:    `.j container/heap <a href="https://jaytaylor.com/andromeda">jaytaylor.com/andromeda</a> container/ring j.`,
			expected: []string{"jaytaylor.com/andromeda"},
		},
		{
			input:    `http://some.blog.com/article?id=99 .j container/heap <a href="https://jaytaylor.com/andromeda">jaytaylor.com/andromeda</a> container/ring j.`,
			expected: []string{"some.blog.com/article", "jaytaylor.com/andromeda"},
		},
		{
			input:    `"selftext": "I'm tried to use [go-cloudflare-scraper](https://github.com/cardigann/go-cloudflare-scraper) but it didn't update more than year`,
			expected: []string{"github.com/cardigann/go-cloudflare-scraper"},
		},
	}

	for i, testCase := range testCases {
		if actual := findPackages(testCase.input); !reflect.DeepEqual(actual, testCase.expected) {
			t.Errorf("[i=%v] Expected returned pkgs(%v)=%+v but acutal(%v)=%v", i, len(testCase.expected), testCase.expected, len(actual), actual)
		}
	}
}
