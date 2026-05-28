package ws

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zyy125/im-system/internal/logging"
	"github.com/zyy125/im-system/internal/model"
	"github.com/zyy125/im-system/internal/repository"
	"github.com/zyy125/im-system/internal/service"
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

func (r *hubTestPresenceRepo) RefreshOnline(_ context.Context, userID uint64) error {
	r.online[userID] = true
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

func (r *hubTestPresenceRepo) BatchGetOnline(_ context.Context, userIDs []uint64) (map[uint64]bool, error) {
	result := make(map[uint64]bool, len(userIDs))
	for _, userID := range userIDs {
		result[userID] = r.online[userID]
	}
	return result, nil
}

type hubTestConversationService struct {
	service.ConversationService
	listFn func(ctx context.Context, userID uint64) ([]model.Message, error)
}

func (l *hubTestConversationService) ListOfflineMessages(ctx context.Context, userID uint64) ([]model.Message, error) {
	if l != nil && l.listFn != nil {
		return l.listFn(ctx, userID)
	}
	return []model.Message{}, nil
}

type hubTestFriendRepo struct {
	repository.FriendRepo
	listFn func(ctx context.Context, userID uint64) ([]uint64, error)
}

func (r *hubTestFriendRepo) ListFriendIDs(ctx context.Context, userID uint64) ([]uint64, error) {
	if r != nil && r.listFn != nil {
		return r.listFn(ctx, userID)
	}
	return []uint64{}, nil
}

type hubTestUserService struct {
	service.UserService
}

func TestHub_RegisterFlushesOfflineAndPendingMessages(t *testing.T) {
	presenceRepo := newHubTestPresenceRepo()
	loaderGate := make(chan struct{})
	conversationSvc := &hubTestConversationService{
		listFn: func(ctx context.Context, userID uint64) ([]model.Message, error) {
			if userID != 1 {
				return []model.Message{}, nil
			}
			<-loaderGate
			return []model.Message{
				{ID: 1, MsgID: "off-1", ConversationID: 10, Type: model.MessageTypeText, From: 2, SendTime: 1000, Content: "offline-1"},
				{ID: 2, MsgID: "off-2", ConversationID: 10, Type: model.MessageTypeText, From: 2, SendTime: 2000, Content: "offline-2"},
			}, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	hub := NewHub(presenceRepo, conversationSvc, nil, &hubTestUserService{})
	go func() {
		defer close(done)
		hub.Run(ctx)
	}()

	client := &Client{UserID: 1, Send: make(chan []byte, 8)}
	hub.Register <- client
	waitForUserID(t, presenceRepo.setOnline, 1)

	hub.Forward <- &ForwardMessage{To: 1, ConversationID: 10, Content: []byte("live-1")}
	close(loaderGate)

	first := readHubPayload(t, client.Send)
	second := readHubPayload(t, client.Send)
	third := readHubPayload(t, client.Send)

	firstMsg := decodeMessage(t, first)
	secondMsg := decodeMessage(t, second)
	assert.Equal(t, "off-1", firstMsg.MsgID)
	assert.Equal(t, "off-2", secondMsg.MsgID)
	assert.Equal(t, "live-1", string(third))

	hub.EnqueueUnregister(client)
	waitForUserID(t, presenceRepo.setOffline, 1)
	cancel()
	waitForHubDone(t, done)
}

func TestHub_PresenceBroadcastsOnlyOnFirstConnectAndLastDisconnect(t *testing.T) {
	presenceRepo := newHubTestPresenceRepo()
	friendRepo := &hubTestFriendRepo{
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
	hub := NewHub(presenceRepo, nil, friendRepo, &hubTestUserService{})
	go func() {
		defer close(done)
		hub.Run(ctx)
	}()

	friend := &Client{UserID: 2, Send: make(chan []byte, 8)}
	hub.Register <- friend
	waitForUserID(t, presenceRepo.setOnline, 2)

	first := &Client{UserID: 1, Send: make(chan []byte, 8)}
	second := &Client{UserID: 1, Send: make(chan []byte, 8)}
	hub.Register <- first
	waitForUserID(t, presenceRepo.setOnline, 1)
	hub.Register <- second

	onlineEvent := decodePresenceEvent(t, readHubPayload(t, friend.Send))
	assert.Equal(t, EventTypePresenceChanged, onlineEvent.Type)
	assert.Equal(t, uint64(1), onlineEvent.UserID)
	assert.True(t, onlineEvent.Online)

	assertNoUserID(t, presenceRepo.setOnline)

	hub.EnqueueUnregister(first)
	assertNoUserID(t, presenceRepo.setOffline)

	hub.EnqueueUnregister(second)
	waitForUserID(t, presenceRepo.setOffline, 1)
	offlineEvent := decodePresenceEvent(t, readHubPayload(t, friend.Send))
	assert.Equal(t, EventTypePresenceChanged, offlineEvent.Type)
	assert.Equal(t, uint64(1), offlineEvent.UserID)
	assert.False(t, offlineEvent.Online)

	hub.EnqueueUnregister(friend)
	waitForUserID(t, presenceRepo.setOffline, 2)
	cancel()
	waitForHubDone(t, done)
}

func TestHub_MultipleConnectionsReceiveBroadcast(t *testing.T) {
	presenceRepo := newHubTestPresenceRepo()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	hub := NewHub(presenceRepo, nil, nil, &hubTestUserService{})
	go func() {
		defer close(done)
		hub.Run(ctx)
	}()

	first := &Client{UserID: 7, Send: make(chan []byte, 4)}
	second := &Client{UserID: 7, Send: make(chan []byte, 4)}
	hub.Register <- first
	waitForUserID(t, presenceRepo.setOnline, 7)
	hub.Register <- second
	time.Sleep(50 * time.Millisecond)

	hub.Forward <- &ForwardMessage{To: 7, ConversationID: 20, Content: []byte("after-reconnect")}
	assert.Equal(t, "after-reconnect", string(readHubPayload(t, first.Send)))
	assert.Equal(t, "after-reconnect", string(readHubPayload(t, second.Send)))

	hub.EnqueueUnregister(first)
	hub.EnqueueUnregister(second)
	waitForUserID(t, presenceRepo.setOffline, 7)
	cancel()
	waitForHubDone(t, done)
}

func TestHub_UnreadyConnectionDoesNotBlockReadySiblingConnection(t *testing.T) {
	presenceRepo := newHubTestPresenceRepo()
	var callCount int
	loaderGate := make(chan struct{})
	conversationSvc := &hubTestConversationService{
		listFn: func(ctx context.Context, userID uint64) ([]model.Message, error) {
			callCount++
			if userID == 9 && callCount == 1 {
				<-loaderGate
			}
			return []model.Message{}, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	hub := NewHub(presenceRepo, conversationSvc, nil, &hubTestUserService{})
	go func() {
		defer close(done)
		hub.Run(ctx)
	}()

	slow := &Client{UserID: 9, Send: make(chan []byte, 4)}
	fast := &Client{UserID: 9, Send: make(chan []byte, 4)}
	hub.Register <- slow
	waitForUserID(t, presenceRepo.setOnline, 9)
	hub.Register <- fast
	time.Sleep(50 * time.Millisecond)

	hub.Forward <- &ForwardMessage{To: 9, ConversationID: 33, Content: []byte("live-ready")}
	assert.Equal(t, "live-ready", string(readHubPayload(t, fast.Send)))

	close(loaderGate)
	assert.Equal(t, "live-ready", string(readHubPayload(t, slow.Send)))

	hub.EnqueueUnregister(slow)
	assertNoUserID(t, presenceRepo.setOffline)
	hub.EnqueueUnregister(fast)
	waitForUserID(t, presenceRepo.setOffline, 9)
	cancel()
	waitForHubDone(t, done)
}

func TestHub_PendingOverflowSendsSyncRequiredAfterBootstrap(t *testing.T) {
	presenceRepo := newHubTestPresenceRepo()
	loaderGate := make(chan struct{})
	conversationSvc := &hubTestConversationService{
		listFn: func(ctx context.Context, userID uint64) ([]model.Message, error) {
			if userID == 11 {
				<-loaderGate
			}
			return []model.Message{}, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	hub := NewHub(presenceRepo, conversationSvc, nil, &hubTestUserService{})
	go func() {
		defer close(done)
		hub.Run(ctx)
	}()

	client := &Client{UserID: 11, Send: make(chan []byte, maxPendingPerConnection+4)}
	hub.Register <- client
	waitForUserID(t, presenceRepo.setOnline, 11)

	for i := 0; i < maxPendingPerConnection+1; i++ {
		hub.Forward <- &ForwardMessage{To: 11, ConversationID: 88, Content: []byte("queued")}
	}

	time.Sleep(50 * time.Millisecond)
	close(loaderGate)
	var syncEvent SyncRequiredData
	foundSyncRequired := false
	for i := 0; i < maxPendingPerConnection+1; i++ {
		payload := readHubPayload(t, client.Send)
		if string(payload) == "queued" {
			continue
		}
		syncEvent = decodeSyncRequiredEvent(t, payload)
		foundSyncRequired = true
		break
	}

	require.True(t, foundSyncRequired, "expected sync_required payload after pending overflow")
	assert.Equal(t, uint64(88), syncEvent.ConversationID)
	assert.Equal(t, SyncReasonPendingQueueFull, syncEvent.Reason)

	hub.EnqueueUnregister(client)
	waitForUserID(t, presenceRepo.setOffline, 11)
	cancel()
	waitForHubDone(t, done)
}

func TestHub_EnqueueUnregisterAfterShutdownDoesNotBlock(t *testing.T) {
	presenceRepo := newHubTestPresenceRepo()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	hub := NewHub(presenceRepo, nil, nil, &hubTestUserService{})
	go func() {
		defer close(done)
		hub.Run(ctx)
	}()

	client := &Client{UserID: 15, Send: make(chan []byte, 4)}
	hub.Register <- client
	waitForUserID(t, presenceRepo.setOnline, 15)

	cancel()
	waitForHubDone(t, done)

	unregistered := make(chan struct{})
	go func() {
		hub.EnqueueUnregister(client)
		close(unregistered)
	}()

	select {
	case <-unregistered:
	case <-time.After(2 * time.Second):
		t.Fatal("enqueue unregister blocked after hub shutdown")
	}
}

func TestHub_EnqueueForwardsDeliversBatch(t *testing.T) {
	presenceRepo := newHubTestPresenceRepo()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	hub := NewHub(presenceRepo, nil, nil, &hubTestUserService{})
	go func() {
		defer close(done)
		hub.Run(ctx)
	}()

	client := &Client{UserID: 21, Send: make(chan []byte, 4)}
	hub.Register <- client
	waitForUserID(t, presenceRepo.setOnline, 21)

	hub.EnqueueForwards(context.Background(), []*ForwardMessage{
		{To: 21, ConversationID: 1, Content: []byte("first")},
		{To: 21, ConversationID: 1, Content: []byte("second")},
	})

	assert.Equal(t, "first", string(readHubPayload(t, client.Send)))
	assert.Equal(t, "second", string(readHubPayload(t, client.Send)))

	hub.EnqueueUnregister(client)
	waitForUserID(t, presenceRepo.setOffline, 21)
	cancel()
	waitForHubDone(t, done)
}

func TestHub_EnqueueForwardsMarksSyncWhenForwardQueueFull(t *testing.T) {
	hub := NewHub(nil, nil, nil, &hubTestUserService{})
	hub.Forward = make(chan *ForwardMessage, 1)
	hub.markSync = make(chan *syncRequest, 1)

	ctx := logging.ContextWithLogger(context.Background(), logging.With("event_type", "test_enqueue_forwards"))
	hub.Forward <- &ForwardMessage{To: 30, ConversationID: 1, Content: []byte("occupied")}

	hub.EnqueueForwards(ctx, []*ForwardMessage{
		{To: 30, ConversationID: 99, Content: []byte("overflow")},
	})

	select {
	case req := <-hub.markSync:
		require.NotNil(t, req)
		assert.Equal(t, uint64(30), req.userID)
		assert.Equal(t, uint64(99), req.conversationID)
		assert.Equal(t, SyncReasonForwardQueueFull, req.reason)
		assert.Equal(t, int64(1), hub.Snapshot().ForwardQueueFullTotal)
	case <-time.After(2 * time.Second):
		t.Fatal("expected markSync request when forward queue is full")
	}
}

func TestHub_SnapshotTracksRegisterAndUnregister(t *testing.T) {
	presenceRepo := newHubTestPresenceRepo()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	hub := NewHub(presenceRepo, nil, nil, &hubTestUserService{})
	go func() {
		defer close(done)
		hub.Run(ctx)
	}()

	first := &Client{UserID: 1, Send: make(chan []byte, 4)}
	second := &Client{UserID: 1, Send: make(chan []byte, 4)}
	hub.Register <- first
	waitForUserID(t, presenceRepo.setOnline, 1)
	hub.Register <- second
	time.Sleep(50 * time.Millisecond)

	snapshot := hub.Snapshot()
	assert.Equal(t, int64(1), snapshot.Users)
	assert.Equal(t, int64(2), snapshot.Connections)
	assert.Equal(t, int64(2), snapshot.RegisterTotal)

	hub.EnqueueUnregister(first)
	hub.EnqueueUnregister(second)
	waitForUserID(t, presenceRepo.setOffline, 1)
	time.Sleep(50 * time.Millisecond)

	snapshot = hub.Snapshot()
	assert.Equal(t, int64(0), snapshot.Users)
	assert.Equal(t, int64(0), snapshot.Connections)
	assert.Equal(t, int64(2), snapshot.UnregisterTotal)

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

func assertNoUserID(t *testing.T, ch <-chan uint64) {
	t.Helper()

	select {
	case got := <-ch:
		t.Fatalf("expected no user id event, got %d", got)
	case <-time.After(200 * time.Millisecond):
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

func decodeMessage(t *testing.T, payload []byte) ServerMessage {
	t.Helper()

	var env Envelope
	require.NoError(t, json.Unmarshal(payload, &env))
	require.Equal(t, EventTypeMessageCreated, env.Type)

	var msg ServerMessage
	require.NoError(t, json.Unmarshal(env.Data, &msg))
	return msg
}

type decodedPresenceEvent struct {
	Type   string
	UserID uint64
	Online bool
}

func decodePresenceEvent(t *testing.T, payload []byte) decodedPresenceEvent {
	t.Helper()

	var env Envelope
	require.NoError(t, json.Unmarshal(payload, &env))

	var data PresenceChangedData
	require.NoError(t, json.Unmarshal(env.Data, &data))
	return decodedPresenceEvent{
		Type:   env.Type,
		UserID: data.UserID,
		Online: data.Online,
	}
}

func decodeSyncRequiredEvent(t *testing.T, payload []byte) SyncRequiredData {
	t.Helper()

	var env Envelope
	require.NoError(t, json.Unmarshal(payload, &env))
	require.Equal(t, EventTypeSyncRequired, env.Type)

	var data SyncRequiredData
	require.NoError(t, json.Unmarshal(env.Data, &data))
	return data
}
