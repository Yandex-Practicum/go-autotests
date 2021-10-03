package random

import (
	"net/url"
	"strings"
)

// URL returns random valid HTTP URL in a form of url.URL
func URL() *url.URL {
	var res url.URL

	// generate SCHEME
	res.Scheme = "http"
	res.Host = Domain(5, 15)

	for i := 0; i < rnd.Intn(4); i++ {
		res.Path += "/" + strings.ToLower(ASCIIString(5, 15))
	}
	return &res
}

// Domain returns random valid domain
func Domain(minLen, maxLen int, zones ...string) string {
	if minLen == 0 {
		minLen = 5
	}
	if maxLen == 0 {
		maxLen = 15
	}

	// generate ZONE
	var zone string
	switch len(zones) {
	case 1:
		zone = zones[0]
	case 0:
		zones = []string{"com", "ru", "net", "biz", "yandex"}
		zone = zones[rnd.Intn(len(zones))]
	default:
		zone = zones[rnd.Intn(len(zones))]
	}

	// generate HOST
	host := strings.ToLower(ASCIIString(minLen, maxLen))
	return host + "." + strings.TrimLeft(zone, ".")
}
