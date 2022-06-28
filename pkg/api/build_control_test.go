package api

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_Matches(t *testing.T) {
	list := List{"main", "master", "bugfix/.*"}
	tests := []struct {
		branch string
		match  bool
	}{
		{
			"main",
			true,
		},
		{
			"master",
			true,
		},
		{
			"bugfix/abc",
			true,
		},
		{
			"feature/abc",
			false,
		},
		{
			"main-123",
			false,
		},
		{
			"master-123",
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.branch, func(t *testing.T) {
			assert.Equal(t, test.match, list.Matches(test.branch))
		})
	}
}
