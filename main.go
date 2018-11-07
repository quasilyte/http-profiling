package main

import (
	"flag"
	"log"
	_ "net/http/pprof" // Для live профилировки
	"os"
	"runtime"
	"runtime/pprof"
	"sync"

	"github.com/valyala/fasthttp"
)

func main() {
	withPool := flag.Bool("withPool", false, `whether to use sync.Pool`)
	cpuProfile := flag.String("cpuprofile", "", `write cpu profile to the specified file`)
	memProfile := flag.String("memprofile", "", `write memory profile to the specified file`)

	flag.Parse()

	handler := &httpHandler{
		stop: make(chan bool),
	}
	if *withPool {
		handler.handleUserSearch = handleUserSearchWithPool
	} else {
		handler.handleUserSearch = handleUserSearch
	}

	if *cpuProfile != "" {
		f := mustCreateFile(*cpuProfile)
		if err := pprof.StartCPUProfile(f); err != nil {
			panic(err)
		}
		defer pprof.StopCPUProfile()
	}
	if *memProfile != "" {
		f := mustCreateFile(*memProfile)
		defer writeMemStats(f)
	}

	go func() {
		err := fasthttp.ListenAndServe(":8080", handler.handleRequest)
		log.Printf("listen and serve error: %v", err)
		handler.stop <- true
	}()

	<-handler.stop
}

type userSearchRequest struct {
	name []byte
	city []byte

	limit  int
	offset int

	results []string
}

type httpHandler struct {
	stop chan bool

	handleUserSearch func(*fasthttp.RequestCtx)
}

func (h *httpHandler) handleRequest(ctx *fasthttp.RequestCtx) {
	switch string(ctx.Path()) {
	case "/userSearch":
		h.handleUserSearch(ctx)
	case "/stop":
		h.stop <- true
	default:
		ctx.Error("unknown resource accessed", fasthttp.StatusNotFound)
	}
}

func handleUserSearch(ctx *fasthttp.RequestCtx) {
	req := new(userSearchRequest)
	resp := doUserSearch(req, ctx.URI().QueryArgs())
	ctx.Write(resp)
}

var requestsPool sync.Pool = sync.Pool{
	New: func() interface{} {
		return new(userSearchRequest)
	},
}

func handleUserSearchWithPool(ctx *fasthttp.RequestCtx) {
	req := requestsPool.Get().(*userSearchRequest)
	resp := doUserSearch(req, ctx.URI().QueryArgs())
	ctx.Write(resp)
	requestsPool.Put(req)
}

var responseStub = []byte(`{"organization": "a", "experience": 5}`)
var sink *userSearchRequest

func doUserSearch(req *userSearchRequest, args *fasthttp.Args) []byte {
	req.name = args.Peek("name")
	req.city = args.Peek("city")
	req.limit = args.GetUintOrZero("limit")
	req.offset = args.GetUintOrZero("offset")

	req.results = req.results[:0]
	for i := 0; i < 5; i++ {
		req.results = append(req.results, "result example")
	}

	sink = req

	return responseStub
}

func mustCreateFile(filename string) *os.File {
	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	return f
}

func writeMemStats(f *os.File) {
	// Для более точных результатов.
	runtime.GC()

	if err := pprof.WriteHeapProfile(f); err != nil {
		log.Fatalf("write mem profile: %v", err)
	}

	f.Close()
}
