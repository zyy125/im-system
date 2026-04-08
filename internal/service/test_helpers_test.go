package service

import (
	"context"

	"github.com/zyy125/im-system/internal/model"
	"github.com/zyy125/im-system/internal/repository"
)

type stubUserRepo struct {
	createFn        func(ctx context.Context, user *model.User) error
	getByUsernameFn func(ctx context.Context, username string) (model.User, error)
	getByIDFn       func(ctx context.Context, id uint64) (model.User, error)
	listByIDsFn     func(ctx context.Context, ids []uint64) ([]model.User, error)
}

func (s *stubUserRepo) Create(ctx context.Context, user *model.User) error {
	if s.createFn != nil {
		return s.createFn(ctx, user)
	}
	return nil
}

func (s *stubUserRepo) GetByUsername(ctx context.Context, username string) (model.User, error) {
	if s.getByUsernameFn != nil {
		return s.getByUsernameFn(ctx, username)
	}
	return model.User{}, nil
}

func (s *stubUserRepo) GetByID(ctx context.Context, id uint64) (model.User, error) {
	if s.getByIDFn != nil {
		return s.getByIDFn(ctx, id)
	}
	return model.User{}, nil
}

func (s *stubUserRepo) ListByIDs(ctx context.Context, ids []uint64) ([]model.User, error) {
	if s.listByIDsFn != nil {
		return s.listByIDsFn(ctx, ids)
	}
	return []model.User{}, nil
}

type stubPresenceRepo struct {
	setOnlineFn  func(ctx context.Context, userID uint64) error
	setOfflineFn func(ctx context.Context, userID uint64) error
	isOnlineFn   func(ctx context.Context, userID uint64) (bool, error)
}

func (s *stubPresenceRepo) SetOnline(ctx context.Context, userID uint64) error {
	if s.setOnlineFn != nil {
		return s.setOnlineFn(ctx, userID)
	}
	return nil
}

func (s *stubPresenceRepo) SetOffline(ctx context.Context, userID uint64) error {
	if s.setOfflineFn != nil {
		return s.setOfflineFn(ctx, userID)
	}
	return nil
}

func (s *stubPresenceRepo) IsOnline(ctx context.Context, userID uint64) (bool, error) {
	if s.isOnlineFn != nil {
		return s.isOnlineFn(ctx, userID)
	}
	return false, nil
}

type stubFriendRepo struct {
	addPairFn       func(ctx context.Context, userID, friendID uint64) error
	removePairFn    func(ctx context.Context, userID, friendID uint64) error
	areFriendsFn    func(ctx context.Context, userID, friendID uint64) (bool, error)
	listFriendIDsFn func(ctx context.Context, userID uint64) ([]uint64, error)
}

func (s *stubFriendRepo) AddPair(ctx context.Context, userID, friendID uint64) error {
	if s.addPairFn != nil {
		return s.addPairFn(ctx, userID, friendID)
	}
	return nil
}

func (s *stubFriendRepo) RemovePair(ctx context.Context, userID, friendID uint64) error {
	if s.removePairFn != nil {
		return s.removePairFn(ctx, userID, friendID)
	}
	return nil
}

func (s *stubFriendRepo) AreFriends(ctx context.Context, userID, friendID uint64) (bool, error) {
	if s.areFriendsFn != nil {
		return s.areFriendsFn(ctx, userID, friendID)
	}
	return false, nil
}

func (s *stubFriendRepo) ListFriendIDs(ctx context.Context, userID uint64) ([]uint64, error) {
	if s.listFriendIDsFn != nil {
		return s.listFriendIDsFn(ctx, userID)
	}
	return []uint64{}, nil
}

type stubConversationRepo struct {
	getOrCreateSingleFn       func(ctx context.Context, userA, userB uint64) (model.Conversation, error)
	getSingleFn               func(ctx context.Context, userA, userB uint64) (model.Conversation, error)
	listMembersByUserFn       func(ctx context.Context, userID uint64) ([]model.ConversationMember, error)
	listConversationsByUserFn func(ctx context.Context, userID uint64) ([]model.Conversation, error)
	getMemberFn               func(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error)
	ensureMemberFn            func(ctx context.Context, conversationID, userID uint64) error
	setVisibleFn              func(ctx context.Context, conversationID, userID uint64, visible bool) error
	updateLastDeliveredFn     func(ctx context.Context, conversationID, userID, msgSeq uint64) error
	updateLastReadFn          func(ctx context.Context, conversationID, userID, msgSeq uint64) error
}

