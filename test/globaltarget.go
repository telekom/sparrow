package test

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/telekom/sparrow/pkg/checks"
)

// Schemes for HTTP and HTTPS
const (
	SchemeHTTP  = "http"
	SchemeHTTPS = "https"
)

const (
	// SparrowLocalName is the name of the local sparrow
	SparrowLocalName = "local.sparrow.telekom.com"
	// SparrowAZ1Name is the name of the AZ1 sparrow
	SparrowAZ1Name = "az1.sparrow.telekom.com"
	// SparrowAZ2Name is the name of the AZ2 sparrow
	SparrowAZ2Name = "az2.sparrow.telekom.com"
)

// SparrowLocal returns a [checks.GlobalTarget] for the local sparrow.
// This is used for testing purposes only.
func SparrowLocal(t testing.TB, tm time.Time) checks.GlobalTarget {
	t.Helper()
	return checks.GlobalTarget{
		URL:      ToURLOrFail(t, fmt.Sprintf("%s://%s", SchemeHTTPS, SparrowLocalName)),
		LastSeen: tm,
	}
}

// SparrowAZ1 returns a [checks.GlobalTarget] for the AZ1 sparrow.
// This is used for testing purposes only.
func SparrowAZ1(t testing.TB, tm time.Time) checks.GlobalTarget {
	t.Helper()
	return checks.GlobalTarget{
		URL:      ToURLOrFail(t, fmt.Sprintf("%s://%s", SchemeHTTPS, SparrowAZ1Name)),
		LastSeen: tm,
	}
}

// SparrowAZ2 returns a [checks.GlobalTarget] for the AZ2 sparrow.
// This is used for testing purposes only.
func SparrowAZ2(t testing.TB, tm time.Time) checks.GlobalTarget {
	t.Helper()
	return checks.GlobalTarget{
		URL:      ToURLOrFail(t, fmt.Sprintf("%s://%s", SchemeHTTPS, SparrowAZ2Name)),
		LastSeen: tm,
	}
}

// ToURLOrFail parses a URI string and returns a URL object.
// It fails the test if the parsing fails.
func ToURLOrFail(t testing.TB, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		t.Fatalf("failed to parse URL %q: %v", s, err)
	}
	return u
}
