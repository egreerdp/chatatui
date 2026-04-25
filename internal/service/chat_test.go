package service

import (
	"errors"
	"testing"
	"time"

	"github.com/EwanGreer/chatatui/internal/domain"
	mocks "github.com/EwanGreer/chatatui/internal/service/_mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const testHistoryLimit = 50

func TestChatService_GetRoom(t *testing.T) {
	roomID := uuid.New()

	tests := []struct {
		name      string
		setup     func(*mocks.MockRoomStore)
		wantRoom  *domain.Room
		wantErrIs error
	}{
		{
			name: "returns Room for existing room",
			setup: func(m *mocks.MockRoomStore) {
				m.EXPECT().GetByID(roomID).Return(&domain.Room{
					ID:   roomID,
					Name: "general",
				}, nil)
			},
			wantRoom: &domain.Room{ID: roomID, Name: "general"},
		},
		{
			name: "maps record not found to domain.ErrNotFound",
			setup: func(m *mocks.MockRoomStore) {
				m.EXPECT().GetByID(roomID).Return(nil, gorm.ErrRecordNotFound)
			},
			wantErrIs: domain.ErrNotFound,
		},
		{
			name: "propagates unexpected store error",
			setup: func(m *mocks.MockRoomStore) {
				m.EXPECT().GetByID(roomID).Return(nil, errors.New("db unavailable"))
			},
			wantErrIs: errors.New("db unavailable"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rooms := mocks.NewMockRoomStore(t)
			messages := mocks.NewMockMessageStore(t)
			tt.setup(rooms)

			svc := NewChatService(rooms, messages, testHistoryLimit)
			got, err := svc.GetRoom(roomID)

			if tt.wantErrIs != nil {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErrIs.Error())
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantRoom, got)
		})
	}
}

func TestChatService_JoinRoom(t *testing.T) {
	roomID := uuid.New()
	userID := uuid.New()
	msgID := uuid.New()
	now := time.Now().Truncate(time.Second)

	domainHistory := []domain.Message{{ID: msgID, Author: "alice", Content: "hello", CreatedAt: now}}
	wantHistory := []domain.WireMessage{domainHistory[0].ToWireMessage()}

	tests := []struct {
		name        string
		setupRooms  func(*mocks.MockRoomStore)
		setupMsgs   func(*mocks.MockMessageStore)
		wantHistory []domain.WireMessage
		wantErr     bool
	}{
		{
			name: "records membership and returns history as wire messages",
			setupRooms: func(m *mocks.MockRoomStore) {
				m.EXPECT().AddMember(roomID, userID).Return(nil)
			},
			setupMsgs: func(m *mocks.MockMessageStore) {
				m.EXPECT().GetByRoom(roomID, testHistoryLimit, 0).Return(domainHistory, nil)
			},
			wantHistory: wantHistory,
		},
		{
			name: "continues to return history when membership fails",
			setupRooms: func(m *mocks.MockRoomStore) {
				m.EXPECT().AddMember(roomID, userID).Return(errors.New("constraint violation"))
			},
			setupMsgs: func(m *mocks.MockMessageStore) {
				m.EXPECT().GetByRoom(roomID, testHistoryLimit, 0).Return(domainHistory, nil)
			},
			wantHistory: wantHistory,
		},
		{
			name: "propagates history fetch error",
			setupRooms: func(m *mocks.MockRoomStore) {
				m.EXPECT().AddMember(roomID, userID).Return(nil)
			},
			setupMsgs: func(m *mocks.MockMessageStore) {
				m.EXPECT().GetByRoom(roomID, testHistoryLimit, 0).Return(nil, errors.New("query failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rooms := mocks.NewMockRoomStore(t)
			messages := mocks.NewMockMessageStore(t)
			tt.setupRooms(rooms)
			tt.setupMsgs(messages)

			svc := NewChatService(rooms, messages, testHistoryLimit)
			got, err := svc.JoinRoom(roomID, userID)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantHistory, got)
		})
	}
}

func TestChatService_PublishMessage(t *testing.T) {
	senderID := uuid.New()
	roomID := uuid.New()
	content := []byte("hello world")
	senderName := "alice"

	tests := []struct {
		name    string
		setup   func(*mocks.MockMessageStore)
		wantErr bool
	}{
		{
			name: "persists message and returns populated domain model",
			setup: func(m *mocks.MockMessageStore) {
				m.EXPECT().Create(mockAny).RunAndReturn(func(msg *domain.Message) error {
					msg.ID = uuid.New()
					msg.CreatedAt = time.Now()
					return nil
				})
			},
		},
		{
			name: "propagates store error",
			setup: func(m *mocks.MockMessageStore) {
				m.EXPECT().Create(mockAny).Return(errors.New("insert failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rooms := mocks.NewMockRoomStore(t)
			messages := mocks.NewMockMessageStore(t)
			tt.setup(messages)

			svc := NewChatService(rooms, messages, testHistoryLimit)
			got, err := svc.PublishMessage(content, senderID, senderName, roomID)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, got.ID)
			assert.NotZero(t, got.Timestamp)
			assert.Equal(t, senderName, got.Author)
			assert.Equal(t, string(content), got.Content)
			assert.Equal(t, domain.WireMessageTypeChat, got.Type)
		})
	}
}

// mockAny matches any argument — used where the exact value is set inside the function.
var mockAny = mock.MatchedBy(func(_ *domain.Message) bool { return true })
