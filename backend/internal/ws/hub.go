package ws

import (
	"context"
	"fmt"

	"github.com/zyy125/im-system/internal/logging"
	"github.com/zyy125/im-system/internal/repository"
	"github.com/zyy125/im-system/internal/service"
)

const maxPendingPerConnection = 512

const (
	CloseReasonClientReadStopped        = "client_read_stopped"
	CloseReasonSyncRequiredDeliveryFail = "sync_required_delivery_failed"
	CloseReasonHubShutdown              = "hub_shutdown"
	CloseReasonUnknown                  = "unknown"
)

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

type unregisterRequest struct {
	client *Client
	reason string
}

type syncRequest struct {
	userID         uint64
	conversationID uint64
	reason         string
}

// clientConnection 表示 Hub 内部维护的一条客户端连接及其运行时投递状态。
// 它同时持有连接实体和 ready/pending/sync 等控制信息，便于 Hub 串行管理投递过程。
type clientConnection struct {
	client      *Client
	ready       bool
	closed      bool
	pending     [][]byte
	pendingSync *SyncRequiredData
	closeReason string
}

type Hub struct {
	Clients            map[uint64]map[string]*clientConnection
	Register           chan *Client
	Unregister         chan *unregisterRequest
	Forward            chan *ForwardMessage
	LifecycleForward   chan *ForwardMessage
	ClientBootstrapped chan *ClientBootstrapResult

	Lifecycle     ClientLifecycle
	markSync      chan *syncRequest
	closeRequests chan struct{}
	done          chan struct{}
}

func NewHub(
	presenceRepo repository.PresenceRepo,
	conversationService service.ConversationService,
	friendRepo repository.FriendRepo,
	userService service.UserService,
) *Hub {
	lifecycleForward := make(chan *ForwardMessage, 256)
	hub := &Hub{
		Clients:            make(map[uint64]map[string]*clientConnection),
		Register:           make(chan *Client, 32),
		Unregister:         make(chan *unregisterRequest, 64),
		Forward:            make(chan *ForwardMessage, 512),
		LifecycleForward:   lifecycleForward,
		ClientBootstrapped: make(chan *ClientBootstrapResult, 32),
		markSync:           make(chan *syncRequest, 256),
		closeRequests:      make(chan struct{}, 1),
		done:               make(chan struct{}),
	}
	hub.Lifecycle = NewClientLifecycle(
		presenceRepo,
		conversationService,
		friendRepo,
		userService,
		hub,
	)
	return hub
}

