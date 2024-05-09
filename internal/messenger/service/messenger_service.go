package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Point-AI/backend/config"
	"github.com/Point-AI/backend/internal/messenger/delivery/model"
	"github.com/Point-AI/backend/internal/messenger/domain/entity"
	_interface "github.com/Point-AI/backend/internal/messenger/domain/interface"
	infrastructureInterface "github.com/Point-AI/backend/internal/messenger/service/interface"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

type MessengerServiceImpl struct {
	messengerRepo    infrastructureInterface.MessengerRepository
	websocketService _interface.WebsocketService
	config           *config.Config
}

func NewMessengerServiceImpl(cfg *config.Config, messengerRepo infrastructureInterface.MessengerRepository, websocketService _interface.WebsocketService) _interface.MessengerService {
	return &MessengerServiceImpl{
		messengerRepo:    messengerRepo,
		websocketService: websocketService,
		config:           cfg,
	}
}

func (ms *MessengerServiceImpl) ReassignTicketToTeam(userId primitive.ObjectID, chatId string, ticketId, workspaceId, teamName string) error {
	session, err := ms.messengerRepo.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(context.Background())

	err = session.StartTransaction()
	if err != nil {
		return err
	}

	err = mongo.WithSession(context.Background(), session, func(sc mongo.SessionContext) error {
		originalChat, err := ms.messengerRepo.FindChatByTicketId(sc, ticketId)
		if err != nil {
			return err
		}

		ticketToMove, err := ms.findTicketInChat(originalChat, ticketId)
		if err != nil {
			return err
		}

		if len(originalChat.Tickets) == 0 {
			if err := ms.messengerRepo.DeleteChat(sc, originalChat.Id); err != nil {
				return err
			}
		} else {
			if err := ms.messengerRepo.UpdateChat(sc, originalChat); err != nil {
				return err
			}
		}

		workspace, err := ms.messengerRepo.FindWorkspaceByWorkspaceId(sc, workspaceId)
		if err != nil {
			return err
		}

		if err = ms.ValidateUserInWorkspace(userId, workspace); err != nil {
			return err
		}

		assigneeId, err := ms.getAssigneeIdByTeam(workspace, teamName)
		if err != nil {
			return err
		}

		chat, err := ms.messengerRepo.FindChatByUserId(sc, originalChat.TgClientId, workspace.Id, assigneeId)
		if err != nil {
			return err
		} else if chat == nil {
			newChat := ms.createChat(originalChat, *ticketToMove, workspace.Id, assigneeId)
			return ms.messengerRepo.InsertNewChat(sc, newChat)
		} else if chat != nil {
			chat.Tickets = append(chat.Tickets, *ticketToMove)
			return ms.messengerRepo.UpdateChat(sc, chat)
		}

		return nil
	})

	if err != nil {
		_ = session.AbortTransaction(context.Background())
		return err
	}
	return session.CommitTransaction(context.Background())
}

func (ms *MessengerServiceImpl) ReassignTicketToUser(userId primitive.ObjectID, chatId string, ticketId, workspaceId, email string) error {
	session, err := ms.messengerRepo.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(context.Background())

	err = session.StartTransaction()
	if err != nil {
		return err
	}

	err = mongo.WithSession(context.Background(), session, func(sc mongo.SessionContext) error {
		originalChat, err := ms.messengerRepo.FindChatByTicketId(sc, ticketId)
		if err != nil {
			return err
		}

		ticketToMove, err := ms.findTicketInChat(originalChat, ticketId)
		if err != nil {
			return err
		}

		if len(originalChat.Tickets) == 0 {
			if err := ms.messengerRepo.DeleteChat(sc, originalChat.Id); err != nil {
				return err
			}
		} else {
			if err := ms.messengerRepo.UpdateChat(sc, originalChat); err != nil {
				return err
			}
		}

		workspace, err := ms.messengerRepo.FindWorkspaceByWorkspaceId(sc, workspaceId)
		if err != nil {
			return err
		}

		if err = ms.ValidateUserInWorkspace(userId, workspace); err != nil {
			return err
		}

		reassignUserId, err := ms.messengerRepo.FindUserByEmail(sc, email)
		if err != nil {
			return err
		}

		chat, err := ms.messengerRepo.FindChatByUserId(sc, originalChat.TgClientId, workspace.Id, reassignUserId)
		if err != nil {
			return err
		} else if chat == nil && err == nil {
			newChat := ms.createChat(originalChat, *ticketToMove, workspace.Id, reassignUserId)
			return ms.messengerRepo.InsertNewChat(sc, newChat)
		} else if chat != nil {
			chat.Tickets = append(chat.Tickets, *ticketToMove)
			return ms.messengerRepo.UpdateChat(sc, chat)
		}

		return nil
	})

	if err != nil {
		_ = session.AbortTransaction(context.Background())
		return err
	}
	err = session.CommitTransaction(context.Background())
	return err
}

