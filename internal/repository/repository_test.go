package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/EwanGreer/chatatui/internal/domain"
	"github.com/google/uuid"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var testDB *gorm.DB

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		panic("failed to start postgres container: " + err.Error())
	}
	defer func() { _ = container.Terminate(ctx) }()

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic("failed to get connection string: " + err.Error())
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic("failed to connect to test database: " + err.Error())
	}

	if err := db.AutoMigrate(&User{}, &Room{}, &Message{}); err != nil {
		panic("failed to migrate test database: " + err.Error())
	}

	testDB = db

	os.Exit(m.Run())
}

// truncate clears all tables between tests to ensure isolation.
func truncate(t *testing.T) {
	t.Helper()
	testDB.Exec("TRUNCATE TABLE room_members, messages, rooms, users RESTART IDENTITY CASCADE")
}

// helpers

func createUser(t *testing.T, name, apiKey string) *domain.User {
	t.Helper()
	u := &domain.User{Name: name, APIKey: apiKey}
	if err := NewUserRepository(testDB).Create(u); err != nil {
		t.Fatalf("createUser: %v", err)
	}
	return u
}

func createRoom(t *testing.T, name string) *domain.Room {
	t.Helper()
	r := &domain.Room{Name: name}
	if err := NewRoomRepository(testDB).Create(r); err != nil {
		t.Fatalf("createRoom: %v", err)
	}
	return r
}

// ── UserRepository ────────────────────────────────────────────────────────────

func TestUserRepository_CreateAndGetByAPIKey(t *testing.T) {
	truncate(t)
	repo := NewUserRepository(testDB)

	u := &domain.User{Name: "alice", APIKey: HashAPIKey("secret-key")}
	if err := repo.Create(u); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByAPIKey("secret-key")
	if err != nil {
		t.Fatalf("GetByAPIKey: %v", err)
	}
	if got.Name != "alice" {
		t.Errorf("expected name alice, got %s", got.Name)
	}
	if got.ID == (uuid.UUID{}) {
		t.Error("expected non-nil ID")
	}
}

func TestUserRepository_GetByAPIKey_NotFound(t *testing.T) {
	truncate(t)
	repo := NewUserRepository(testDB)

	_, err := repo.GetByAPIKey("does-not-exist")
	if err == nil {
		t.Fatal("expected error for missing key, got nil")
	}
}

func TestUserRepository_DuplicateAPIKey_Rejected(t *testing.T) {
	truncate(t)
	repo := NewUserRepository(testDB)

	key := HashAPIKey("same-key")
	if err := repo.Create(&domain.User{Name: "alice", APIKey: key}); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	if err := repo.Create(&domain.User{Name: "bob", APIKey: key}); err == nil {
		t.Fatal("expected unique constraint violation, got nil")
	}
}

