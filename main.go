package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
)

type RouteCtx struct {
	w   http.ResponseWriter
	req *http.Request
}

type InMemoryResponseWriter struct {
	content []byte
}

func (w *InMemoryResponseWriter) Header() http.Header {
	panic("implement me")
}

func (w *InMemoryResponseWriter) Write(bytes []byte) (int, error) {
	w.content = append(w.content, bytes...)
	return len(bytes), nil
}

func (w *InMemoryResponseWriter) WriteHeader(statusCode int) {
	panic("implement me")
}

func (c *RouteCtx) View(path string, dataHandler RouteDataHandler) error {
	dataWriter := &InMemoryResponseWriter{content: []byte{}}
	dataContext := &RouteCtx{
		w:   dataWriter,
		req: c.req,
	}
	if err := dataHandler(dataContext); err != nil {
		return err
	}
	_, err := fmt.Fprintf(c.w, "<!DOCTYPE html><html><head><title>Hello</title></head><body><h1>%s</h1><pre>%s</pre></body></html>", path, string(dataWriter.content))
	return err
}

func (c *RouteCtx) Param(name string) string {
	params := mux.Vars(c.req)
	return params[name]
}

func (c *RouteCtx) Json(model interface{}) error {
	return json.NewEncoder(c.w).Encode(model)
}

type RouteHandler func(ctx *RouteCtx) error
type RouteDataHandler func(ctx *RouteCtx) error

type NexusRouter struct {
	mux *mux.Router
}

func NewRouter() *NexusRouter {
	return &NexusRouter{
		mux: mux.NewRouter(),
	}
}

func (r *NexusRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

func (r *NexusRouter) Route(path string, handler RouteHandler) {
	r.mux.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		ctx := &RouteCtx{
			w:   w,
			req: req,
		}
		if err := handler(ctx); err != nil {
			log.Println(err)
		}
	})
}

type IndexViewModel struct {
	Counter int32 `json:"counter"`
}

func main() {
	r := NewRouter()
	r.Route("/{id}", func(ctx *RouteCtx) error {
		return ctx.View("/", func(dataCtx *RouteCtx) error {
			idStr := dataCtx.Param("id")
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return err
			}
			return dataCtx.Json(IndexViewModel{
				Counter: int32(id),
			})
		})
	})
	r.Route("/", func(ctx *RouteCtx) error {
		return ctx.View("/", func(dataCtx *RouteCtx) error {
			return dataCtx.Json(IndexViewModel{
				Counter: 0,
			})
		})
	})
	if err := http.ListenAndServe(":3000", r); err != nil {
		panic(err)
	}
}
