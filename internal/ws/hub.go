package ws

import (
	"context"
	"fmt"

	"github.com/zyy125/im-system/internal/logging"
	"github.com/zyy125/im-system/internal/model"
	"github.com/zyy125/im-system/internal/repository"
)

const maxPendingPerConnection = 512

type ForwardMessage struct {
	To             uint64
	From           uint64
	ConversationID uint64
	Content        []byte
}

type ClientBootstrapResult struct {
	Client          *Client
	OfflineMessages [][]byte
}

type PresenceAudienceProvider interface {
	ListFriendIDs(ctx context.Context, userID uint64) ([]uint64, error)
}

type syncRequest struct {
	UserID         uint64
	ConnectionID   string
	ConversationID uint64
	Reason         string
}

type hubConnectionState struct {
	client      *Client
	ready       bool
	closed      bool
	pending     [][]byte
	pendingSync *SyncRequiredData
}

type Hub struct {
	Clients            map[uint64]map[string]*hubConnectionState
	Register           chan *Client
	Unregister         chan *Client
	Forward            chan *ForwardMessage
	LifecycleForward   chan *ForwardMessage
	ClientBootstrapped chan *ClientBootstrapResult

	Lifecycle     ClientLifecycle
	markSync      chan *syncRequest
	closeRequests chan struct{}
	done          chan struct{}
}

type OfflineMessageLoader interface {
	ListOfflineMessages(ctx context.Context, userID uint64) ([]model.Message, error)
}

func NewHub(
	presenceRepo repository.PresenceRepo,
	offlineLoader OfflineMessageLoader,
	presenceAudience PresenceAudienceProvider,
) *Hub {
	lifecycleForward := make(chan *ForwardMessage, 256)
	hub := &Hub{
		Clients:            make(map[uint64]map[string]*hubConnectionState),
		Register:           make(chan *Client, 32),
		Unregister:         make(chan *Client, 64),
		Forward:            make(chan *ForwardMessage, 512),
		LifecycleForward:   lifecycleForward,
		ClientBootstrapped: make(chan *ClientBootstrapResult, 32),
		markSync:           make(chan *syncRequest, 256),
		closeRequests:      make(chan struct{}, 1),
		done:               make(chan struct{}),
	}
	hub.Lifecycle = NewClientLifecycle(
		presenceRepo,
		offlineLoader,
		presenceAudience,
		lifecycleForward,
		func(userID, conversationID uint64, reason string) {
			hub.MarkNeedsSync(userID, "", conversationID, reason)
		},
	)
	return hub
}

func (h *Hub) Run(ctx context.Context) {
	defer close(h.done)

	for {
		select {
		case client := <-h.Register:
			h.handleRegister(ctx, client)

		case client := <-h.Unregister:
			h.removeConnection(ctx, client)

		case msg := <-h.Forward:
			h.forwardLocal(ctx, msg)

		case msg := <-h.LifecycleForward:
			h.forwardLocal(ctx, msg)

		case result := <-h.ClientBootstrapped:
			h.handleBootstrapResult(ctx, result)

		case req := <-h.markSync:
			h.handleMarkSync(ctx, req)

		case <-h.closeRequests:
			h.shutdown(context.Background())
			return

		case <-ctx.Done():
			h.shutdown(context.Background())
			logging.With("event_type", "hub_shutdown").Info("hub context canceled")
			return
		}
	}
}

func (h *Hub) Done() <-chan struct{} {
	return h.done
}

func (h *Hub) CloseAll() {
	select {
	case h.closeRequests <- struct{}{}:
	case <-h.done:
	}
}

func (h *Hub) EnqueueUnregister(client *Client) {
	if client == nil {
		return
	}
	select {
	case h.Unregister <- client:
	case <-h.done:
	}
}

func (h *Hub) MarkNeedsSync(userID uint64, connectionID string, conversationID uint64, reason string) {
	if userID == 0 || reason == "" {
		return
	}
	req := &syncRequest{
		UserID:         userID,
		ConnectionID:   connectionID,
		ConversationID: conversationID,
		Reason:         reason,
	}

	select {
	case h.markSync <- req:
	case <-h.done:
	}
}