func TestUserRepository_GetByID(t *testing.T) {
	truncate(t)
	u := createUser(t, "alice", HashAPIKey("k1"))

	got, err := NewUserRepository(testDB).GetByID(u.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != u.ID {
		t.Errorf("expected ID %s, got %s", u.ID, got.ID)
	}
}

// ── RoomRepository ────────────────────────────────────────────────────────────

func TestRoomRepository_CreateAndGetByID(t *testing.T) {
	truncate(t)
	repo := NewRoomRepository(testDB)

	r := &domain.Room{Name: "general"}
	if err := repo.Create(r); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if r.ID == (uuid.UUID{}) {
		t.Error("expected ID to be set after create")
	}

	got, err := repo.GetByID(r.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "general" {
		t.Errorf("expected name general, got %s", got.Name)
	}
}

func TestRoomRepository_GetByID_NotFound(t *testing.T) {
	truncate(t)
	id := uuid.New()
	_, err := NewRoomRepository(testDB).GetByID(id)
	if err == nil {
		t.Fatal("expected error for missing room")
	}
}

func TestRoomRepository_List_OrderedByCreatedAt(t *testing.T) {
	truncate(t)
	repo := NewRoomRepository(testDB)

	names := []string{"alpha", "beta", "gamma"}
	for _, n := range names {
		r := &domain.Room{Name: n}
		if err := repo.Create(r); err != nil {
			t.Fatalf("Create %s: %v", n, err)
		}
		time.Sleep(2 * time.Millisecond) // ensure distinct created_at
	}

	rooms, err := repo.List(10, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rooms) != 3 {
		t.Fatalf("expected 3 rooms, got %d", len(rooms))
	}
	// List returns DESC so newest first
	if rooms[0].Name != "gamma" {
		t.Errorf("expected gamma first, got %s", rooms[0].Name)
	}
}

func TestRoomRepository_List_RespectsLimit(t *testing.T) {
	truncate(t)
	repo := NewRoomRepository(testDB)

	for i := 0; i < 5; i++ {
		if err := repo.Create(&domain.Room{Name: "room"}); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	rooms, err := repo.List(3, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rooms) != 3 {
		t.Errorf("expected 3, got %d", len(rooms))
	}
}

func TestRoomRepository_AddAndRemoveMember(t *testing.T) {
	truncate(t)
	u := createUser(t, "alice", HashAPIKey("k1"))
	r := createRoom(t, "general")
	repo := NewRoomRepository(testDB)

	if err := repo.AddMember(r.ID, u.ID); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	got, err := repo.GetByID(r.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if len(got.Members) != 1 || got.Members[0].UserID != u.ID {
		t.Errorf("expected member alice, got %+v", got.Members)
	}

	if err := repo.RemoveMember(r.ID, u.ID); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}

	got, _ = repo.GetByID(r.ID)
	if len(got.Members) != 0 {
		t.Errorf("expected 0 members after remove, got %d", len(got.Members))
	}
}

func TestRoomRepository_AddMember_Idempotent(t *testing.T) {
	truncate(t)
	u := createUser(t, "alice", HashAPIKey("k1"))
	r := createRoom(t, "general")
	repo := NewRoomRepository(testDB)

	if err := repo.AddMember(r.ID, u.ID); err != nil {
		t.Fatalf("first AddMember: %v", err)
	}
	if err := repo.AddMember(r.ID, u.ID); err != nil {
		t.Fatalf("duplicate AddMember should be idempotent: %v", err)
	}

	got, _ := repo.GetByID(r.ID)
	if len(got.Members) != 1 {
		t.Errorf("expected 1 member, got %d", len(got.Members))
	}
}

// ── MessageRepository ─────────────────────────────────────────────────────────

func TestMessageRepository_CreateAndGetByRoom(t *testing.T) {
	truncate(t)
	u := createUser(t, "alice", HashAPIKey("k1"))
	r := createRoom(t, "general")
	repo := NewMessageRepository(testDB)

	msg := &domain.Message{Content: "hello", SenderID: u.ID, RoomID: r.ID}
	if err := repo.Create(msg); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if msg.ID == (uuid.UUID{}) {
		t.Error("expected ID to be set after create")
	}

	messages, err := repo.GetByRoom(r.ID, 10, 0)
	if err != nil {
		t.Fatalf("GetByRoom: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	if messages[0].Content != "hello" {
		t.Errorf("expected content hello, got %s", messages[0].Content)
	}
	if messages[0].Author != "alice" {
		t.Errorf("expected author alice, got %s", messages[0].Author)
	}
}

func TestMessageRepository_GetByRoom_OrderedDescending(t *testing.T) {
	truncate(t)
	u := createUser(t, "alice", HashAPIKey("k1"))
	r := createRoom(t, "general")
	repo := NewMessageRepository(testDB)

	for _, content := range []string{"first", "second", "third"} {
		if err := repo.Create(&domain.Message{Content: content, SenderID: u.ID, RoomID: r.ID}); err != nil {
			t.Fatalf("Create: %v", err)
		}
		time.Sleep(2 * time.Millisecond)
	}

	messages, err := repo.GetByRoom(r.ID, 10, 0)
	if err != nil {
		t.Fatalf("GetByRoom: %v", err)
	}
	if messages[0].Content[0] != 't' { // "third"
		t.Errorf("expected third first (DESC), got %s", messages[0].Content)
	}
}

func TestMessageRepository_GetByRoom_RespectsLimitAndOffset(t *testing.T) {
	truncate(t)
	u := createUser(t, "alice", HashAPIKey("k1"))
	r := createRoom(t, "general")
	repo := NewMessageRepository(testDB)

	for i := 0; i < 5; i++ {
		if err := repo.Create(&domain.Message{Content: "msg", SenderID: u.ID, RoomID: r.ID}); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	page1, _ := repo.GetByRoom(r.ID, 3, 0)
	page2, _ := repo.GetByRoom(r.ID, 3, 3)

	if len(page1) != 3 {
		t.Errorf("expected 3 on page 1, got %d", len(page1))
	}
	if len(page2) != 2 {
		t.Errorf("expected 2 on page 2, got %d", len(page2))
	}
}

func TestMessageRepository_GetByRoom_IsolatedByRoom(t *testing.T) {
	truncate(t)
	u := createUser(t, "alice", HashAPIKey("k1"))
	r1 := createRoom(t, "room1")
	r2 := createRoom(t, "room2")
	repo := NewMessageRepository(testDB)

	_ = repo.Create(&domain.Message{Content: "in r1", SenderID: u.ID, RoomID: r1.ID})
	_ = repo.Create(&domain.Message{Content: "in r2", SenderID: u.ID, RoomID: r2.ID})

	msgs, _ := repo.GetByRoom(r1.ID, 10, 0)
	if len(msgs) != 1 || msgs[0].Content != "in r1" {
		t.Errorf("room isolation failed: got %+v", msgs)
	}
}
