package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
	"github.com/zyy125/im-system/internal/repository"
	"gorm.io/gorm"
)

const groupMemberLimit = 200

type conversationService struct {
	conversationRepo repository.ConversationRepo
	messageRepo      repository.MessageRepo
	userRepo         repository.UserRepo
	presenceRepo     repository.PresenceRepo
	txManager        repository.MessageTxManager
	seqAllocator     SeqAllocator
}

type ConversationService interface {
	// OpenConversation 按会话 ID 打开一个当前用户可访问的会话。
	OpenConversation(ctx context.Context, userID, conversationID uint64) (OpenConversationResult, error)
	// ListOfflineMessages 汇总当前用户所有会话中尚未读到的离线消息。
	ListOfflineMessages(ctx context.Context, userID uint64) ([]model.Message, error)
	// MarkRead 将当前用户在指定会话中的已读游标推进到指定 seq，并返回需要接收回执的活跃成员。
	MarkRead(ctx context.Context, userID, conversationID, readSeq uint64) ([]uint64, error)
	// ListConversations 返回当前用户可见且仍可访问的会话列表摘要。
	ListConversations(ctx context.Context, userID uint64) ([]ConversationSummary, error)
	// ListGroups 返回当前用户仍为活跃成员的全部群聊列表，不受 visible 影响。
	ListGroups(ctx context.Context, userID uint64) ([]ConversationSummary, error)
	// HideConversation 仅对当前用户隐藏指定会话，不删除会话和消息。
	HideConversation(ctx context.Context, userID, conversationID uint64) error
	// CreateGroup 创建一个新的群聊，并初始化群主和初始成员。
	CreateGroup(ctx context.Context, ownerID uint64, name string, memberIDs []uint64) (ConversationSummary, error)
	// GetGroupDetail 返回群聊基础信息和当前用户在群内的身份。
	GetGroupDetail(ctx context.Context, userID, conversationID uint64) (GroupDetail, error)
	// ListGroupMembers 返回群聊当前全部活跃成员。
	ListGroupMembers(ctx context.Context, userID, conversationID uint64) ([]GroupMember, error)
	// UpdateGroupName 修改群名称，并写入一条系统消息。
	UpdateGroupName(ctx context.Context, userID, conversationID uint64, name string) error
	// InviteGroupMembers 邀请新成员入群，已不活跃成员会按重新入群处理。
	InviteGroupMembers(ctx context.Context, userID, conversationID uint64, memberIDs []uint64) error
	// RemoveGroupMember 将某个成员移出群聊。
	RemoveGroupMember(ctx context.Context, userID, conversationID, memberID uint64) error
	// LeaveGroup 让当前用户主动退出群聊。
	LeaveGroup(ctx context.Context, userID, conversationID uint64) error
	// DismissGroup 解散群聊，并让全部成员失效。
	DismissGroup(ctx context.Context, userID, conversationID uint64) error
}

var _ ConversationService = (*conversationService)(nil)

type ConversationSummary struct {
	ID          uint64
	Type        model.ConversationType
	Name        string
	UnreadCount int64
	LastMessage *model.Message
	Peer        *ConversationPeer
}

type ConversationPeer struct {
	ID       uint64
	Avatar   string
	Username string
	Online   bool
}

type LatestReadState struct {
	LatestSentSeq uint64
	ReadByUserIDs []uint64
}

type OpenConversationResult struct {
	Conversation    ConversationSummary
	LatestReadState *LatestReadState
}

type GroupDetail struct {
	ID          uint64
	Name        string
	Avatar      string
	OwnerID     uint64
	Status      model.ConversationStatus
	MyRole      model.ConversationMemberRole
	MemberCount int64
}

type GroupMember struct {
	UserID   uint64
	Avatar   string
	Username string
	Role     model.ConversationMemberRole
	Online   bool
}

func NewConversationService(
	conversationRepo repository.ConversationRepo,
	messageRepo repository.MessageRepo,
	userRepo repository.UserRepo,
	presenceRepo repository.PresenceRepo,
	txManager repository.MessageTxManager,
) ConversationService {
	return NewConversationServiceWithRuntime(
		conversationRepo,
		messageRepo,
		userRepo,
		presenceRepo,
		txManager,
		nil,
	)
}

