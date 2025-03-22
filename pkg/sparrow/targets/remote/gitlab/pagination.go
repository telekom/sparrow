// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package gitlab

import (
	"net/http"
	"strings"
)

const (
	linkHeader = "Link"
	linkNext   = "next"
)

// getNextLink returns the url to the next page of
// a paginated http response provided in the passed response header.
func getNextLink(header http.Header) string {
	link := header.Get(linkHeader)
	if link == "" {
		return ""
	}

	for _, link := range strings.Split(link, ",") {
		linkParts := strings.Split(link, ";")
		if len(linkParts) != 2 {
			continue
		}
		linkType := strings.Trim(strings.Split(linkParts[1], "=")[1], "\"")

		if linkType != linkNext {
			continue
		}
		return strings.Trim(linkParts[0], "< >")
	}
	return ""
}
