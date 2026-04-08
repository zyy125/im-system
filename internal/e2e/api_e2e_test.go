package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zyy125/im-system/config"
	"github.com/zyy125/im-system/internal/handler"
	"github.com/zyy125/im-system/internal/handler/dto"
	"github.com/zyy125/im-system/internal/model"
	"github.com/zyy125/im-system/internal/repository"
	"github.com/zyy125/im-system/internal/router"
	"github.com/zyy125/im-system/internal/service"
	"github.com/zyy125/im-system/internal/ws"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type apiResponse struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type inMemoryPresenceRepo struct {
	mu     sync.RWMutex
	online map[uint64]bool
}

func newInMemoryPresenceRepo() *inMemoryPresenceRepo {
	return &inMemoryPresenceRepo{online: make(map[uint64]bool)}
}

func (r *inMemoryPresenceRepo) SetOnline(_ context.Context, userID uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.online[userID] = true
	return nil
}

func (r *inMemoryPresenceRepo) SetOffline(_ context.Context, userID uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.online, userID)
	return nil
}

func (r *inMemoryPresenceRepo) IsOnline(_ context.Context, userID uint64) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.online[userID], nil
}

type inMemoryTokenBlacklistRepo struct {
	mu      sync.RWMutex
	blocked map[string]struct{}
}

func newInMemoryTokenBlacklistRepo() *inMemoryTokenBlacklistRepo {
	return &inMemoryTokenBlacklistRepo{blocked: make(map[string]struct{})}
}

func (r *inMemoryTokenBlacklistRepo) IsBlacklisted(_ context.Context, jti string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.blocked[jti]
	return ok, nil
}

func (r *inMemoryTokenBlacklistRepo) Blacklist(_ context.Context, jti string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.blocked[jti] = struct{}{}
	return nil
}

type testEnv struct {
	server *httptest.Server
	ctx    context.Context
	cancel context.CancelFunc
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&model.User{},
		&model.ChatMessage{},
		&model.Friend{},
		&model.FriendRequest{},
		&model.Conversation{},
		&model.ConversationMember{},
	)
	require.NoError(t, err)

	userRepo := repository.NewUserRepo(db)
	friendRepo := repository.NewFriendRepo(db)
	friendRequestRepo := repository.NewFriendRequestRepo(db)
	conversationRepo := repository.NewConversationRepo(db)
	messageRepo := repository.NewMessageRepo(db)
	messageTxManager := repository.NewMessageTxManager(db)
	presenceRepo := newInMemoryPresenceRepo()
	blacklistRepo := newInMemoryTokenBlacklistRepo()

	cfg := &config.Config{
		JWT: config.JWT{
			Secret: "test-secret",
			Expiry: 24,
		},
	}

	conversationSvc := service.NewConversationService(conversationRepo, messageRepo, userRepo, presenceRepo, friendRepo)
	friendSvc := service.NewFriendService(friendRepo, userRepo, presenceRepo, conversationRepo)
	messageSvc := service.NewMessageService(messageRepo, conversationRepo, messageTxManager)

	ctx, cancel := context.WithCancel(context.Background())
	hub := ws.NewHub(presenceRepo, conversationSvc, friendRepo)
	go hub.Run(ctx)

	engine := router.InitRouter(&router.InitRouterParams{
		AuthHandler:          handler.NewAuthHandler(service.NewAuthService(userRepo, &cfg.JWT, blacklistRepo)),
		WSHandler:            handler.NewWSHandler(hub, messageSvc, friendSvc, conversationSvc),
		UserHandler:          handler.NewUserHandler(service.NewUserService(userRepo, presenceRepo)),
		FriendHandler:        handler.NewFriendHandler(friendSvc),
		FriendRequestHandler: handler.NewFriendRequestHandler(service.NewFriendRequestService(friendRequestRepo, friendSvc, userRepo, presenceRepo)),
		MessageHandler:       handler.NewMessageHandler(messageSvc, friendSvc, conversationSvc),
		ConversationHandler:  handler.NewConversationHandler(conversationSvc),
		BlacklistRepo:        blacklistRepo,
		JwtCfg:               &cfg.JWT,
	})

	server := httptest.NewServer(engine)
	t.Cleanup(func() {
		cancel()
		server.Close()
	})

	return &testEnv{
		server: server,
		ctx:    ctx,
		cancel: cancel,
	}
}

