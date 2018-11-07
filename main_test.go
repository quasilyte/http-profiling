package main

import (
	"testing"

	"github.com/valyala/fasthttp"
)

var userSearchParams = &fasthttp.Args{}

func init() {
	p := userSearchParams
	p.Add("name", "Gopher")
	p.Add("city", "Kazan")
	p.Add("limit", "34")
	p.Add("offset", "0")
}

func BenchmarkUserSearch(b *testing.B) {

	for i := 0; i < b.N; i++ {
		req := new(userSearchRequest)
		_ = doUserSearch(req, userSearchParams)
	}
}

func BenchmarkUserSearchWithPool(b *testing.B) {
	for i := 0; i < b.N; i++ {
		req := requestsPool.Get().(*userSearchRequest)
		_ = doUserSearch(req, userSearchParams)
		requestsPool.Put(req)
	}
}
