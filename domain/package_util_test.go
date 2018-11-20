package domain

import (
	"testing"
)

func TestPackagePathFromURL(t *testing.T) {
	type tuple struct {
		pkgPath  string
		redirect bool
	}
	testCases := []struct {
		input    string
		expected tuple
	}{
		{
			input:    "",
			expected: tuple{"", false},
		},
		{
			input:    "/",
			expected: tuple{"", false},
		},
		{
			input:    "///",
			expected: tuple{"", false},
		},
		{
			input:    "github.com/jaytaylor/andromeda",
			expected: tuple{"github.com/jaytaylor/andromeda", false},
		},
		{
			input:    "github.com/jaytaylor/andromeda/crawler",
			expected: tuple{"github.com/jaytaylor/andromeda/crawler", false},
		},
		{
			input:    "github.com/jaytaylor/andromeda/crawler/feed",
			expected: tuple{"github.com/jaytaylor/andromeda/crawler/feed", false},
		},
		{
			input:    "github.com/jaytaylor/andromeda.git",
			expected: tuple{"github.com/jaytaylor/andromeda", true},
		},
		{
			input:    "git@github.com:jaytaylor/andromeda",
			expected: tuple{"github.com/jaytaylor/andromeda", true},
		},
		{
			input:    "git@github.com:jaytaylor/andromeda.git",
			expected: tuple{"github.com/jaytaylor/andromeda", true},
		},
		{
			input:    "http://github.com/jaytaylor/andromeda",
			expected: tuple{"github.com/jaytaylor/andromeda", true},
		},
		{
			input:    "https://github.com/jaytaylor/andromeda",
			expected: tuple{"github.com/jaytaylor/andromeda", true},
		},
		{
			input:    "https://github.com/jaytaylor/andromeda.git",
			expected: tuple{"github.com/jaytaylor/andromeda", true},
		},
		{
			input:    "https://github.com/jaytaylor/andromeda/",
			expected: tuple{"github.com/jaytaylor/andromeda", true},
		},
	}
	for i, testCase := range testCases {
		pkgPath, redirect := PackagePathFromURL(testCase.input)
		if expected, actual := testCase.expected.pkgPath, pkgPath; actual != expected {
			t.Errorf("[i=%v] For input=%q: Expected clean=%q but actual=%q", i, testCase.input, expected, actual)
		}
		if expected, actual := testCase.expected.redirect, redirect; actual != expected {
			t.Errorf("[i=%v] For input=%q: Expected redirect=%v but actual=%v", i, testCase.input, expected, actual)
		}
	}
}