func TestGolden_RegisterAndLogin(t *testing.T) {
	env := newTestEnv(t)

	userID, token := registerAndLogin(t, env, "alice", "secret123")
	assert.NotEmpty(t, token)
	assert.NotZero(t, userID)

	resp := doJSON(t, env, http.MethodGet, "/api/v1/users/me", token, nil)
	assert.Equal(t, "ok", resp.Code)

	var me dto.UserInfoResp
	decodeData(t, resp, &me)
	assert.Equal(t, userID, me.ID)
	assert.Equal(t, "alice", me.Username)
	assert.False(t, me.Online)
}

func TestGolden_FriendRequestAndAccept(t *testing.T) {
	env := newTestEnv(t)

	aliceID, aliceToken := registerAndLogin(t, env, "alice", "secret123")
	bobID, bobToken := registerAndLogin(t, env, "bob", "secret123")

	resp := doJSON(t, env, http.MethodPost, "/api/v1/friend-requests/"+uintToString(bobID), aliceToken, map[string]any{
		"message": "hi bob",
	})
	assert.Equal(t, "ok", resp.Code)

	incoming := doJSON(t, env, http.MethodGet, "/api/v1/friend-requests/incoming", bobToken, nil)
	var incomingBody dto.FriendRequestListResp
	decodeData(t, incoming, &incomingBody)
	require.Len(t, incomingBody.Requests, 1)
	assert.Equal(t, aliceID, incomingBody.Requests[0].Requester.ID)

	requestID := incomingBody.Requests[0].ID
	resp = doJSON(t, env, http.MethodPost, "/api/v1/friend-requests/"+uintToString(requestID)+"/accept", bobToken, nil)
	assert.Equal(t, "ok", resp.Code)

	aliceFriends := doJSON(t, env, http.MethodGet, "/api/v1/friends", aliceToken, nil)
	var aliceFriendList dto.FriendListResp
	decodeData(t, aliceFriends, &aliceFriendList)
	require.Len(t, aliceFriendList.Friends, 1)
	assert.Equal(t, bobID, aliceFriendList.Friends[0].UserID)

	bobFriends := doJSON(t, env, http.MethodGet, "/api/v1/friends", bobToken, nil)
	var bobFriendList dto.FriendListResp
	decodeData(t, bobFriends, &bobFriendList)
	require.Len(t, bobFriendList.Friends, 1)
	assert.Equal(t, aliceID, bobFriendList.Friends[0].UserID)

	aliceConversations := doJSON(t, env, http.MethodGet, "/api/v1/conversations", aliceToken, nil)
	var aliceConversationList dto.ConversationListResp
	decodeData(t, aliceConversations, &aliceConversationList)
	require.Len(t, aliceConversationList.Conversations, 1)
	assert.Equal(t, bobID, aliceConversationList.Conversations[0].Peer.ID)

	bobConversations := doJSON(t, env, http.MethodGet, "/api/v1/conversations", bobToken, nil)
	var bobConversationList dto.ConversationListResp
	decodeData(t, bobConversations, &bobConversationList)
	require.Len(t, bobConversationList.Conversations, 1)
	assert.Equal(t, aliceID, bobConversationList.Conversations[0].Peer.ID)
}

func TestGolden_OpenConversationAndSendMessage(t *testing.T) {
	env := newTestEnv(t)

	aliceID, aliceToken := registerAndLogin(t, env, "alice", "secret123")
	bobID, bobToken := registerAndLogin(t, env, "bob", "secret123")
	makeFriends(t, env, aliceToken, bobToken, bobID)

	openResp := doJSON(t, env, http.MethodPost, "/api/v1/conversations/direct/"+uintToString(bobID)+"/open", aliceToken, nil)
	var openBody dto.OpenConversationResp
	decodeData(t, openResp, &openBody)
	assert.Equal(t, bobID, openBody.Conversation.Peer.ID)

	aliceConn := openWebSocket(t, env, aliceToken)
	defer aliceConn.Close()

	bobConn := openWebSocket(t, env, bobToken)
	defer bobConn.Close()

	msgID := "msg-e2e-1"
	require.NoError(t, aliceConn.WriteJSON(map[string]any{
		"type":    ws.EventTypeChatSend,
		"version": ws.ProtocolVersion,
		"data": map[string]any{
			"msg_id":  msgID,
			"to":      bobID,
			"content": "hello bob",
		},
	}))

	delivered := readChatMessage(t, bobConn, msgID, 5*time.Second)
	assert.Equal(t, msgID, delivered.MsgID)
	assert.Equal(t, aliceID, delivered.From)
	assert.Equal(t, bobID, delivered.To)
	assert.Equal(t, "hello bob", delivered.Content)
	assert.Equal(t, uintToString(openBody.Conversation.ID), delivered.ConversationID)

	require.Eventually(t, func() bool {
		historyResp := doJSON(t, env, http.MethodGet, "/api/v1/messages/history?peer_id="+uintToString(bobID), aliceToken, nil)
		var history dto.MessageHistoryResp
		decodeData(t, historyResp, &history)
		if len(history.Messages) == 0 {
			return false
		}
		return history.Messages[0].MsgID == msgID
	}, 5*time.Second, 100*time.Millisecond)
}

