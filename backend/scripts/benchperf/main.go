package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	base := "http://localhost:8080"
	routes := []string{
		"/health",
		"/api/posts",
		"/api/posts?sort=hot",
		"/api/posts?sort=new",
		"/api/posts/1",
		"/api/search?q=test",
		"/api/comments?post_id=1",
	}
	for _, route := range routes {
		avg, status, size := benchGET(base+route, 8)
		fmt.Printf("GET %-28s status=%d avg=%6.1fms avg_body=%dB\n", route, status, avg, size)
	}
}

func benchGET(url string, n int) (float64, int, int) {
	var total time.Duration
	status := 0
	size := 0
	for i := 0; i < n; i++ {
		start := time.Now()
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			return 0, 0, 0
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		total += time.Since(start)
		status = resp.StatusCode
		size += len(body)
	}
	return float64(total.Milliseconds()) / float64(n), status, size / n
}