func NewConversationServiceWithRuntime(
	conversationRepo repository.ConversationRepo,
	messageRepo repository.MessageRepo,
	userRepo repository.UserRepo,
	presenceRepo repository.PresenceRepo,
	txManager repository.MessageTxManager,
	seqAllocator SeqAllocator,
) ConversationService {
	return &conversationService{
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		userRepo:         userRepo,
		presenceRepo:     presenceRepo,
		txManager:        txManager,
		seqAllocator:     seqAllocator,
	}
}

func (s *conversationService) OpenConversation(ctx context.Context, userID, conversationID uint64) (OpenConversationResult, error) {
	conv, member, err := s.requireActiveConversationMember(ctx, conversationID, userID)
	if err != nil {
		return OpenConversationResult{}, err
	}
	if !member.Visible {
		if err := s.conversationRepo.SetVisible(ctx, conversationID, userID, true); err != nil {
			return OpenConversationResult{}, err
		}
	}
	summary, err := s.buildConversationSummary(ctx, userID, conv)
	if err != nil {
		return OpenConversationResult{}, err
	}
	latestReadState, err := s.buildLatestReadState(ctx, member)
	if err != nil {
		return OpenConversationResult{}, err
	}
	return OpenConversationResult{
		Conversation:    summary,
		LatestReadState: latestReadState,
	}, nil
}

func (s *conversationService) CreateGroup(ctx context.Context, ownerID uint64, name string, memberIDs []uint64) (ConversationSummary, error) {
	if ownerID == 0 {
		return ConversationSummary{}, apperr.RequiredOne("owner_id")
	}
	name = normalizeConversationName(name)
	if name == "" {
		return ConversationSummary{}, apperr.ConversationInvalidName()
	}

	memberIDs = uniqueUserIDs(memberIDs)
	memberIDs = filterOutUserID(memberIDs, ownerID)
	if len(memberIDs)+1 > groupMemberLimit {
		return ConversationSummary{}, apperr.ConversationMemberLimitExceeded()
	}
	if err := s.ensureUsersExist(ctx, append([]uint64{ownerID}, memberIDs...)); err != nil {
		return ConversationSummary{}, err
	}

	var created model.Conversation
	err := s.withinConversationTx(ctx, func(messageRepo repository.MessageRepo, conversationRepo repository.ConversationRepo) error {
		created = model.Conversation{
			Type:    model.ConversationTypeGroup,
			Name:    name,
			OwnerID: ownerID,
			Status:  model.ConversationStatusActive,
		}
		if err := conversationRepo.Create(ctx, &created); err != nil {
			return err
		}
		if err := conversationRepo.UpsertMember(ctx, &model.ConversationMember{
			ConversationID: created.ID,
			UserID:         ownerID,
			Role:           model.ConversationMemberRoleOwner,
			Status:         model.ConversationMemberStatusActive,
			Visible:        true,
		}); err != nil {
			return err
		}
		for _, memberID := range memberIDs {
			if err := conversationRepo.UpsertMember(ctx, &model.ConversationMember{
				ConversationID: created.ID,
				UserID:         memberID,
				Role:           model.ConversationMemberRoleMember,
				Status:         model.ConversationMemberStatusActive,
				Visible:        true,
				InvitedBy:      ownerID,
			}); err != nil {
				return err
			}
		}
		event, content, extra, err := buildGroupCreatedMessage(name)
		if err != nil {
			return err
		}
		_, err = s.appendSystemMessage(ctx, messageRepo, conversationRepo, created.ID, ownerID, event, content, extra)
		return err
	})
	if err != nil {
		return ConversationSummary{}, err
	}
	return s.buildConversationSummary(ctx, ownerID, created)
}

func (s *conversationService) GetGroupDetail(ctx context.Context, userID, conversationID uint64) (GroupDetail, error) {
	conv, member, err := s.requireActiveGroupMember(ctx, conversationID, userID)
	if err != nil {
		return GroupDetail{}, err
	}
	count, err := s.conversationRepo.CountActiveMembers(ctx, conversationID)
	if err != nil {
		return GroupDetail{}, err
	}
	owner, err := s.userRepo.GetByID(ctx, conv.OwnerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return GroupDetail{}, apperr.UserNotFound()
		}
		return GroupDetail{}, err
	}
	return GroupDetail{
		ID:          conv.ID,
		Name:        conv.Name,
		Avatar:      conv.Avatar,
		OwnerID:     owner.ID,
		Status:      conv.Status,
		MyRole:      member.Role,
		MemberCount: count,
	}, nil
}