func (h *Hub) Run(ctx context.Context) {
	defer close(h.done)

	for {
		select {
		case client := <-h.Register:
			h.handleRegister(ctx, client)

		case req := <-h.Unregister:
			h.removeConnection(ctx, req.client, req.reason)

		case msg := <-h.Forward:
			h.deliverForward(ctx, msg)

		case msg := <-h.LifecycleForward:
			h.deliverForward(ctx, msg)

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
	h.EnqueueUnregisterWithReason(client, CloseReasonClientReadStopped)
}

func (h *Hub) EnqueueUnregisterWithReason(client *Client, reason string) {
	if client == nil {
		return
	}
	select {
	case h.Unregister <- &unregisterRequest{
		client: client,
		reason: normalizeCloseReason(reason),
	}:
	case <-h.done:
	}
}

// EnqueueUserSync 将用户级 sync 请求投递回 Hub 主循环，保持连接状态修改仍由 Hub 串行处理。
func (h *Hub) EnqueueUserSync(userID uint64, conversationID uint64, reason string) {
	if userID == 0 || reason == "" {
		return
	}
	req := &syncRequest{
		userID:         userID,
		conversationID: conversationID,
		reason:         reason,
	}

	select {
	case h.markSync <- req:
	case <-h.done:
	}
}

// EnqueueForwards 是 Hub 统一的批量投递入口，避免 Client 直接处理 Hub 内部投递细节。
func (h *Hub) EnqueueForwards(ctx context.Context, forwardMsgs []*ForwardMessage) {
	for _, forwardMsg := range forwardMsgs {
		select {
		case h.Forward <- forwardMsg:
		default:
			h.EnqueueUserSync(forwardMsg.To, forwardMsg.ConversationID, SyncReasonForwardQueueFull)
			logging.FromContext(ctx).With("target_user_id", forwardMsg.To).Warn("forward queue full, mark connection needs sync")
		}
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
		userConnections = make(map[string]*clientConnection)
		h.Clients[client.UserID] = userConnections
	}
	firstConnection := len(userConnections) == 0
	userConnections[client.ConnectionID] = &clientConnection{
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

// deliverForward 按目标用户找到全部在线连接，并对每条连接应用统一的投递策略。
func (h *Hub) deliverForward(ctx context.Context, msg *ForwardMessage) bool {
	if msg == nil {
		return false
	}

	userConnections, ok := h.Clients[msg.To]
	if !ok || len(userConnections) == 0 {
		return false
	}

	for _, conn := range userConnections {
		h.deliverToConnection(ctx, conn, msg)
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
	conn, ok := h.findConnection(result.Client.UserID, result.Client.ConnectionID)
	if !ok || conn.client != result.Client || conn.closed {
		return
	}

	h.flushMessages(ctx, conn, result.OfflineMessages)
	h.flushMessages(ctx, conn, conn.pending)
	conn.pending = nil
	conn.ready = true
	if conn.pendingSync != nil {
		h.emitSyncRequired(ctx, conn)
	}
}

func (h *Hub) handleMarkSync(ctx context.Context, req *syncRequest) {
	if req == nil || req.userID == 0 || req.reason == "" {
		return
	}

	userConnections, ok := h.Clients[req.userID]
	if !ok {
		return
	}
	for _, conn := range userConnections {
		h.requireConnectionSync(ctx, conn, req.conversationID, req.reason)
	}
}

// deliverToConnection 负责处理单条连接的 ready/pending/sync 降级逻辑。
func (h *Hub) deliverToConnection(ctx context.Context, conn *clientConnection, msg *ForwardMessage) {
	if conn == nil || conn.closed || msg == nil {
		return
	}
	if !conn.ready {
		h.enqueuePending(ctx, conn, msg)
		return
	}
	if h.pushPayload(conn, msg.Content) {
		return
	}
	h.requireConnectionSync(ctx, conn, msg.ConversationID, SyncReasonSendQueueFull)
	logging.With(
		"user_id", conn.client.UserID,
		"connection_id", conn.client.ConnectionID,
	).Warn("send queue is full, mark connection needs sync")
}

func (h *Hub) enqueuePending(ctx context.Context, conn *clientConnection, msg *ForwardMessage) {
	if len(conn.pending) >= maxPendingPerConnection {
		h.requireConnectionSync(ctx, conn, msg.ConversationID, SyncReasonPendingQueueFull)
		logging.With(
			"user_id", conn.client.UserID,
			"connection_id", conn.client.ConnectionID,
		).Warn("pending queue is full, mark connection needs sync")
		return
	}
	conn.pending = append(conn.pending, msg.Content)
}

// flushMessages 用于补发 bootstrap 阶段积压的消息；一旦补发失败，说明这条连接的实时流已不再可靠，转入 sync 补偿。
func (h *Hub) flushMessages(ctx context.Context, conn *clientConnection, payloads [][]byte) {
	for _, payload := range payloads {
		if h.pushPayload(conn, payload) {
			continue
		}
		h.requireConnectionSync(ctx, conn, 0, SyncReasonSendQueueFull)
		logging.With(
			"user_id", conn.client.UserID,
			"connection_id", conn.client.ConnectionID,
		).Warn("send queue is full while flushing bootstrap payloads")
		return
	}
}

func (h *Hub) pushPayload(conn *clientConnection, payload []byte) bool {
	if conn == nil || conn.closed || conn.client == nil {
		return false
	}
	select {
	case conn.client.Send <- payload:
		return true
	default:
		return false
	}
}

func (h *Hub) requireConnectionSync(ctx context.Context, conn *clientConnection, conversationID uint64, reason string) {
	if conn == nil || conn.closed || reason == "" {
		return
	}
	if conn.pendingSync == nil {
		conn.pendingSync = &SyncRequiredData{
			ConversationID: conversationID,
			Reason:         reason,
		}
	}
	if conn.ready {
		h.emitSyncRequired(ctx, conn)
	}
}

func (h *Hub) emitSyncRequired(ctx context.Context, conn *clientConnection) {
	if conn == nil || conn.closed || conn.pendingSync == nil {
		return
	}

	payload, err := MarshalEnvelope(EventTypeSyncRequired, *conn.pendingSync)
	if err != nil {
		logging.With(
			"user_id", conn.client.UserID,
			"connection_id", conn.client.ConnectionID,
		).Error("marshal sync required payload failed", "error", err)
		conn.pendingSync = nil
		return
	}
	if h.pushPayload(conn, payload) {
		conn.pendingSync = nil
		return
	}

	logging.With(
		"user_id", conn.client.UserID,
		"connection_id", conn.client.ConnectionID,
	).Warn("sync required payload dropped: send queue full, close connection")
	h.removeConnection(ctx, conn.client, CloseReasonSyncRequiredDeliveryFail)
}

func (h *Hub) removeConnection(ctx context.Context, client *Client, reason string) {
	if client == nil {
		return
	}
	reason = normalizeCloseReason(reason)

	userConnections, ok := h.Clients[client.UserID]
	if !ok {
		return
	}
	conn, ok := userConnections[client.ConnectionID]
	if !ok || conn.client != client {
		return
	}

	h.closeConnection(conn, reason)
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
		"reason", reason,
	).Info(
		"client unregistered",
		"user_count", len(h.Clients),
		"connection_count", h.connectionCount(),
	)
}

// closeConnection 负责连接资源回收和运行时状态清理，避免后续继续投递到失效连接。
func (h *Hub) closeConnection(conn *clientConnection, reason string) {
	if conn == nil || conn.closed {
		return
	}

	conn.closed = true
	conn.ready = false
	conn.pending = nil
	conn.pendingSync = nil
	conn.closeReason = normalizeCloseReason(reason)

	if conn.client != nil {
		conn.client.closed.Store(true)
		closeClientConn(conn.client)
		close(conn.client.Send)
	}
}

func (h *Hub) shutdown(ctx context.Context) {
	userIDs := make([]uint64, 0, len(h.Clients))
	for userID, userConnections := range h.Clients {
		userIDs = append(userIDs, userID)
		for connectionID, conn := range userConnections {
			h.closeConnection(conn, CloseReasonHubShutdown)
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

func (h *Hub) findConnection(userID uint64, connectionID string) (*clientConnection, bool) {
	userConnections, ok := h.Clients[userID]
	if !ok {
		return nil, false
	}
	conn, ok := userConnections[connectionID]
	return conn, ok
}

func (h *Hub) connectionCount() int {
	count := 0
	for _, userConnections := range h.Clients {
		count += len(userConnections)
	}
	return count
}

func normalizeCloseReason(reason string) string {
	if reason == "" {
		return CloseReasonUnknown
	}
	return reason
}
