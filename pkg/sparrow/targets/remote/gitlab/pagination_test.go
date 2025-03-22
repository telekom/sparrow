// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package gitlab

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetNextLink(t *testing.T) {
	type header struct {
		noLinkHeader bool
		key          string
		value        string
	}
	tests := []struct {
		name   string
		header header
		want   string
	}{
		{
			"no link header present",
			header{
				noLinkHeader: true,
			},
			"",
		},
		{
			"no next link in link header present",
			header{
				key:   "link",
				value: "<https://link.first.de>; rel=\"first\", <https://link.last.de>; rel=\"last\"",
			},
			"",
		},
		{
			"link header syntax not valid",
			header{
				key:   "link",
				value: "no link here",
			},
			"",
		},
		{
			"valid next link",
			header{
				key:   "link",
				value: "<https://link.next.de>; rel=\"next\", <https://link.first.de>; rel=\"first\", <https://link.last.de>; rel=\"last\"",
			},
			"https://link.next.de",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testHeader := http.Header{}
			testHeader.Add(tt.header.key, tt.header.value)

			assert.Equal(t, tt.want, getNextLink(testHeader))
		})
	}
}
