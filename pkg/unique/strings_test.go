package unique

import (
	"reflect"
	"testing"
)

func TestStrings(t *testing.T) {
	testCases := []struct {
		in  []string
		out []string
	}{
		{
			in:  []string{"a"},
			out: []string{"a"},
		},
		{
			in:  []string{"b", "a"},
			out: []string{"b", "a"},
		},
		{
			in:  []string{"b", "a", "b"},
			out: []string{"b", "a"},
		},
		{
			in:  []string{"c", "c", "c"},
			out: []string{"c"},
		},
	}
	for i, testCase := range testCases {
		if expected, actual := testCase.out, Strings(testCase.in); !reflect.DeepEqual(actual, expected) {
			t.Errorf("[i=%v] Expected result=%+v but actual=%+v", i, expected, actual)
		}
	}
}

func TestStringsSorted(t *testing.T) {
	testCases := []struct {
		in  []string
		out []string
	}{
		{
			in:  []string{"a"},
			out: []string{"a"},
		},
		{
			in:  []string{"b", "a"},
			out: []string{"a", "b"},
		},
		{
			in:  []string{"b", "a", "b"},
			out: []string{"a", "b"},
		},
		{
			in:  []string{"c", "c", "c"},
			out: []string{"c"},
		},
	}
	for i, testCase := range testCases {
		if expected, actual := testCase.out, StringsSorted(testCase.in); !reflect.DeepEqual(actual, expected) {
			t.Errorf("[i=%v] Expected result=%+v but actual=%+v", i, expected, actual)
		}
	}
}