func (s *conversationService) ListGroupMembers(ctx context.Context, userID, conversationID uint64) ([]GroupMember, error) {
	_, _, err := s.requireActiveGroupMember(ctx, conversationID, userID)
	if err != nil {
		return nil, err
	}

	members, err := s.conversationRepo.ListActiveMembers(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	userIDs := make([]uint64, 0, len(members))
	roleByUser := make(map[uint64]model.ConversationMemberRole, len(members))
	for _, member := range members {
		userIDs = append(userIDs, member.UserID)
		roleByUser[member.UserID] = member.Role
	}
	users, err := s.userRepo.ListByIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	onlineByUserID, err := s.presenceRepo.BatchGetOnline(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	items := make([]GroupMember, 0, len(users))
	for _, user := range users {
		items = append(items, GroupMember{
			UserID:   user.ID,
			Avatar:   user.Avatar,
			Username: user.Username,
			Role:     roleByUser[user.ID],
			Online:   onlineByUserID[user.ID],
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Role == items[j].Role {
			return items[i].UserID < items[j].UserID
		}
		return items[i].Role < items[j].Role
	})
	return items, nil
}

func (s *conversationService) UpdateGroupName(ctx context.Context, userID, conversationID uint64, name string) error {
	name = normalizeConversationName(name)
	if name == "" {
		return apperr.ConversationInvalidName()
	}
	conv, member, err := s.requireActiveGroupMember(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	if member.Role != model.ConversationMemberRoleOwner {
		return apperr.ConversationNoPermission("rename")
	}
	if err := s.withinConversationTx(ctx, func(messageRepo repository.MessageRepo, conversationRepo repository.ConversationRepo) error {
		if err := conversationRepo.UpdateName(ctx, conversationID, name); err != nil {
			return err
		}
		event, content, extra, err := buildGroupRenamedMessage(name)
		if err != nil {
			return err
		}
		_, err = s.appendSystemMessage(ctx, messageRepo, conversationRepo, conv.ID, userID, event, content, extra)
		return err
	}); err != nil {
		return err
	}
	return nil
}

func (s *conversationService) InviteGroupMembers(ctx context.Context, userID, conversationID uint64, memberIDs []uint64) error {
	conv, member, err := s.requireActiveGroupMember(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	if member.Role != model.ConversationMemberRoleOwner {
		return apperr.ConversationNoPermission("invite")
	}

	memberIDs = uniqueUserIDs(memberIDs)
	memberIDs = filterOutUserID(memberIDs, userID)
	if len(memberIDs) == 0 {
		return nil
	}
	if err := s.ensureUsersExist(ctx, memberIDs); err != nil {
		return err
	}

	currentMembers, err := s.conversationRepo.ListActiveMembers(ctx, conversationID)
	if err != nil {
		return err
	}
	currentCount := len(currentMembers)
	activeSet := make(map[uint64]struct{}, currentCount)
	for _, current := range currentMembers {
		activeSet[current.UserID] = struct{}{}
	}
	toInvite := make([]uint64, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		if _, ok := activeSet[memberID]; ok {
			continue
		}
		toInvite = append(toInvite, memberID)
	}
	if currentCount+len(toInvite) > groupMemberLimit {
		return apperr.ConversationMemberLimitExceeded()
	}
	if len(toInvite) == 0 {
		return nil
	}

	joinedMsgSeq, err := s.messageRepo.GetMaxSeqByConversation(ctx, conversationID)
	if err != nil {
		return err
	}

	if err := s.withinConversationTx(ctx, func(messageRepo repository.MessageRepo, conversationRepo repository.ConversationRepo) error {
		for _, memberID := range toInvite {
			if err := conversationRepo.UpsertMember(ctx, &model.ConversationMember{
				ConversationID: conversationID,
				UserID:         memberID,
				Role:           model.ConversationMemberRoleMember,
				Status:         model.ConversationMemberStatusActive,
				Visible:        true,
				InvitedBy:      userID,
				JoinedMsgSeq:   joinedMsgSeq,
			}); err != nil {
				return err
			}
		}
		userIDs, err := s.userIDsForUsers(ctx, toInvite)
		if err != nil {
			return err
		}
		event, content, extra, err := buildGroupMembersJoinedMessage(userIDs)
		if err != nil {
			return err
		}
		_, err = s.appendSystemMessage(ctx, messageRepo, conversationRepo, conv.ID, userID, event, content, extra)
		return err
	}); err != nil {
		return err
	}
	return nil
}

func (s *conversationService) RemoveGroupMember(ctx context.Context, userID, conversationID, memberID uint64) error {
	conv, member, err := s.requireActiveGroupMember(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	if member.Role != model.ConversationMemberRoleOwner {
		return apperr.ConversationNoPermission("remove member")
	}
	if memberID == 0 || memberID == userID {
		return apperr.InvalidArgument("invalid member id")
	}

	target, err := s.conversationRepo.GetMember(ctx, conversationID, memberID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.ConversationMemberNotFound()
		}
		return err
	}
	if !target.IsActive() {
		return apperr.ConversationMemberNotFound()
	}

	if err := s.withinConversationTx(ctx, func(messageRepo repository.MessageRepo, conversationRepo repository.ConversationRepo) error {
		if err := conversationRepo.UpdateMemberStatus(ctx, conversationID, memberID, model.ConversationMemberStatusRemoved, false); err != nil {
			return err
		}
		userIDs, err := s.userIDsForUsers(ctx, []uint64{memberID})
		if err != nil {
			return err
		}
		event, content, extra, err := buildGroupMemberRemovedMessage(userIDs[0])
		if err != nil {
			return err
		}
		_, err = s.appendSystemMessage(ctx, messageRepo, conversationRepo, conv.ID, userID, event, content, extra)
		return err
	}); err != nil {
		return err
	}
	return nil
}

func (s *conversationService) LeaveGroup(ctx context.Context, userID, conversationID uint64) error {
	conv, member, err := s.requireActiveGroupMember(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	if member.Role == model.ConversationMemberRoleOwner {
		return apperr.ConversationOwnerCannotLeave()
	}
	if err := s.withinConversationTx(ctx, func(messageRepo repository.MessageRepo, conversationRepo repository.ConversationRepo) error {
		if err := conversationRepo.UpdateMemberStatus(ctx, conversationID, userID, model.ConversationMemberStatusLeft, false); err != nil {
			return err
		}
		userIDs, err := s.userIDsForUsers(ctx, []uint64{userID})
		if err != nil {
			return err
		}
		event, content, extra, err := buildGroupMemberLeftMessage(userIDs[0])
		if err != nil {
			return err
		}
		_, err = s.appendSystemMessage(ctx, messageRepo, conversationRepo, conv.ID, userID, event, content, extra)
		return err
	}); err != nil {
		return err
	}
	return nil
}

func (s *conversationService) DismissGroup(ctx context.Context, userID, conversationID uint64) error {
	conv, member, err := s.requireActiveGroupMember(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	if member.Role != model.ConversationMemberRoleOwner {
		return apperr.ConversationNoPermission("dismiss")
	}
	if err := s.withinConversationTx(ctx, func(messageRepo repository.MessageRepo, conversationRepo repository.ConversationRepo) error {
		if err := conversationRepo.UpdateStatus(ctx, conversationID, model.ConversationStatusDismissed); err != nil {
			return err
		}
		if err := conversationRepo.UpdateAllMemberStatus(ctx, conversationID, model.ConversationMemberStatusLeft, false); err != nil {
			return err
		}
		event, content, extra, err := buildGroupDismissedMessage()
		if err != nil {
			return err
		}
		_, err = s.appendSystemMessage(ctx, messageRepo, conversationRepo, conv.ID, userID, event, content, extra)
		return err
	}); err != nil {
		return err
	}
	return nil
}

func (s *conversationService) ListOfflineMessages(ctx context.Context, userID uint64) ([]model.Message, error) {
	if userID == 0 {
		return nil, apperr.RequiredOne("user_id")
	}

	members, err := s.conversationRepo.ListMembersByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	msgs := make([]model.Message, 0)
	for _, member := range members {
		if !member.IsActive() {
			continue
		}

		// 离线补推以“最后已确认收到的最大 seq”为起点，而不是仅看已读位置。
		afterSeq := member.LastAckedMsgSeq
		if member.JoinedMsgSeq > afterSeq {
			afterSeq = member.JoinedMsgSeq
		}
		maxSeq, err := s.messageRepo.GetMaxSeqByConversation(ctx, member.ConversationID)
		if err != nil {
			return nil, err
		}
		pending, err := s.listConversationRange(ctx, member.ConversationID, afterSeq, maxSeq)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, pending...)
	}

	sort.Slice(msgs, func(i, j int) bool {
		if msgs[i].SendTime == msgs[j].SendTime {
			if msgs[i].ConversationID == msgs[j].ConversationID {
				return msgs[i].Seq < msgs[j].Seq
			}
			return msgs[i].ConversationID < msgs[j].ConversationID
		}
		return msgs[i].SendTime < msgs[j].SendTime
	})

	return msgs, nil
}

func (s *conversationService) MarkRead(ctx context.Context, userID, conversationID, readSeq uint64) ([]uint64, error) {
	if userID == 0 || conversationID == 0 || readSeq == 0 {
		return nil, apperr.Required("user_id", "conversation_id", "read_seq")
	}

	conv, member, err := s.requireActiveConversationMember(ctx, conversationID, userID)
	if err != nil {
		return nil, err
	}

	if readSeq <= member.JoinedMsgSeq {
		return nil, apperr.MessageNotReadable()
	}
	if readSeq > member.LastAckedMsgSeq {
		return nil, apperr.MessageNotDelivered()
	}

	oldReadSeq := member.LastReadMsgSeq
	if err := s.conversationRepo.UpdateLastReadMsgSeq(ctx, conversationID, userID, readSeq); err != nil {
		return nil, err
	}

	if conv.IsGroup() {
		return s.conversationRepo.ListGroupReadReceiptTargets(ctx, conversationID, userID, oldReadSeq, readSeq)
	}

	members, err := s.conversationRepo.ListActiveMembers(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	return memberUserIDs(members), nil
}

func (s *conversationService) ListConversations(ctx context.Context, userID uint64) ([]ConversationSummary, error) {
	if userID == 0 {
		return nil, apperr.RequiredOne("user_id")
	}

	members, err := s.conversationRepo.ListMembersByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return []ConversationSummary{}, nil
	}

	memberByConversation := make(map[uint64]model.ConversationMember, len(members))
	for _, member := range members {
		memberByConversation[member.ConversationID] = member
	}

	conversations, err := s.conversationRepo.ListConversationsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	items, err := s.buildConversationSummaries(ctx, userID, conversations, memberByConversation)
	if err != nil {
		return nil, err
	}

	sort.Slice(items, func(i, j int) bool {
		leftTime := int64(0)
		if items[i].LastMessage != nil {
			leftTime = items[i].LastMessage.SendTime
		}
		rightTime := int64(0)
		if items[j].LastMessage != nil {
			rightTime = items[j].LastMessage.SendTime
		}
		if leftTime == rightTime {
			return items[i].ID > items[j].ID
		}
		return leftTime > rightTime
	})

	return items, nil
}

func (s *conversationService) ListGroups(ctx context.Context, userID uint64) ([]ConversationSummary, error) {
	if userID == 0 {
		return nil, apperr.RequiredOne("user_id")
	}

	conversations, err := s.conversationRepo.ListActiveGroupsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(conversations) == 0 {
		return []ConversationSummary{}, nil
	}

	members, err := s.conversationRepo.ListMembersByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	memberByConversation := make(map[uint64]model.ConversationMember, len(members))
	for _, member := range members {
		memberByConversation[member.ConversationID] = member
	}
	if len(memberByConversation) == 0 {
		for _, conversation := range conversations {
			member, err := s.conversationRepo.GetMember(ctx, conversation.ID, userID)
			if err != nil {
				return nil, err
			}
			memberByConversation[conversation.ID] = member
		}
	}

	return s.buildConversationSummaries(ctx, userID, conversations, memberByConversation)
}

func (s *conversationService) HideConversation(ctx context.Context, userID, conversationID uint64) error {
	if userID == 0 || conversationID == 0 {
		return apperr.Required("user_id", "conversation_id")
	}
	return s.conversationRepo.SetVisible(ctx, conversationID, userID, false)
}

func (s *conversationService) buildConversationSummary(ctx context.Context, userID uint64, conversation model.Conversation) (ConversationSummary, error) {
	member, err := s.conversationRepo.GetMember(ctx, conversation.ID, userID)
	if err != nil {
		return ConversationSummary{}, err
	}
	items, err := s.buildConversationSummaries(ctx, userID, []model.Conversation{conversation}, map[uint64]model.ConversationMember{
		conversation.ID: member,
	})
	if err != nil {
		return ConversationSummary{}, err
	}
	if len(items) == 0 {
		return ConversationSummary{}, apperr.ConversationNotAccessible()
	}
	return items[0], nil
}

func (s *conversationService) buildLatestReadState(ctx context.Context, member model.ConversationMember) (*LatestReadState, error) {
	if member.ConversationID == 0 || member.UserID == 0 || member.LastSentMsgSeq == 0 {
		return nil, nil
	}

	readByUserIDs, err := s.conversationRepo.ListReadReceiptUsersBySentSeq(ctx, member.ConversationID, member.UserID, member.LastSentMsgSeq)
	if err != nil {
		return nil, err
	}
	users, err := s.userRepo.ListByIDs(ctx, uniqueUserIDs(readByUserIDs))
	if err != nil {
		return nil, err
	}
	return &LatestReadState{
		LatestSentSeq: member.LastSentMsgSeq,
		ReadByUserIDs: userIDsForUsers(usersByID(users), readByUserIDs),
	}, nil
}

func (s *conversationService) buildConversationSummaries(
	ctx context.Context,
	userID uint64,
	conversations []model.Conversation,
	memberByConversation map[uint64]model.ConversationMember,
) ([]ConversationSummary, error) {
	if len(conversations) == 0 {
		return []ConversationSummary{}, nil
	}

	conversationIDs := make([]uint64, 0, len(conversations))
	singlePeerIDs := make([]uint64, 0)
	peerByConversation := make(map[uint64]uint64)
	filtered := make([]model.Conversation, 0, len(conversations))

	for _, conversation := range conversations {
		member, ok := memberByConversation[conversation.ID]
		if !ok || !member.IsActive() {
			continue
		}
		conversationIDs = append(conversationIDs, conversation.ID)
		filtered = append(filtered, conversation)

		if !conversation.IsSingle() {
			continue
		}
		peerID, err := extractPeerID(conversation, userID)
		if err != nil {
			return nil, err
		}
		if peerID == 0 {
			continue
		}
		peerByConversation[conversation.ID] = peerID
		singlePeerIDs = append(singlePeerIDs, peerID)
	}

	latestByConversation, err := s.messageRepo.ListLatestByConversationIDs(ctx, conversationIDs)
	if err != nil {
		return nil, err
	}
	unreadByConversation, err := s.messageRepo.CountUnreadByConversationIDs(ctx, userID, conversationIDs)
	if err != nil {
		return nil, err
	}

	peerUsers := make(map[uint64]model.User)
	if len(singlePeerIDs) > 0 {
		users, err := s.userRepo.ListByIDs(ctx, uniqueUserIDs(singlePeerIDs))
		if err != nil {
			return nil, err
		}
		for _, user := range users {
			peerUsers[user.ID] = user
		}
	}

	onlineByUserID, err := s.presenceRepo.BatchGetOnline(ctx, uniqueUserIDs(singlePeerIDs))
	if err != nil {
		return nil, err
	}

	items := make([]ConversationSummary, 0, len(filtered))
	for _, conversation := range filtered {
		item := ConversationSummary{
			ID:          conversation.ID,
			Type:        conversation.Type,
			Name:        conversation.Name,
			UnreadCount: unreadByConversation[conversation.ID],
		}
		if latestMessage, ok := latestByConversation[conversation.ID]; ok {
			item.LastMessage = &latestMessage
		}
		if peerID, ok := peerByConversation[conversation.ID]; ok {
			user, ok := peerUsers[peerID]
			if !ok {
				return nil, apperr.UserNotFound()
			}
			item.Peer = &ConversationPeer{
				ID:       user.ID,
				Avatar:   user.Avatar,
				Username: user.Username,
				Online:   onlineByUserID[user.ID],
			}
			if item.Name == "" {
				item.Name = user.Username
			}
		}
		items = append(items, item)
	}
	return items, nil
}

func extractPeerID(conversation model.Conversation, userID uint64) (uint64, error) {
	singleKey := conversation.SingleKeyValue()
	if singleKey == "" {
		return 0, nil
	}

	parts := strings.Split(singleKey, ":")
	if len(parts) != 2 {
		return 0, apperr.ConversationInvalidSingleKey()
	}

	leftID, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	rightID, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return 0, err
	}

	switch {
	case leftID == userID:
		return rightID, nil
	case rightID == userID:
		return leftID, nil
	default:
		return 0, apperr.ConversationNotAccessible()
	}
}

func (s *conversationService) requireActiveConversationMember(ctx context.Context, conversationID, userID uint64) (model.Conversation, model.ConversationMember, error) {
	conv, err := s.conversationRepo.GetByID(ctx, conversationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Conversation{}, model.ConversationMember{}, apperr.NotFound(apperr.CodeConversationNotFound, "conversation not found")
		}
		return model.Conversation{}, model.ConversationMember{}, err
	}
	if !conv.IsActive() {
		return model.Conversation{}, model.ConversationMember{}, apperr.ConversationDismissed()
	}
	member, err := s.conversationRepo.GetMember(ctx, conversationID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Conversation{}, model.ConversationMember{}, apperr.ConversationMemberNotFound()
		}
		return model.Conversation{}, model.ConversationMember{}, err
	}
	if !member.IsActive() {
		return model.Conversation{}, model.ConversationMember{}, apperr.ConversationNotAccessible()
	}
	return conv, member, nil
}

func (s *conversationService) requireActiveGroupMember(ctx context.Context, conversationID, userID uint64) (model.Conversation, model.ConversationMember, error) {
	conv, member, err := s.requireActiveConversationMember(ctx, conversationID, userID)
	if err != nil {
		return model.Conversation{}, model.ConversationMember{}, err
	}
	if !conv.IsGroup() {
		return model.Conversation{}, model.ConversationMember{}, apperr.ConversationInvalidType()
	}
	return conv, member, nil
}

func (s *conversationService) allocateMessageSeq(ctx context.Context, conversationID uint64) (uint64, error) {
	if s.seqAllocator != nil {
		return s.seqAllocator.Allocate(ctx, conversationID)
	}
	// 旧路径或测试桩未注入 SeqAllocator 时，退回到基于 DB 最大 seq 的简单分配方式。
	maxSeq, err := s.messageRepo.GetMaxSeqByConversation(ctx, conversationID)
	if err != nil {
		return 0, err
	}
	return maxSeq + 1, nil
}

func (s *conversationService) listConversationRange(ctx context.Context, conversationID, afterSeq, untilSeq uint64) ([]model.Message, error) {
	const batchSize = 200
	msgs := make([]model.Message, 0)
	for {
		batch, hasMore, err := s.messageRepo.ListConversationRangeAfterSeq(ctx, conversationID, afterSeq, untilSeq, batchSize)
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			return msgs, nil
		}
		msgs = append(msgs, batch...)
		afterSeq = batch[len(batch)-1].Seq
		if !hasMore {
			return msgs, nil
		}
	}
}

func (s *conversationService) appendSystemMessage(
	ctx context.Context,
	messageRepo repository.MessageRepo,
	conversationRepo repository.ConversationRepo,
	conversationID, actorID uint64,
	event model.MessageEvent,
	content string,
	extra json.RawMessage,
) (model.Message, error) {
	seq, err := s.allocateMessageSeq(ctx, conversationID)
	if err != nil {
		return model.Message{}, err
	}

	msg := model.Message{
		MsgID:          buildSystemMsgID(conversationID, actorID),
		ConversationID: conversationID,
		Seq:            seq,
		Type:           model.MessageTypeSystem,
		Event:          event,
		From:           actorID,
		SendTime:       time.Now().UnixMilli(),
		Content:        content,
		Extra:          extra,
	}
	if err := messageRepo.Create(ctx, &msg); err != nil {
		return model.Message{}, err
	}

	members, err := conversationRepo.ListActiveMembers(ctx, conversationID)
	if err != nil {
		return model.Message{}, err
	}
	memberIDs := memberUserIDs(members)
	if err := conversationRepo.SetVisibleForUsers(ctx, conversationID, memberIDs, true); err != nil {
		return model.Message{}, err
	}
	if err := conversationRepo.UpdateLastAckedMsgSeq(ctx, conversationID, actorID, msg.Seq); err != nil && apperr.CodeOf(err) != apperr.CodeConversationMemberNotFound {
		return model.Message{}, err
	}
	if err := conversationRepo.UpdateLastReadMsgSeq(ctx, conversationID, actorID, msg.Seq); err != nil && apperr.CodeOf(err) != apperr.CodeConversationMemberNotFound {
		return model.Message{}, err
	}
	return msg, nil
}

func (s *conversationService) withinConversationTx(ctx context.Context, fn func(messageRepo repository.MessageRepo, conversationRepo repository.ConversationRepo) error) error {
	if s.txManager != nil {
		return s.txManager.WithinMessageTx(ctx, fn)
	}
	// 无事务管理器时退回到直连仓储，主要用于轻量测试桩。
	return fn(s.messageRepo, s.conversationRepo)
}

func (s *conversationService) ensureUsersExist(ctx context.Context, userIDs []uint64) error {
	if len(userIDs) == 0 {
		return nil
	}
	users, err := s.userRepo.ListByIDs(ctx, userIDs)
	if err != nil {
		return err
	}
	if len(users) != len(userIDs) {
		return apperr.UserNotFound()
	}
	return nil
}

func uniqueUserIDs(ids []uint64) []uint64 {
	if len(ids) == 0 {
		return ids
	}
	seen := make(map[uint64]struct{}, len(ids))
	result := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func memberUserIDs(members []model.ConversationMember) []uint64 {
	ids := make([]uint64, 0, len(members))
	for _, member := range members {
		ids = append(ids, member.UserID)
	}
	return ids
}

func filterOutUserID(ids []uint64, userID uint64) []uint64 {
	if len(ids) == 0 {
		return ids
	}
	result := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if id == userID {
			continue
		}
		result = append(result, id)
	}
	return result
}

func normalizeConversationName(name string) string {
	return strings.TrimSpace(name)
}

func buildSystemMsgID(conversationID, actorID uint64) string {
	return fmt.Sprintf("sys:%d:%d:%d", conversationID, actorID, time.Now().UnixNano())
}

func (s *conversationService) userIDsForUsers(ctx context.Context, userIDs []uint64) ([]uint64, error) {
	users, err := s.userRepo.ListByIDs(ctx, uniqueUserIDs(userIDs))
	if err != nil {
		return nil, err
	}
	byID := usersByID(users)
	result := make([]uint64, 0, len(userIDs))
	for _, userID := range userIDs {
		if _, ok := byID[userID]; !ok {
			return nil, apperr.UserNotFound()
		}
		result = append(result, userID)
	}
	return result, nil
}

func usersByID(users []model.User) map[uint64]model.User {
	result := make(map[uint64]model.User, len(users))
	for _, user := range users {
		result[user.ID] = user
	}
	return result
}

func userIDsForUsers(users map[uint64]model.User, ids []uint64) []uint64 {
	result := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if _, ok := users[id]; ok {
			result = append(result, id)
			continue
		}
		result = append(result, id)
	}
	return result
}

func marshalMessageExtra(v any) (json.RawMessage, error) {
	if v == nil {
		return nil, nil
	}
	payload, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(payload), nil
}

func buildGroupCreatedMessage(name string) (model.MessageEvent, string, json.RawMessage, error) {
	extra, err := marshalMessageExtra(struct {
		GroupName string `json:"group_name"`
	}{
		GroupName: name,
	})
	if err != nil {
		return model.MessageEventNone, "", nil, err
	}
	return model.MessageEventGroupCreated, "创建了群聊 " + name, extra, nil
}

func buildGroupRenamedMessage(name string) (model.MessageEvent, string, json.RawMessage, error) {
	extra, err := marshalMessageExtra(struct {
		NewName string `json:"new_name"`
	}{
		NewName: name,
	})
	if err != nil {
		return model.MessageEventNone, "", nil, err
	}
	return model.MessageEventGroupRenamed, "修改群名称为 " + name, extra, nil
}

func buildGroupMembersJoinedMessage(memberIDs []uint64) (model.MessageEvent, string, json.RawMessage, error) {
	extra, err := marshalMessageExtra(struct {
		MemberIDs []uint64 `json:"member_ids"`
		Count     int      `json:"count"`
	}{
		MemberIDs: memberIDs,
		Count:     len(memberIDs),
	})
	if err != nil {
		return model.MessageEventNone, "", nil, err
	}
	return model.MessageEventGroupMembersJoined, fmt.Sprintf("%d 位成员加入了群聊", len(memberIDs)), extra, nil
}

func buildGroupMemberRemovedMessage(memberID uint64) (model.MessageEvent, string, json.RawMessage, error) {
	extra, err := marshalMessageExtra(struct {
		TargetUserID uint64 `json:"target_user_id"`
	}{
		TargetUserID: memberID,
	})
	if err != nil {
		return model.MessageEventNone, "", nil, err
	}
	return model.MessageEventGroupMemberRemoved, fmt.Sprintf("成员 %d 被移出群聊", memberID), extra, nil
}

func buildGroupMemberLeftMessage(userID uint64) (model.MessageEvent, string, json.RawMessage, error) {
	extra, err := marshalMessageExtra(struct {
		UserID uint64 `json:"user_id"`
	}{
		UserID: userID,
	})
	if err != nil {
		return model.MessageEventNone, "", nil, err
	}
	return model.MessageEventGroupMemberLeft, "退出了群聊", extra, nil
}

func buildGroupDismissedMessage() (model.MessageEvent, string, json.RawMessage, error) {
	return model.MessageEventGroupDismissed, "解散了群聊", nil, nil
}
