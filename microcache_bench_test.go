package microcache

import (
	"net/http"
	"strconv"
	"testing"
	"time"
)

func BenchmarkHits(b *testing.B) {
	cache := New(Config{
		TTL:    30 * time.Second,
		Driver: NewDriverLRU(10),
	})
	defer cache.Stop()
	handler := cache.Middleware(http.HandlerFunc(successHandler))
	r, _ := http.NewRequest("GET", "/", nil)
	w := &noopWriter{http.Header{}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(w, r)
	}
}

func BenchmarkNocache(b *testing.B) {
	cache := New(Config{
		Nocache: true,
		Driver:  NewDriverLRU(10),
	})
	defer cache.Stop()
	handler := cache.Middleware(http.HandlerFunc(successHandler))
	r, _ := http.NewRequest("GET", "/", nil)
	w := &noopWriter{http.Header{}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(w, r)
	}
}

func BenchmarkMisses(b *testing.B) {
	cache := New(Config{
		TTL:    30 * time.Second,
		Driver: NewDriverLRU(10),
	})
	defer cache.Stop()
	handler := cache.Middleware(http.HandlerFunc(successHandler))
	r, _ := http.NewRequest("GET", "/", nil)
	w := &noopWriter{http.Header{}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.URL.Path = "/" + strconv.Itoa(i)
		handler.ServeHTTP(w, r)
	}
}

func BenchmarkCompression1kHits(b *testing.B) {
	cache := New(Config{
		TTL:        30 * time.Second,
		Driver:     NewDriverLRU(10),
		Compressor: CompressorSnappy{},
	})
	defer cache.Stop()
	handler := cache.Middleware(http.HandlerFunc(success1kHandler))
	r, _ := http.NewRequest("GET", "/", nil)
	w := &noopWriter{http.Header{}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(w, r)
	}
}

func BenchmarkCompression1kNocache(b *testing.B) {
	cache := New(Config{
		Nocache:    true,
		Driver:     NewDriverLRU(10),
		Compressor: CompressorSnappy{},
	})
	defer cache.Stop()
	handler := cache.Middleware(http.HandlerFunc(success1kHandler))
	r, _ := http.NewRequest("GET", "/", nil)
	w := &noopWriter{http.Header{}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(w, r)
	}
}

func BenchmarkCompression1kMisses(b *testing.B) {
	cache := New(Config{
		TTL:        30 * time.Second,
		Driver:     NewDriverLRU(10),
		Compressor: CompressorSnappy{},
	})
	defer cache.Stop()
	handler := cache.Middleware(http.HandlerFunc(success1kHandler))
	r, _ := http.NewRequest("GET", "/", nil)
	w := &noopWriter{http.Header{}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.URL.Path = "/" + strconv.Itoa(i)
		handler.ServeHTTP(w, r)
	}
}

func BenchmarkParallelCompression1kHits(b *testing.B) {
	cache := New(Config{
		TTL:        30 * time.Second,
		Driver:     NewDriverLRU(10),
		Compressor: CompressorSnappy{},
	})
	defer cache.Stop()
	handler := cache.Middleware(http.HandlerFunc(success1kHandler))
	r, _ := http.NewRequest("GET", "/", nil)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		w := &noopWriter{http.Header{}}
		for i := 0; pb.Next(); i++ {
			handler.ServeHTTP(w, r)
		}
	})
}

func BenchmarkParallelCompression1kNocache(b *testing.B) {
	cache := New(Config{
		Nocache:    true,
		Driver:     NewDriverLRU(10),
		Compressor: CompressorSnappy{},
	})
	defer cache.Stop()
	handler := cache.Middleware(http.HandlerFunc(success1kHandler))
	r, _ := http.NewRequest("GET", "/", nil)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		w := &noopWriter{http.Header{}}
		for i := 0; pb.Next(); i++ {
			handler.ServeHTTP(w, r)
		}
	})
}

func BenchmarkParallelCompression1kMisses(b *testing.B) {
	cache := New(Config{
		TTL:        30 * time.Second,
		Driver:     NewDriverLRU(10),
		Compressor: CompressorSnappy{},
	})
	defer cache.Stop()
	handler := cache.Middleware(http.HandlerFunc(success1kHandler))
	r, _ := http.NewRequest("GET", "/", nil)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		w := &noopWriter{http.Header{}}
		for i := 0; pb.Next(); i++ {
			r.URL.Path = "/" + strconv.Itoa(i)
			handler.ServeHTTP(w, r)
		}
	})
}

type noopWriter struct {
	header http.Header
}

func (w *noopWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func (w *noopWriter) Header() http.Header {
	return w.header
}

func (w *noopWriter) WriteHeader(code int) {}

func successHandler(w http.ResponseWriter, r *http.Request) {}

var json1k = []byte(`{"counts":[{"bucket":1569888000,"total":115,"unique":39},{"bucket":1569801600,"total":150,"unique":38},{"bucket":1569715200,"total":129,"unique":34},{"bucket":1569628800,"total":142,"unique":34},{"bucket":1569542400,"total":151,"unique":39},{"bucket":1569456000,"total":145,"unique":44},{"bucket":1569369600,"total":143,"unique":49},{"bucket":1569283200,"total":174,"unique":46},{"bucket":1569196800,"total":407,"unique":52},{"bucket":1569110400,"total":357,"unique":51},{"bucket":1569024000,"total":227,"unique":44},{"bucket":1568937600,"total":257,"unique":44},{"bucket":1568851200,"total":238,"unique":47},{"bucket":1568764800,"total":246,"unique":62}],"summary":{"total":2881,"unique":108}}`)

func success1kHandler(w http.ResponseWriter, r *http.Request) {
	w.Write(json1k)
}
