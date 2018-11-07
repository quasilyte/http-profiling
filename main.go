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
	handler := &httpHandler{
		stop: make(chan bool),
	}

	withPool := flag.Bool("withPool", false,
		`whether to use sync.Pool`)
	flag.StringVar(&handler.flags.cpuprofile, "cpuprofile", "",
		`write cpu profile to the specified file`)
	flag.StringVar(&handler.flags.memprofile, "memprofile", "",
		`write memory profile to the specified file`)
	flag.Parse()

	if *withPool {
		handler.handleUserSearch = handleUserSearchWithPool
	} else {
		handler.handleUserSearch = handleUserSearch
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

	flags struct {
		cpuprofile string
		memprofile string
	}
	memprofFile *os.File

	handleUserSearch func(*fasthttp.RequestCtx)
}

func (h *httpHandler) handleRequest(ctx *fasthttp.RequestCtx) {
	switch string(ctx.Path()) {
	case "/userSearch":
		h.handleUserSearch(ctx)
	case "/startProfiling":
		h.startProfiling()
	case "/stopProfiling":
		h.stopProfiling()
	case "/stop":
		h.stop <- true
	default:
		ctx.Error("unknown resource accessed", fasthttp.StatusNotFound)
	}
}

func (h *httpHandler) startProfiling() {
	// Отладочное логгирование должно идти до профилирования.
	if h.flags.cpuprofile != "" {
		log.Printf("collecting CPU profile to %v", h.flags.cpuprofile)
	}
	if h.flags.memprofile != "" {
		log.Printf("collecting mem profile to %v", h.flags.memprofile)
	}

	// CPU профилирование.
	if h.flags.cpuprofile != "" {
		cpuprofFile := mustCreateFile(h.flags.cpuprofile)
		if err := pprof.StartCPUProfile(cpuprofFile); err != nil {
			log.Fatalf("failed to start cpu profiling", fasthttp.StatusInternalServerError)
		}
	}

	// Heap профилирование.
	if h.flags.memprofile != "" {
		h.memprofFile = mustCreateFile(h.flags.memprofile)
	}
}

func (h *httpHandler) stopProfiling() {
	pprof.StopCPUProfile()
	writeMemStats(h.memprofFile)
	h.memprofFile.Close()

	if h.flags.cpuprofile != "" {
		log.Printf("written CPU profile to %v", h.flags.cpuprofile)
	}
	if h.flags.memprofile != "" {
		log.Printf("written mem profile to %v", h.flags.memprofile)
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