func (ms *MessengerServiceImpl) UpdateTicketStatus(userId primitive.ObjectID, ticketId, workspaceId, status string) error {
	workspace, err := ms.messengerRepo.FindWorkspaceByWorkspaceId(nil, workspaceId)
	if err != nil {
		return err
	}

	if err = ms.ValidateUserInWorkspace(userId, workspace); err != nil {
		return err
	}

	chat, err := ms.messengerRepo.FindChatByTicketId(nil, ticketId)
	if err != nil {
		return err
	}

	fmtdStatus, err := ms.validateTicketStatus(status)
	if err != nil {
		return err
	}

	found := false
	for i, ticket := range chat.Tickets {
		if ticket.TicketId == ticketId {
			chat.Tickets[i].Status = fmtdStatus
			found = true
			break
		}
	}
	if !found {
		return errors.New("ticket not found")
	}

	return ms.messengerRepo.UpdateChat(nil, chat)
}

func (ms *MessengerServiceImpl) ValidateUserInWorkspace(userId primitive.ObjectID, workspace *entity.Workspace) error {
	if _, exists := workspace.Team[userId]; exists {
		return nil
	}

	return errors.New("user does not have the permissions")
}

func (ms *MessengerServiceImpl) ValidateUserInWorkspaceById(userId primitive.ObjectID, workspaceId string) error {
	workspace, err := ms.messengerRepo.FindWorkspaceByWorkspaceId(nil, workspaceId)
	if err != nil {
		return err
	}
	if _, exists := workspace.Team[userId]; exists {
		return nil
	}

	return errors.New("user does not have the permissions")
}

func (ms *MessengerServiceImpl) HandleMessage(userId primitive.ObjectID, workspaceId, ticketId, chatId, messageType, message string) error {
	if messageType == "chat_note" {
		chat, err := ms.messengerRepo.FindChatByChatId(chatId)
		if err != nil {
			return err
		}

		note := ms.createNote(userId, message)
		chat.Notes = append(chat.Notes, *note)

		if err = ms.messengerRepo.UpdateChat(nil, chat); err != nil {
			return err
		}

		res, err := json.Marshal(ms.createMessageResponse(nil, note.CreatedAt, ticketId, chatId, message, messageType))
		if err != nil {
			return err
		}

		ms.websocketService.SendToAll(workspaceId, res)
		return nil
	} else if messageType == "ticket_note" {
		chat, err := ms.messengerRepo.FindChatByChatId(chatId)
		if err != nil {
			return err
		}
		ticket, err := ms.findTicketInChat(chat, ticketId)
		if err != nil {
			return err
		}

		note := ms.createNote(userId, message)
		ticket.Notes = append(ticket.Notes, *note)
		if err = ms.messengerRepo.UpdateChat(nil, chat); err != nil {
			return err
		}

		res, err := json.Marshal(ms.createMessageResponse(nil, note.CreatedAt, ticketId, chatId, message, messageType))
		if err != nil {
			return err
		}

		ms.websocketService.SendToAll(workspaceId, res)
		return nil
	}

	return errors.New("unknown message type")
}