func (h *Hub) handleRegister(ctx context.Context, client *Client) {
	if client == nil {
		return
	}
	if client.ConnectionID == "" {
		client.ConnectionID = fmt.Sprintf("conn-%p", client)
	}
	if client.Send == nil {
		client.Send = make(chan []byte, 256)
	}
	client.Hub = h
	client.Lifecycle = h.Lifecycle

	userConnections, ok := h.Clients[client.UserID]
	if !ok {
		userConnections = make(map[string]*hubConnectionState)
		h.Clients[client.UserID] = userConnections
	}
	firstConnection := len(userConnections) == 0
	userConnections[client.ConnectionID] = &hubConnectionState{
		client:  client,
		pending: make([][]byte, 0),
	}

	if firstConnection && h.Lifecycle != nil {
		h.Lifecycle.Connect(ctx, client.UserID)
	}

	go h.bootstrapClient(ctx, client)
	logging.With(
		"user_id", client.UserID,
		"connection_id", client.ConnectionID,
	).Info(
		"client registered",
		"user_count", len(h.Clients),
		"connection_count", h.connectionCount(),
	)
}

func (h *Hub) forwardLocal(ctx context.Context, msg *ForwardMessage) bool {
	if msg == nil {
		return false
	}

	userConnections, ok := h.Clients[msg.To]
	if !ok || len(userConnections) == 0 {
		return false
	}

	for _, state := range userConnections {
		h.forwardToConnection(ctx, state, msg)
	}
	return true
}

func closeClientConn(client *Client) {
	if client == nil || client.Conn == nil {
		return
	}
	_ = client.Conn.Close()
}

func (h *Hub) bootstrapClient(ctx context.Context, client *Client) {
	payloads := make([][]byte, 0)
	if h.Lifecycle != nil {
		offlinePayloads, err := h.Lifecycle.Bootstrap(ctx, client.UserID)
		if err != nil {
			logging.With("user_id", client.UserID, "connection_id", client.ConnectionID).Error("bootstrap client failed", "error", err)
		} else {
			payloads = offlinePayloads
		}
	}
	select {
	case h.ClientBootstrapped <- &ClientBootstrapResult{
		Client:          client,
		OfflineMessages: payloads,
	}:
	case <-h.done:
	case <-ctx.Done():
	}
}

func (h *Hub) handleBootstrapResult(ctx context.Context, result *ClientBootstrapResult) {
	if result == nil || result.Client == nil {
		return
	}
	state, ok := h.connectionState(result.Client.UserID, result.Client.ConnectionID)
	if !ok || state.client != result.Client || state.closed {
		return
	}

	h.flushMessages(ctx, state, result.OfflineMessages)
	h.flushMessages(ctx, state, state.pending)
	state.pending = nil
	state.ready = true
	if state.pendingSync != nil {
		h.emitSyncRequired(ctx, state)
	}
}

func (h *Hub) handleMarkSync(ctx context.Context, req *syncRequest) {
	if req == nil || req.UserID == 0 {
		return
	}

	if req.ConnectionID != "" {
		state, ok := h.connectionState(req.UserID, req.ConnectionID)
		if !ok {
			return
		}
		h.markConnectionNeedsSync(ctx, state, req.ConversationID, req.Reason)
		return
	}

	userConnections, ok := h.Clients[req.UserID]
	if !ok {
		return
	}
	for _, state := range userConnections {
		h.markConnectionNeedsSync(ctx, state, req.ConversationID, req.Reason)
	}
}

func (h *Hub) forwardToConnection(ctx context.Context, state *hubConnectionState, msg *ForwardMessage) {
	if state == nil || state.closed || msg == nil {
		return
	}
	if !state.ready {
		h.enqueuePending(ctx, state, msg)
		return
	}
	if h.pushPayload(state, msg.Content) {
		return
	}
	h.markConnectionNeedsSync(ctx, state, msg.ConversationID, SyncReasonSendQueueFull)
	logging.With(
		"user_id", state.client.UserID,
		"connection_id", state.client.ConnectionID,
	).Warn("send queue is full, mark connection needs sync")
}

func (h *Hub) enqueuePending(ctx context.Context, state *hubConnectionState, msg *ForwardMessage) {
	if len(state.pending) >= maxPendingPerConnection {
		h.markConnectionNeedsSync(ctx, state, msg.ConversationID, SyncReasonPendingQueueFull)
		logging.With(
			"user_id", state.client.UserID,
			"connection_id", state.client.ConnectionID,
		).Warn("pending queue is full, mark connection needs sync")
		return
	}
	state.pending = append(state.pending, msg.Content)
}

