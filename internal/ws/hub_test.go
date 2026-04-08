package ws

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zyy125/im-system/internal/model"
)

type hubTestPresenceRepo struct {
	online     map[uint64]bool
	setOnline  chan uint64
	setOffline chan uint64
}

func newHubTestPresenceRepo() *hubTestPresenceRepo {
	return &hubTestPresenceRepo{
		online:     make(map[uint64]bool),
		setOnline:  make(chan uint64, 16),
		setOffline: make(chan uint64, 16),
	}
}

func (r *hubTestPresenceRepo) SetOnline(_ context.Context, userID uint64) error {
	r.online[userID] = true
	r.setOnline <- userID
	return nil
}

func (r *hubTestPresenceRepo) SetOffline(_ context.Context, userID uint64) error {
	delete(r.online, userID)
	r.setOffline <- userID
	return nil
}

func (r *hubTestPresenceRepo) IsOnline(_ context.Context, userID uint64) (bool, error) {
	return r.online[userID], nil
}

type hubTestOfflineLoader struct {
	listFn func(ctx context.Context, userID uint64) ([]model.ChatMessage, error)
}

func (l *hubTestOfflineLoader) ListOfflineMessages(ctx context.Context, userID uint64) ([]model.ChatMessage, error) {
	if l != nil && l.listFn != nil {
		return l.listFn(ctx, userID)
	}
	return []model.ChatMessage{}, nil
}

type hubTestAudience struct {
	listFn func(ctx context.Context, userID uint64) ([]uint64, error)
}

func (a *hubTestAudience) ListFriendIDs(ctx context.Context, userID uint64) ([]uint64, error) {
	if a != nil && a.listFn != nil {
		return a.listFn(ctx, userID)
	}
	return []uint64{}, nil
}