func registerAndLogin(t *testing.T, env *testEnv, username, password string) (uint64, string) {
	t.Helper()

	resp := doJSON(t, env, http.MethodPost, "/api/v1/auth/register", "", map[string]any{
		"username": username,
		"password": password,
	})
	assert.Equal(t, "ok", resp.Code)

	resp = doJSON(t, env, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"username": username,
		"password": password,
	})
	assert.Equal(t, "ok", resp.Code)

	var login dto.UserLoginResp
	decodeData(t, resp, &login)
	require.NotEmpty(t, login.Token)

	me := doJSON(t, env, http.MethodGet, "/api/v1/users/me", login.Token, nil)
	var meBody dto.UserInfoResp
	decodeData(t, me, &meBody)
	return meBody.ID, login.Token
}

func makeFriends(t *testing.T, env *testEnv, requesterToken, receiverToken string, receiverID uint64) {
	t.Helper()

	resp := doJSON(t, env, http.MethodPost, "/api/v1/friend-requests/"+uintToString(receiverID), requesterToken, map[string]any{
		"message": "let's chat",
	})
	assert.Equal(t, "ok", resp.Code)

	incoming := doJSON(t, env, http.MethodGet, "/api/v1/friend-requests/incoming", receiverToken, nil)
	var body dto.FriendRequestListResp
	decodeData(t, incoming, &body)
	require.Len(t, body.Requests, 1)

	resp = doJSON(t, env, http.MethodPost, "/api/v1/friend-requests/"+uintToString(body.Requests[0].ID)+"/accept", receiverToken, nil)
	assert.Equal(t, "ok", resp.Code)
}

func openWebSocket(t *testing.T, env *testEnv, token string) *websocket.Conn {
	t.Helper()

	wsURL := "ws" + strings.TrimPrefix(env.server.URL, "http") + "/api/v1/ws/?token=" + url.QueryEscape(token)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	return conn
}

func readChatMessage(t *testing.T, conn *websocket.Conn, wantMsgID string, timeout time.Duration) model.ChatMessage {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		require.NoError(t, conn.SetReadDeadline(deadline))
		_, payload, err := conn.ReadMessage()
		require.NoError(t, err)

		var env ws.Envelope
		require.NoError(t, json.Unmarshal(payload, &env))
		if env.Type != ws.EventTypeChatMessage {
			continue
		}

		var msg model.ChatMessage
		require.NoError(t, json.Unmarshal(env.Data, &msg))
		if msg.MsgID == wantMsgID {
			return msg
		}
	}

	t.Fatalf("chat message %s not received before timeout", wantMsgID)
	return model.ChatMessage{}
}

func doJSON(t *testing.T, env *testEnv, method, path, token string, body any) apiResponse {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(env.ctx, method, env.server.URL+path, bodyReader)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := env.server.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Less(t, resp.StatusCode, 500, string(raw))

	var result apiResponse
	require.NoError(t, json.Unmarshal(raw, &result), string(raw))
	return result
}

func decodeData(t *testing.T, resp apiResponse, dst any) {
	t.Helper()
	if len(resp.Data) == 0 || string(resp.Data) == "null" {
		return
	}
	require.NoError(t, json.Unmarshal(resp.Data, dst))
}

func uintToString(value uint64) string {
	return strconv.FormatUint(value, 10)
}