func (s *stubConversationRepo) GetOrCreateSingle(ctx context.Context, userA, userB uint64) (model.Conversation, error) {
	if s.getOrCreateSingleFn != nil {
		return s.getOrCreateSingleFn(ctx, userA, userB)
	}
	return model.Conversation{}, nil
}

func (s *stubConversationRepo) GetSingle(ctx context.Context, userA, userB uint64) (model.Conversation, error) {
	if s.getSingleFn != nil {
		return s.getSingleFn(ctx, userA, userB)
	}
	return model.Conversation{}, nil
}

func (s *stubConversationRepo) ListMembersByUser(ctx context.Context, userID uint64) ([]model.ConversationMember, error) {
	if s.listMembersByUserFn != nil {
		return s.listMembersByUserFn(ctx, userID)
	}
	return []model.ConversationMember{}, nil
}

func (s *stubConversationRepo) ListConversationsByUser(ctx context.Context, userID uint64) ([]model.Conversation, error) {
	if s.listConversationsByUserFn != nil {
		return s.listConversationsByUserFn(ctx, userID)
	}
	return []model.Conversation{}, nil
}

func (s *stubConversationRepo) GetMember(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
	if s.getMemberFn != nil {
		return s.getMemberFn(ctx, conversationID, userID)
	}
	return model.ConversationMember{}, nil
}

func (s *stubConversationRepo) EnsureMember(ctx context.Context, conversationID, userID uint64) error {
	if s.ensureMemberFn != nil {
		return s.ensureMemberFn(ctx, conversationID, userID)
	}
	return nil
}

func (s *stubConversationRepo) SetVisible(ctx context.Context, conversationID, userID uint64, visible bool) error {
	if s.setVisibleFn != nil {
		return s.setVisibleFn(ctx, conversationID, userID, visible)
	}
	return nil
}

func (s *stubConversationRepo) UpdateLastDeliveredMsgSeq(ctx context.Context, conversationID, userID, msgSeq uint64) error {
	if s.updateLastDeliveredFn != nil {
		return s.updateLastDeliveredFn(ctx, conversationID, userID, msgSeq)
	}
	return nil
}

func (s *stubConversationRepo) UpdateLastReadMsgSeq(ctx context.Context, conversationID, userID, msgSeq uint64) error {
	if s.updateLastReadFn != nil {
		return s.updateLastReadFn(ctx, conversationID, userID, msgSeq)
	}
	return nil
}

type stubMessageRepo struct {
	createFn                         func(ctx context.Context, msg *model.ChatMessage) error
	listBetweenFn                    func(ctx context.Context, userID, peerID uint64, limit int, beforeID uint64) ([]model.ChatMessage, bool, error)
	listConversationPendingFn        func(ctx context.Context, conversationID string, afterSeq, untilSeq uint64) ([]model.ChatMessage, error)
	listConversationPendingForUserFn func(ctx context.Context, conversationID string, userID, afterSeq, untilSeq uint64) ([]model.ChatMessage, error)
	getLatestByConversationFn        func(ctx context.Context, conversationID string) (model.ChatMessage, error)
	countUnreadFn                    func(ctx context.Context, conversationID string, userID uint64, afterSeq uint64) (int64, error)
	getByConversationMsgIDFn         func(ctx context.Context, conversationID, msgID string) (model.ChatMessage, error)
}

func (s *stubMessageRepo) Create(ctx context.Context, msg *model.ChatMessage) error {
	if s.createFn != nil {
		return s.createFn(ctx, msg)
	}
	return nil
}

func (s *stubMessageRepo) ListBetween(ctx context.Context, userID, peerID uint64, limit int, beforeID uint64) ([]model.ChatMessage, bool, error) {
	if s.listBetweenFn != nil {
		return s.listBetweenFn(ctx, userID, peerID, limit, beforeID)
	}
	return []model.ChatMessage{}, false, nil
}

func (s *stubMessageRepo) ListConversationPending(ctx context.Context, conversationID string, afterSeq, untilSeq uint64) ([]model.ChatMessage, error) {
	if s.listConversationPendingFn != nil {
		return s.listConversationPendingFn(ctx, conversationID, afterSeq, untilSeq)
	}
	return []model.ChatMessage{}, nil
}

func (s *stubMessageRepo) ListConversationPendingForUser(ctx context.Context, conversationID string, userID, afterSeq, untilSeq uint64) ([]model.ChatMessage, error) {
	if s.listConversationPendingForUserFn != nil {
		return s.listConversationPendingForUserFn(ctx, conversationID, userID, afterSeq, untilSeq)
	}
	return []model.ChatMessage{}, nil
}

