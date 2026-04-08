package service

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
	"github.com/zyy125/im-system/internal/repository"
	"gorm.io/gorm"
)

type conversationService struct {
	conversationRepo repository.ConversationRepo
	messageRepo      repository.MessageRepo
	userRepo         repository.UserRepo
	presenceRepo     repository.PresenceRepo
	friendRepo       repository.FriendRepo
}

type ConversationService interface {
	EnsureDirectConversationID(ctx context.Context, userA, userB uint64) (string, error)
	OpenDirectConversation(ctx context.Context, userID, peerID uint64) (ConversationSummary, error)
	ListOfflineMessages(ctx context.Context, userID uint64) ([]model.ChatMessage, error)
	MarkRead(ctx context.Context, userID uint64, conversationID, msgID string) error
	ListConversations(ctx context.Context, userID uint64) ([]ConversationSummary, error)
	HideConversation(ctx context.Context, userID, conversationID uint64) error
}

var _ ConversationService = (*conversationService)(nil)

type ConversationSummary struct {
	ID          uint64
	Type        model.ConversationType
	Name        string
	UnreadCount int64
	LastMessage *model.ChatMessage
	Peer        *ConversationPeer
}

type ConversationPeer struct {
	ID       uint64
	Username string
	Online   bool
}

func NewConversationService(
	conversationRepo repository.ConversationRepo,
	messageRepo repository.MessageRepo,
	userRepo repository.UserRepo,
	presenceRepo repository.PresenceRepo,
	friendRepo repository.FriendRepo,
) ConversationService {
	return &conversationService{
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		userRepo:         userRepo,
		presenceRepo:     presenceRepo,
		friendRepo:       friendRepo,
	}
}

func (s *conversationService) EnsureDirectConversationID(ctx context.Context, userA, userB uint64) (string, error) {
	if userA == 0 || userB == 0 {
		return "", apperr.Required("user_id", "peer_id")
	}
	conv, err := s.conversationRepo.GetOrCreateSingle(ctx, userA, userB)
	if err != nil {
		return "", err
	}
	return strconv.FormatUint(conv.ID, 10), nil
}

func (s *conversationService) OpenDirectConversation(ctx context.Context, userID, peerID uint64) (ConversationSummary, error) {
	if userID == 0 || peerID == 0 {
		return ConversationSummary{}, apperr.Required("user_id", "peer_id")
	}

	ok, err := s.friendRepo.AreFriends(ctx, userID, peerID)
	if err != nil {
		return ConversationSummary{}, err
	}
	if !ok {
		return ConversationSummary{}, apperr.FriendNotFriends()
	}

	conv, err := s.conversationRepo.GetOrCreateSingle(ctx, userID, peerID)
	if err != nil {
		return ConversationSummary{}, err
	}
	if err := s.conversationRepo.SetVisible(ctx, conv.ID, userID, true); err != nil {
		return ConversationSummary{}, err
	}

	return s.buildConversationSummary(ctx, userID, conv)
}

func (s *conversationService) ListOfflineMessages(ctx context.Context, userID uint64) ([]model.ChatMessage, error) {
	if userID == 0 {
		return nil, apperr.RequiredOne("user_id")
	}

	members, err := s.conversationRepo.ListMembersByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	msgs := make([]model.ChatMessage, 0)
	for _, member := range members {
		if member.LastDeliveredMsgSeq == 0 || member.LastDeliveredMsgSeq == member.LastReadMsgSeq {
			continue
		}

		conversationID := strconv.FormatUint(member.ConversationID, 10)
		pending, err := s.messageRepo.ListConversationPendingForUser(
			ctx,
			conversationID,
			userID,
			member.LastReadMsgSeq,
			member.LastDeliveredMsgSeq,
		)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, pending...)
	}

	sort.Slice(msgs, func(i, j int) bool {
		if msgs[i].SendTime == msgs[j].SendTime {
			return msgs[i].ID < msgs[j].ID
		}
		return msgs[i].SendTime < msgs[j].SendTime
	})

	return msgs, nil
}

func (s *conversationService) MarkRead(ctx context.Context, userID uint64, conversationID, msgID string) error {
	if conversationID == "" || msgID == "" {
		return apperr.Required("conversation_id", "msg_id")
	}

	convID, err := strconv.ParseUint(conversationID, 10, 64)
	if err != nil {
		return apperr.InvalidID("conversation_id")
	}

	member, err := s.conversationRepo.GetMember(ctx, convID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.ConversationMemberNotFound()
		}
		return err
	}

	msg, err := s.messageRepo.GetByConversationAndMsgID(ctx, conversationID, msgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.MessageNotFound()
		}
		return err
	}
	if msg.To != userID {
		return apperr.MessageNotReadable()
	}
	if msg.ID > member.LastDeliveredMsgSeq {
		return apperr.MessageNotDelivered()
	}

	return s.conversationRepo.UpdateLastReadMsgSeq(ctx, convID, userID, msg.ID)
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

	items := make([]ConversationSummary, 0, len(conversations))
	for _, conversation := range conversations {
		if _, ok := memberByConversation[conversation.ID]; !ok {
			continue
		}
		item, err := s.buildConversationSummary(ctx, userID, conversation)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
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

	item := ConversationSummary{
		ID:   conversation.ID,
		Type: conversation.Type,
		Name: conversation.Name,
	}

	conversationID := strconv.FormatUint(conversation.ID, 10)

	lastMsg, err := s.messageRepo.GetLatestByConversation(ctx, conversationID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return ConversationSummary{}, err
		}
	} else {
		item.LastMessage = &lastMsg
	}

	unreadCount, err := s.messageRepo.CountUnreadByConversation(ctx, conversationID, userID, member.LastReadMsgSeq)
	if err != nil {
		return ConversationSummary{}, err
	}
	item.UnreadCount = unreadCount

	if conversation.IsSingle() {
		peer, err := s.buildSingleConversationPeer(ctx, userID, conversation)
		if err != nil {
			return ConversationSummary{}, err
		}
		item.Peer = peer
		if item.Name == "" && peer != nil {
			item.Name = peer.Username
		}
	}

	return item, nil
}

func (s *conversationService) buildSingleConversationPeer(ctx context.Context, userID uint64, conversation model.Conversation) (*ConversationPeer, error) {
	singleKey := conversation.SingleKeyValue()
	if singleKey == "" {
		return nil, nil
	}

	parts := strings.Split(singleKey, ":")
	if len(parts) != 2 {
		return nil, apperr.ConversationInvalidSingleKey()
	}

	leftID, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return nil, err
	}
	rightID, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return nil, err
	}

	peerID := leftID
	if leftID == userID {
		peerID = rightID
	}
	if rightID == userID {
		peerID = leftID
	}
	if leftID != userID && rightID != userID {
		return nil, apperr.ConversationNotAccessible()
	}

	user, err := s.userRepo.GetByID(ctx, peerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.UserNotFound()
		}
		return nil, err
	}
	online, err := s.presenceRepo.IsOnline(ctx, peerID)
	if err != nil {
		return nil, err
	}

	return &ConversationPeer{
		ID:       peerID,
		Username: user.Username,
		Online:   online,
	}, nil
}