func (ms *MessengerServiceImpl) UpdateChatInfo(userId primitive.ObjectID, chatId string, tags []string, workspaceId string) error {
	workspace, err := ms.messengerRepo.FindWorkspaceByWorkspaceId(nil, workspaceId)
	if err != nil {
		return err
	}

	if err = ms.ValidateUserInWorkspace(userId, workspace); err != nil {
		return err
	}

	chat, err := ms.messengerRepo.FindChatByWorkspaceIdAndChatId(workspace.Id, chatId)
	if err != nil {
		return err
	}
	chat.Tags = tags

	return ms.messengerRepo.UpdateChat(nil, chat)
}

func (ms *MessengerServiceImpl) getAssigneeIdByTeam(workspace *entity.Workspace, teamName string) (primitive.ObjectID, error) {
	if team, exists := workspace.InternalTeams[teamName]; exists {
		return ms.findLeastBusyMember(team)
	}

	return primitive.NilObjectID, errors.New("specified team does not exist in the workspace")
}

func (ms *MessengerServiceImpl) validateTicketStatus(status string) (entity.TicketStatus, error) {
	switch entity.TicketStatus(status) {
	case entity.StatusOpen, entity.StatusPending, entity.StatusClosed:
		return entity.TicketStatus(status), nil
	default:
		return "", fmt.Errorf("invalid ticket status: %s", status)
	}
}

func (ms *MessengerServiceImpl) findLeastBusyMember(team map[primitive.ObjectID]entity.UserStatus) (primitive.ObjectID, error) {
	var leastBusyMember primitive.ObjectID
	minTickets := int(^uint(0) >> 1)

	findMember := func(status entity.UserStatus) bool {
		for memberId, userStatus := range team {
			if userStatus != status {
				continue
			}
			activeTicketsCount, err := ms.messengerRepo.CountActiveTickets(memberId)
			if err != nil {
				continue
			}
			if activeTicketsCount < minTickets {
				minTickets = activeTicketsCount
				leastBusyMember = memberId
			}
		}
		return !leastBusyMember.IsZero()
	}

	if findMember(entity.StatusAvailable) || findMember(entity.StatusBusy) || findMember(entity.StatusOffline) {
		return leastBusyMember, nil
	}

	return primitive.NilObjectID, errors.New("no suitable team member found")
}

func (ms *MessengerServiceImpl) findTicketInChat(chat *entity.Chat, ticketId string) (*entity.Ticket, error) {
	for _, ticket := range chat.Tickets {
		if ticket.TicketId == ticketId {
			return &ticket, nil
		}
	}
	return nil, errors.New("ticket not found")
}

func (ms *MessengerServiceImpl) createChat(currentChat *entity.Chat, ticket entity.Ticket, workspaceId, assigneeId primitive.ObjectID) *entity.Chat {
	return &entity.Chat{
		UserId:      assigneeId,
		WorkspaceId: workspaceId,
		ChatId:      uuid.New().String(),
		TgClientId:  currentChat.TgClientId,
		Tickets:     []entity.Ticket{ticket},
		Notes:       []entity.Note{},
		Tags:        []string{},
		Source:      currentChat.Source,
		CreatedAt:   time.Now(),
	}
}

func (ms *MessengerServiceImpl) createNote(userId primitive.ObjectID, message string) *entity.Note {
	return &entity.Note{
		UserId:    userId,
		Text:      message,
		NoteId:    uuid.New().String(),
		CreatedAt: time.Now(),
	}
}

func (ms *MessengerServiceImpl) createMessageResponse(content []byte, createdAt time.Time, ticketId, chatId, message, messageType string) *model.MessageResponse {
	return &model.MessageResponse{
		TicketId:  ticketId,
		ChatId:    chatId,
		Message:   message,
		Content:   content,
		Type:      messageType,
		CreatedAt: createdAt,
	}
}

func (ms *MessengerServiceImpl) isAdmin(userRole entity.WorkspaceRole) bool {
	return userRole == entity.RoleAdmin
}

func (ms *MessengerServiceImpl) isOwner(userRole entity.WorkspaceRole) bool {
	return userRole == entity.RoleOwner
}