func TestHub_RegisterFlushesOfflineAndPendingMessages(t *testing.T) {
	presenceRepo := newHubTestPresenceRepo()
	loaderGate := make(chan struct{})
	loader := &hubTestOfflineLoader{
		listFn: func(ctx context.Context, userID uint64) ([]model.ChatMessage, error) {
			if userID != 1 {
				return []model.ChatMessage{}, nil
			}
			<-loaderGate
			return []model.ChatMessage{
				{ID: 1, MsgID: "off-1", ConversationID: "10", From: 2, To: 1, SendTime: 1000, Content: "offline-1"},
				{ID: 2, MsgID: "off-2", ConversationID: "10", From: 2, To: 1, SendTime: 2000, Content: "offline-2"},
			}, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	hub := NewHub(presenceRepo, loader, nil)
	go func() {
		defer close(done)
		hub.Run(ctx)
	}()

	client := &Client{
		UserID: 1,
		Send:   make(chan []byte, 8),
	}
	hub.Register <- client

	waitForUserID(t, presenceRepo.setOnline, 1)

	hub.Forward <- &ForwardMessage{
		To:      1,
		Content: []byte("live-1"),
	}

	close(loaderGate)

	first := readHubPayload(t, client.Send)
	second := readHubPayload(t, client.Send)
	third := readHubPayload(t, client.Send)

	firstMsg := decodeChatMessage(t, first)
	secondMsg := decodeChatMessage(t, second)

	assert.Equal(t, "off-1", firstMsg.MsgID)
	assert.Equal(t, "off-2", secondMsg.MsgID)
	assert.Equal(t, "live-1", string(third))

	hub.Unregister <- client
	waitForUserID(t, presenceRepo.setOffline, 1)
	cancel()
	waitForHubDone(t, done)
}

func TestHub_PresenceQueuedUntilFriendReadyAndBroadcastsOfflineOnUnregister(t *testing.T) {
	presenceRepo := newHubTestPresenceRepo()
	friendReadyGate := make(chan struct{})
	loader := &hubTestOfflineLoader{
		listFn: func(ctx context.Context, userID uint64) ([]model.ChatMessage, error) {
			if userID != 2 {
				return []model.ChatMessage{}, nil
			}
			<-friendReadyGate
			return []model.ChatMessage{}, nil
		},
	}
	audience := &hubTestAudience{
		listFn: func(ctx context.Context, userID uint64) ([]uint64, error) {
			switch userID {
			case 1:
				return []uint64{2}, nil
			case 2:
				return []uint64{1}, nil
			default:
				return []uint64{}, nil
			}
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	hub := NewHub(presenceRepo, loader, audience)
	go func() {
		defer close(done)
		hub.Run(ctx)
	}()

	friend := &Client{UserID: 2, Send: make(chan []byte, 8)}
	hub.Register <- friend
	waitForUserID(t, presenceRepo.setOnline, 2)

	user := &Client{UserID: 1, Send: make(chan []byte, 8)}
	hub.Register <- user
	waitForUserID(t, presenceRepo.setOnline, 1)

	close(friendReadyGate)

	onlineEvent := decodePresenceEvent(t, readHubPayload(t, friend.Send))
	assert.Equal(t, EventTypePresenceChanged, onlineEvent.Type)
	assert.Equal(t, uint64(1), onlineEvent.UserID)
	assert.True(t, onlineEvent.Online)

	hub.Unregister <- user
	waitForUserID(t, presenceRepo.setOffline, 1)

	offlineEvent := decodePresenceEvent(t, readHubPayload(t, friend.Send))
	assert.Equal(t, EventTypePresenceChanged, offlineEvent.Type)
	assert.Equal(t, uint64(1), offlineEvent.UserID)
	assert.False(t, offlineEvent.Online)

	hub.Unregister <- friend
	waitForUserID(t, presenceRepo.setOffline, 2)
	cancel()
	waitForHubDone(t, done)
}

func TestHub_StaleUnregisterDoesNotRemoveCurrentClient(t *testing.T) {
	presenceRepo := newHubTestPresenceRepo()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	hub := NewHub(presenceRepo, nil, nil)
	go func() {
		defer close(done)
		hub.Run(ctx)
	}()

	oldClient := &Client{UserID: 7, Send: make(chan []byte, 4)}
	hub.Register <- oldClient
	waitForUserID(t, presenceRepo.setOnline, 7)

	newClient := &Client{UserID: 7, Send: make(chan []byte, 4)}
	hub.Register <- newClient
	waitForUserID(t, presenceRepo.setOnline, 7)

	hub.Unregister <- oldClient

	hub.Forward <- &ForwardMessage{
		To:      7,
		Content: []byte("after-reconnect"),
	}

	assert.Equal(t, "after-reconnect", string(readHubPayload(t, newClient.Send)))

	hub.Unregister <- newClient
	waitForUserID(t, presenceRepo.setOffline, 7)
	cancel()
	waitForHubDone(t, done)
}

func TestHub_ForwardToOfflineUserIsDropped(t *testing.T) {
	presenceRepo := newHubTestPresenceRepo()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	hub := NewHub(presenceRepo, nil, nil)
	go func() {
		defer close(done)
		hub.Run(ctx)
	}()

	hub.Forward <- &ForwardMessage{
		To:      99,
		Content: []byte("dropped"),
	}
	time.Sleep(100 * time.Millisecond)

	client := &Client{UserID: 99, Send: make(chan []byte, 4)}
	hub.Register <- client
	waitForUserID(t, presenceRepo.setOnline, 99)

	select {
	case payload := <-client.Send:
		t.Fatalf("expected no replayed payload for offline user, got %q", string(payload))
	case <-time.After(200 * time.Millisecond):
	}

	hub.Unregister <- client
	waitForUserID(t, presenceRepo.setOffline, 99)
	cancel()
	waitForHubDone(t, done)
}

func waitForUserID(t *testing.T, ch <-chan uint64, want uint64) {
	t.Helper()

	select {
	case got := <-ch:
		require.Equal(t, want, got)
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for user id %d", want)
	}
}

func waitForHubDone(t *testing.T, done <-chan struct{}) {
	t.Helper()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting hub goroutine to stop")
	}
}

func readHubPayload(t *testing.T, ch <-chan []byte) []byte {
	t.Helper()

	select {
	case payload := <-ch:
		return payload
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting hub payload")
		return nil
	}
}

func decodeChatMessage(t *testing.T, payload []byte) model.ChatMessage {
	t.Helper()

	var env Envelope
	require.NoError(t, json.Unmarshal(payload, &env))
	require.Equal(t, EventTypeChatMessage, env.Type)
	require.Equal(t, ProtocolVersion, env.Version)

	var msg model.ChatMessage
	require.NoError(t, json.Unmarshal(env.Data, &msg))
	return msg
}

type decodedPresenceEvent struct {
	Type    string
	Version int
	UserID  uint64
	Online  bool
}

func decodePresenceEvent(t *testing.T, payload []byte) decodedPresenceEvent {
	t.Helper()

	var env Envelope
	require.NoError(t, json.Unmarshal(payload, &env))

	var data PresenceChangedData
	require.NoError(t, json.Unmarshal(env.Data, &data))
	return decodedPresenceEvent{
		Type:    env.Type,
		Version: env.Version,
		UserID:  data.UserID,
		Online:  data.Online,
	}
}
