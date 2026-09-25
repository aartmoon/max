package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResidentStatusTransitionRouteIsRemoved(t *testing.T) {
	h := Handler{MockStatusEnabled: true}.Routes()
	res := httptest.NewRecorder()

	h.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/requests/1/mock-next-status", nil))

	if res.Code != http.StatusNotFound {
		t.Fatalf("resident status route must be unavailable, status=%d body=%s", res.Code, res.Body.String())
	}
}
