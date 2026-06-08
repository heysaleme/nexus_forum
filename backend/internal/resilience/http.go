package resilience

import (
	"net/http"
	"net/url"
)

// PostForm executes an HTTP POST form through the named circuit breaker.
func PostForm(name, endpoint string, data url.Values) (*http.Response, error) {
	result, err := Execute(name, func() (interface{}, error) {
		return http.PostForm(endpoint, data)
	})
	if err != nil {
		return nil, err
	}
	return result.(*http.Response), nil
}

// Do executes an HTTP request through the named circuit breaker.
func Do(name string, req *http.Request) (*http.Response, error) {
	result, err := Execute(name, func() (interface{}, error) {
		return http.DefaultClient.Do(req)
	})
	if err != nil {
		return nil, err
	}
	return result.(*http.Response), nil
}