func (h *Hub) flushMessages(ctx context.Context, state *hubConnectionState, payloads [][]byte) {
	for _, payload := range payloads {
		if h.pushPayload(state, payload) {
			continue
		}
		h.markConnectionNeedsSync(ctx, state, 0, SyncReasonSendQueueFull)
		logging.With(
			"user_id", state.client.UserID,
			"connection_id", state.client.ConnectionID,
		).Warn("send queue is full while flushing bootstrap payloads")
		return
	}
}

func (h *Hub) pushPayload(state *hubConnectionState, payload []byte) bool {
	if state == nil || state.closed || state.client == nil {
		return false
	}
	select {
	case state.client.Send <- payload:
		return true
	default:
		return false
	}
}

func (h *Hub) markConnectionNeedsSync(ctx context.Context, state *hubConnectionState, conversationID uint64, reason string) {
	if state == nil || state.closed || reason == "" {
		return
	}
	if state.pendingSync == nil {
		state.pendingSync = &SyncRequiredData{
			ConversationID: conversationID,
			Reason:         reason,
		}
	}
	if state.ready {
		h.emitSyncRequired(ctx, state)
	}
}

func (h *Hub) emitSyncRequired(ctx context.Context, state *hubConnectionState) {
	if state == nil || state.closed || state.pendingSync == nil {
		return
	}

	payload, err := MarshalEnvelope(EventTypeSyncRequired, *state.pendingSync)
	if err != nil {
		logging.With(
			"user_id", state.client.UserID,
			"connection_id", state.client.ConnectionID,
		).Error("marshal sync required payload failed", "error", err)
		state.pendingSync = nil
		return
	}
	if h.pushPayload(state, payload) {
		state.pendingSync = nil
		return
	}

	logging.With(
		"user_id", state.client.UserID,
		"connection_id", state.client.ConnectionID,
	).Warn("sync required payload dropped: send queue full, close connection")
	h.removeConnection(ctx, state.client)
}

func (h *Hub) removeConnection(ctx context.Context, client *Client) {
	if client == nil {
		return
	}

	userConnections, ok := h.Clients[client.UserID]
	if !ok {
		return
	}
	state, ok := userConnections[client.ConnectionID]
	if !ok || state.client != client {
		return
	}

	h.closeConnectionState(state)
	delete(userConnections, client.ConnectionID)
	if len(userConnections) == 0 {
		delete(h.Clients, client.UserID)
		if h.Lifecycle != nil {
			h.Lifecycle.Disconnect(ctx, client.UserID)
		}
	}

	logging.With(
		"user_id", client.UserID,
		"connection_id", client.ConnectionID,
	).Info(
		"client unregistered",
		"user_count", len(h.Clients),
		"connection_count", h.connectionCount(),
	)
}

func (h *Hub) closeConnectionState(state *hubConnectionState) {
	if state == nil || state.closed {
		return
	}

	state.closed = true
	state.ready = false
	state.pending = nil
	state.pendingSync = nil

	if state.client != nil {
		state.client.closed.Store(true)
		closeClientConn(state.client)
		close(state.client.Send)
	}
}

func (h *Hub) shutdown(ctx context.Context) {
	userIDs := make([]uint64, 0, len(h.Clients))
	for userID, userConnections := range h.Clients {
		userIDs = append(userIDs, userID)
		for connectionID, state := range userConnections {
			h.closeConnectionState(state)
			delete(userConnections, connectionID)
		}
		delete(h.Clients, userID)
	}

	for _, userID := range userIDs {
		if h.Lifecycle != nil {
			h.Lifecycle.Disconnect(ctx, userID)
		}
	}
}

func (h *Hub) connectionState(userID uint64, connectionID string) (*hubConnectionState, bool) {
	userConnections, ok := h.Clients[userID]
	if !ok {
		return nil, false
	}
	state, ok := userConnections[connectionID]
	return state, ok
}

func (h *Hub) connectionCount() int {
	count := 0
	for _, userConnections := range h.Clients {
		count += len(userConnections)
	}
	return count
}
