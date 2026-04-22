package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	mocks "github.com/EwanGreer/chatatui/internal/server/api/_mocks"
	"github.com/EwanGreer/chatatui/internal/server/hub"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type testBroker struct {
	mu   sync.Mutex
	subs map[uuid.UUID][]chan []byte
}

func newTestBroker() *testBroker {
	return &testBroker{subs: make(map[uuid.UUID][]chan []byte)}
}

func (b *testBroker) Publish(_ context.Context, roomID uuid.UUID, msg []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ch := range b.subs[roomID] {
		select {
		case ch <- msg:
		default:
		}
	}
	return nil
}

func (b *testBroker) Subscribe(_ context.Context, roomID uuid.UUID) (<-chan []byte, func(), error) {
	ch := make(chan []byte, 64)
	b.mu.Lock()
	b.subs[roomID] = append(b.subs[roomID], ch)
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		chans := b.subs[roomID]
		for i, c := range chans {
			if c == ch {
				b.subs[roomID] = append(chans[:i], chans[i+1:]...)
				break
			}
		}
		close(ch)
	}, nil
}

// newWSHandlerRouter wires a WSHandler into a chi router matching the real route
// shape, without starting the hub status goroutine.
func newWSHandlerRouter(svc ChatService) http.Handler {
	h := &WSHandler{
		hub:                 hub.NewHub(newTestBroker()),
		svc:                 svc,
		messageHistoryLimit: 50,
	}
	r := chi.NewRouter()
	r.Get("/ws/{roomID}", h.Handle)
	return r
}

func parseErrorResponse(t *testing.T, body []byte) apiError {
	t.Helper()
	var resp apiError
	require.NoError(t, json.Unmarshal(body, &resp))
	return resp
}

func TestWSHandler_MissingRoomID(t *testing.T) {
	svc := mocks.NewMockChatService(t)

	// Register a route without the {roomID} segment so chi sets it to "".
	h := &WSHandler{hub: hub.NewHub(newTestBroker()), svc: svc, messageHistoryLimit: 50}
	r := chi.NewRouter()
	r.Get("/ws/", h.Handle)

	req := httptest.NewRequest(http.MethodGet, "/ws/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseErrorResponse(t, w.Body.Bytes())
	assert.Equal(t, "ROOM_REQUIRED", resp.Code)
}

func TestWSHandler_InvalidRoomUUID(t *testing.T) {
	svc := mocks.NewMockChatService(t)
	router := newWSHandlerRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/ws/not-a-uuid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseErrorResponse(t, w.Body.Bytes())
	assert.Equal(t, "INVALID_ROOM_ID", resp.Code)
}

func TestWSHandler_RoomNotFound(t *testing.T) {
	roomID := uuid.New()
	svc := mocks.NewMockChatService(t)
	svc.EXPECT().GetRoom(roomID).Return(nil, gorm.ErrRecordNotFound)

	router := newWSHandlerRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/ws/"+roomID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	resp := parseErrorResponse(t, w.Body.Bytes())
	assert.Equal(t, "ROOM_NOT_FOUND", resp.Code)
}

func TestWSHandler_RoomLookupInternalError(t *testing.T) {
	roomID := uuid.New()
	svc := mocks.NewMockChatService(t)
	svc.EXPECT().GetRoom(roomID).Return(nil, errors.New("db connection lost"))

	router := newWSHandlerRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/ws/"+roomID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	resp := parseErrorResponse(t, w.Body.Bytes())
	assert.Equal(t, "INTERNAL_ERROR", resp.Code)
}
