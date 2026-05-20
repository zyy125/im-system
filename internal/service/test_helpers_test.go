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
	items := make([]model.User, 0, len(ids))
	for _, id := range ids {
		if s.getByIDFn == nil {
			continue
		}
		user, err := s.getByIDFn(ctx, id)
		if err != nil {
			return nil, err
		}
		items = append(items, user)
	}
	return items, nil
}

type stubPresenceRepo struct {
	setOnlineFn      func(ctx context.Context, userID uint64) error
	refreshOnlineFn  func(ctx context.Context, userID uint64) error
	setOfflineFn     func(ctx context.Context, userID uint64) error
	isOnlineFn       func(ctx context.Context, userID uint64) (bool, error)
	batchGetOnlineFn func(ctx context.Context, userIDs []uint64) (map[uint64]bool, error)
}

func (s *stubPresenceRepo) SetOnline(ctx context.Context, userID uint64) error {
	if s.setOnlineFn != nil {
		return s.setOnlineFn(ctx, userID)
	}
	return nil
}

func (s *stubPresenceRepo) RefreshOnline(ctx context.Context, userID uint64) error {
	if s.refreshOnlineFn != nil {
		return s.refreshOnlineFn(ctx, userID)
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

func (s *stubPresenceRepo) BatchGetOnline(ctx context.Context, userIDs []uint64) (map[uint64]bool, error) {
	if s.batchGetOnlineFn != nil {
		return s.batchGetOnlineFn(ctx, userIDs)
	}
	result := make(map[uint64]bool, len(userIDs))
	for _, userID := range userIDs {
		online, err := s.IsOnline(ctx, userID)
		if err != nil {
			return nil, err
		}
		result[userID] = online
	}
	return result, nil
}

type stubFriendRepo struct {
	addPairFn            func(ctx context.Context, userID, friendID, conversationID uint64) error
	removePairFn         func(ctx context.Context, userID, friendID uint64) error
	areFriendsFn         func(ctx context.Context, userID, friendID uint64) (bool, error)
	listFriendIDsFn      func(ctx context.Context, userID uint64) ([]uint64, error)
	listFriendProfilesFn func(ctx context.Context, userID uint64) ([]repository.FriendProfile, error)
}

func (s *stubFriendRepo) AddPair(ctx context.Context, userID, friendID, conversationID uint64) error {
	if s.addPairFn != nil {
		return s.addPairFn(ctx, userID, friendID, conversationID)
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

func (s *stubFriendRepo) ListFriendProfiles(ctx context.Context, userID uint64) ([]repository.FriendProfile, error) {
	if s.listFriendProfilesFn != nil {
		return s.listFriendProfilesFn(ctx, userID)
	}
	return []repository.FriendProfile{}, nil
}

type stubConversationRepo struct {
	createFn                  func(ctx context.Context, conversation *model.Conversation) error
	getByIDFn                 func(ctx context.Context, conversationID uint64) (model.Conversation, error)
	getOrCreateSingleFn       func(ctx context.Context, userA, userB uint64) (model.Conversation, error)
	listMembersByUserFn       func(ctx context.Context, userID uint64) ([]model.ConversationMember, error)
	listConversationsByUserFn func(ctx context.Context, userID uint64) ([]model.Conversation, error)
	listActiveGroupsByUserFn  func(ctx context.Context, userID uint64) ([]model.Conversation, error)
	listActiveMembersFn       func(ctx context.Context, conversationID uint64) ([]model.ConversationMember, error)
	countActiveMembersFn      func(ctx context.Context, conversationID uint64) (int64, error)
	getMemberFn               func(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error)
	upsertMemberFn            func(ctx context.Context, member *model.ConversationMember) error
	setVisibleFn              func(ctx context.Context, conversationID, userID uint64, visible bool) error
	setVisibleForUsersFn      func(ctx context.Context, conversationID uint64, userIDs []uint64, visible bool) error
	updateNameFn              func(ctx context.Context, conversationID uint64, name string) error
	updateStatusFn            func(ctx context.Context, conversationID uint64, status model.ConversationStatus) error
	updateMemberStatusFn      func(ctx context.Context, conversationID, userID uint64, status model.ConversationMemberStatus, visible bool) error
	updateLastAckedFn         func(ctx context.Context, conversationID, userID, msgSeq uint64) error
	updateLastReadFn          func(ctx context.Context, conversationID, userID, msgSeq uint64) error
}

func (s *stubConversationRepo) Create(ctx context.Context, conversation *model.Conversation) error {
	if s.createFn != nil {
		return s.createFn(ctx, conversation)
	}
	return nil
}

func (s *stubConversationRepo) GetByID(ctx context.Context, conversationID uint64) (model.Conversation, error) {
	if s.getByIDFn != nil {
		return s.getByIDFn(ctx, conversationID)
	}
	return model.Conversation{}, nil
}

func (s *stubConversationRepo) GetOrCreateSingle(ctx context.Context, userA, userB uint64) (model.Conversation, error) {
	if s.getOrCreateSingleFn != nil {
		return s.getOrCreateSingleFn(ctx, userA, userB)
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

func (s *stubConversationRepo) ListActiveGroupsByUser(ctx context.Context, userID uint64) ([]model.Conversation, error) {
	if s.listActiveGroupsByUserFn != nil {
		return s.listActiveGroupsByUserFn(ctx, userID)
	}
	return []model.Conversation{}, nil
}

func (s *stubConversationRepo) ListActiveMembers(ctx context.Context, conversationID uint64) ([]model.ConversationMember, error) {
	if s.listActiveMembersFn != nil {
		return s.listActiveMembersFn(ctx, conversationID)
	}
	return []model.ConversationMember{}, nil
}

func (s *stubConversationRepo) CountActiveMembers(ctx context.Context, conversationID uint64) (int64, error) {
	if s.countActiveMembersFn != nil {
		return s.countActiveMembersFn(ctx, conversationID)
	}
	return 0, nil
}

func (s *stubConversationRepo) GetMember(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
	if s.getMemberFn != nil {
		return s.getMemberFn(ctx, conversationID, userID)
	}
	return model.ConversationMember{}, nil
}

func (s *stubConversationRepo) UpsertMember(ctx context.Context, member *model.ConversationMember) error {
	if s.upsertMemberFn != nil {
		return s.upsertMemberFn(ctx, member)
	}
	return nil
}

func (s *stubConversationRepo) SetVisible(ctx context.Context, conversationID, userID uint64, visible bool) error {
	if s.setVisibleFn != nil {
		return s.setVisibleFn(ctx, conversationID, userID, visible)
	}
	return nil
}

func (s *stubConversationRepo) SetVisibleForUsers(ctx context.Context, conversationID uint64, userIDs []uint64, visible bool) error {
	if s.setVisibleForUsersFn != nil {
		return s.setVisibleForUsersFn(ctx, conversationID, userIDs, visible)
	}
	return nil
}

func (s *stubConversationRepo) UpdateName(ctx context.Context, conversationID uint64, name string) error {
	if s.updateNameFn != nil {
		return s.updateNameFn(ctx, conversationID, name)
	}
	return nil
}

func (s *stubConversationRepo) UpdateStatus(ctx context.Context, conversationID uint64, status model.ConversationStatus) error {
	if s.updateStatusFn != nil {
		return s.updateStatusFn(ctx, conversationID, status)
	}
	return nil
}

func (s *stubConversationRepo) UpdateMemberStatus(ctx context.Context, conversationID, userID uint64, status model.ConversationMemberStatus, visible bool) error {
	if s.updateMemberStatusFn != nil {
		return s.updateMemberStatusFn(ctx, conversationID, userID, status, visible)
	}
	return nil
}

func (s *stubConversationRepo) UpdateAllMemberStatus(ctx context.Context, conversationID uint64, status model.ConversationMemberStatus, visible bool) error {
	return nil
}

func (s *stubConversationRepo) UpdateLastAckedMsgSeq(ctx context.Context, conversationID, userID, msgSeq uint64) error {
	if s.updateLastAckedFn != nil {
		return s.updateLastAckedFn(ctx, conversationID, userID, msgSeq)
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
	createFn                   func(ctx context.Context, msg *model.Message) error
	listConversationHistoryFn  func(ctx context.Context, conversationID uint64, limit int, beforeSeq, afterSeq uint64) ([]model.Message, bool, error)
	listConversationAfterSeqFn func(ctx context.Context, conversationID, afterSeq uint64, limit int) ([]model.Message, bool, error)
	listConversationRangeFn    func(ctx context.Context, conversationID, afterSeq, untilSeq uint64, limit int) ([]model.Message, bool, error)
	getLatestByConversationFn  func(ctx context.Context, conversationID uint64) (model.Message, error)
	getMaxSeqByConversationFn  func(ctx context.Context, conversationID uint64) (uint64, error)
	countUnreadFn              func(ctx context.Context, conversationID, userID, afterSeq uint64) (int64, error)
}

func (s *stubMessageRepo) Create(ctx context.Context, msg *model.Message) error {
	if s.createFn != nil {
		return s.createFn(ctx, msg)
	}
	return nil
}

func (s *stubMessageRepo) ListConversationHistory(ctx context.Context, conversationID uint64, limit int, beforeSeq, afterSeq uint64) ([]model.Message, bool, error) {
	if s.listConversationHistoryFn != nil {
		return s.listConversationHistoryFn(ctx, conversationID, limit, beforeSeq, afterSeq)
	}
	return []model.Message{}, false, nil
}

func (s *stubMessageRepo) ListConversationAfterSeq(ctx context.Context, conversationID, afterSeq uint64, limit int) ([]model.Message, bool, error) {
	if s.listConversationAfterSeqFn != nil {
		return s.listConversationAfterSeqFn(ctx, conversationID, afterSeq, limit)
	}
	return []model.Message{}, false, nil
}

func (s *stubMessageRepo) ListConversationRangeAfterSeq(ctx context.Context, conversationID, afterSeq, untilSeq uint64, limit int) ([]model.Message, bool, error) {
	if s.listConversationRangeFn != nil {
		return s.listConversationRangeFn(ctx, conversationID, afterSeq, untilSeq, limit)
	}
	return []model.Message{}, false, nil
}

func (s *stubMessageRepo) ListLatestByConversationIDs(ctx context.Context, conversationIDs []uint64) (map[uint64]model.Message, error) {
	result := make(map[uint64]model.Message, len(conversationIDs))
	for _, conversationID := range conversationIDs {
		if s.getLatestByConversationFn == nil {
			continue
		}
		msg, err := s.getLatestByConversationFn(ctx, conversationID)
		if err != nil {
			continue
		}
		result[conversationID] = msg
	}
	return result, nil
}

func (s *stubMessageRepo) GetMaxSeqByConversation(ctx context.Context, conversationID uint64) (uint64, error) {
	if s.getMaxSeqByConversationFn != nil {
		return s.getMaxSeqByConversationFn(ctx, conversationID)
	}
	return 0, nil
}

func (s *stubMessageRepo) CountUnreadByConversationIDs(ctx context.Context, userID uint64, conversationIDs []uint64) (map[uint64]int64, error) {
	result := make(map[uint64]int64, len(conversationIDs))
	for _, conversationID := range conversationIDs {
		if s.countUnreadFn == nil {
			continue
		}
		count, err := s.countUnreadFn(ctx, conversationID, userID, 0)
		if err != nil {
			return nil, err
		}
		result[conversationID] = count
	}
	return result, nil
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
