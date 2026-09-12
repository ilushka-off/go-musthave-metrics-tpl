package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/hash"
	"go.uber.org/zap"
)

func newTestHashHandler(t *testing.T, key string) http.Handler {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	})
	return Hash(key, zap.NewNop())(inner)
}

func TestHash_BodyWithoutHeaderPassesThrough(t *testing.T) {
	// Так ведёт себя официальный metricstest (TestIteration14): часть запросов
	// шлётся без заголовка HashSHA256 даже при заданном ключе, и сервер должен
	// пропускать их, а не отбрасывать — проверка идёт только если заголовок пришёл.
	wrapped := newTestHashHandler(t, "secret")

	req := httptest.NewRequest(http.MethodPost, "/updates", strings.NewReader("some body"))
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (без заголовка проверка должна пропускаться)", rec.Code, http.StatusOK)
	}
}

func TestHash_EmptyBodyPassesWithoutHeader(t *testing.T) {
	wrapped := newTestHashHandler(t, "secret")

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/Alloc", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (без тела подпись не нужна)", rec.Code, http.StatusOK)
	}
}

func TestHash_ValidSignatureIsAccepted(t *testing.T) {
	wrapped := newTestHashHandler(t, "secret")

	body := "some body"
	sig := hash.Sign([]byte(body), "secret")

	req := httptest.NewRequest(http.MethodPost, "/updates", strings.NewReader(body))
	req.Header.Set("HashSHA256", sig)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHash_InvalidSignatureIsRejected(t *testing.T) {
	wrapped := newTestHashHandler(t, "secret")

	req := httptest.NewRequest(http.MethodPost, "/updates", strings.NewReader("some body"))
	req.Header.Set("HashSHA256", "wrong")
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
