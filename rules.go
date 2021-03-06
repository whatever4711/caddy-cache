package cache

import (
	"github.com/pquerna/cachecontrol/cacheobject"
	"net/http"
	"strings"
	"time"
)

type CacheRule interface {
	matches(*http.Request, int, *http.Header) bool
}

type PathCacheRule struct {
	Path string
}

type HeaderCacheRule struct {
	Header string
	Value  []string
}

/* This rules decide if the request must be cached and are added to handler config if are present in Caddyfile */

func (rule *PathCacheRule) matches(req *http.Request, statusCode int, respHeaders *http.Header) bool {
	return strings.HasPrefix(req.URL.Path, rule.Path)
}

func (rule *HeaderCacheRule) matches(req *http.Request, statusCode int, respHeaders *http.Header) bool {
	headerValue := respHeaders.Get(rule.Header)
	for _, expectedValue := range rule.Value {
		if expectedValue == headerValue {
			return true
		}
	}
	return false
}

func shouldUseCache(req *http.Request) bool {
	// TODO Add more logic like get params, ?nocache=true

	if req.Method != "GET" && req.Method != "HEAD" {
		// Only cache Get and head request
		return false
	}

	// Range responses still not supported
	if req.Header.Get("accept-ranges") != "" {
		return false
	}

	return true
}

func getCacheableStatus(req *http.Request, statusCode int, respHeaders http.Header, config *Config) (bool, time.Time, error) {
	reasonsNotToCache, expiration, err := cacheobject.UsingRequestResponse(req, statusCode, respHeaders, false)

	if err != nil {
		return false, time.Now(), err
	}

	canBeStored := len(reasonsNotToCache) == 0

	if !canBeStored {
		return false, time.Now(), nil
	}

	varyHeaders, ok := respHeaders["Vary"]
	if ok && varyHeaders[0] == "*" {
		return false, time.Now(), nil
	}

	// Sometimes the returned date is 31 Dec 1969
	// So an expiration is given if it is after now
	hasExplicitExpiration := expiration.After(time.Now().UTC())

	if !hasExplicitExpiration {
		// If expiration is not specified use default MaxAge
		expiration = time.Now().UTC().Add(config.DefaultMaxAge)
	}

	anyCacheRulesMatches := false
	for _, rule := range config.CacheRules {
		if rule.matches(req, statusCode, &respHeaders) {
			anyCacheRulesMatches = true
			break
		}
	}

	return anyCacheRulesMatches || hasExplicitExpiration, expiration, nil
}