func (s *stubMessageRepo) GetLatestByConversation(ctx context.Context, conversationID string) (model.ChatMessage, error) {
	if s.getLatestByConversationFn != nil {
		return s.getLatestByConversationFn(ctx, conversationID)
	}
	return model.ChatMessage{}, nil
}

func (s *stubMessageRepo) CountUnreadByConversation(ctx context.Context, conversationID string, userID uint64, afterSeq uint64) (int64, error) {
	if s.countUnreadFn != nil {
		return s.countUnreadFn(ctx, conversationID, userID, afterSeq)
	}
	return 0, nil
}

func (s *stubMessageRepo) GetByConversationAndMsgID(ctx context.Context, conversationID, msgID string) (model.ChatMessage, error) {
	if s.getByConversationMsgIDFn != nil {
		return s.getByConversationMsgIDFn(ctx, conversationID, msgID)
	}
	return model.ChatMessage{}, nil
}

type stubFriendRequestRepo struct {
	createFn              func(ctx context.Context, req *model.FriendRequest) error
	getByIDFn             func(ctx context.Context, id uint64) (model.FriendRequest, error)
	findPendingBetweenFn  func(ctx context.Context, requesterID, receiverID uint64) (model.FriendRequest, error)
	listIncomingPendingFn func(ctx context.Context, userID uint64) ([]model.FriendRequest, error)
	listOutgoingPendingFn func(ctx context.Context, userID uint64) ([]model.FriendRequest, error)
	updateStatusFn        func(ctx context.Context, id uint64, status model.FriendRequestStatus) error
	resolvePendingFn      func(ctx context.Context, userA, userB uint64, status model.FriendRequestStatus) error
}

func (s *stubFriendRequestRepo) Create(ctx context.Context, req *model.FriendRequest) error {
	if s.createFn != nil {
		return s.createFn(ctx, req)
	}
	return nil
}

func (s *stubFriendRequestRepo) GetByID(ctx context.Context, id uint64) (model.FriendRequest, error) {
	if s.getByIDFn != nil {
		return s.getByIDFn(ctx, id)
	}
	return model.FriendRequest{}, nil
}

func (s *stubFriendRequestRepo) FindPendingBetween(ctx context.Context, requesterID, receiverID uint64) (model.FriendRequest, error) {
	if s.findPendingBetweenFn != nil {
		return s.findPendingBetweenFn(ctx, requesterID, receiverID)
	}
	return model.FriendRequest{}, nil
}

func (s *stubFriendRequestRepo) ListIncomingPending(ctx context.Context, userID uint64) ([]model.FriendRequest, error) {
	if s.listIncomingPendingFn != nil {
		return s.listIncomingPendingFn(ctx, userID)
	}
	return []model.FriendRequest{}, nil
}

func (s *stubFriendRequestRepo) ListOutgoingPending(ctx context.Context, userID uint64) ([]model.FriendRequest, error) {
	if s.listOutgoingPendingFn != nil {
		return s.listOutgoingPendingFn(ctx, userID)
	}
	return []model.FriendRequest{}, nil
}

func (s *stubFriendRequestRepo) UpdateStatus(ctx context.Context, id uint64, status model.FriendRequestStatus) error {
	if s.updateStatusFn != nil {
		return s.updateStatusFn(ctx, id, status)
	}
	return nil
}

func (s *stubFriendRequestRepo) ResolvePendingBetween(ctx context.Context, userA, userB uint64, status model.FriendRequestStatus) error {
	if s.resolvePendingFn != nil {
		return s.resolvePendingFn(ctx, userA, userB, status)
	}
	return nil
}

var (
	_ repository.UserRepo          = (*stubUserRepo)(nil)
	_ repository.PresenceRepo      = (*stubPresenceRepo)(nil)
	_ repository.FriendRepo        = (*stubFriendRepo)(nil)
	_ repository.ConversationRepo  = (*stubConversationRepo)(nil)
	_ repository.MessageRepo       = (*stubMessageRepo)(nil)
	_ repository.FriendRequestRepo = (*stubFriendRequestRepo)(nil)
)

type stubMessageTxManager struct {
	withinFn         func(ctx context.Context, fn func(messageRepo repository.MessageRepo, conversationRepo repository.ConversationRepo) error) error
	messageRepo      repository.MessageRepo
	conversationRepo repository.ConversationRepo
}

func (s *stubMessageTxManager) WithinMessageTx(ctx context.Context, fn func(messageRepo repository.MessageRepo, conversationRepo repository.ConversationRepo) error) error {
	if s.withinFn != nil {
		return s.withinFn(ctx, fn)
	}
	return fn(s.messageRepo, s.conversationRepo)
}

var _ repository.MessageTxManager = (*stubMessageTxManager)(nil)
