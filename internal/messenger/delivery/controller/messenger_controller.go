package controller

import (
	"github.com/Point-AI/backend/config"
	"github.com/Point-AI/backend/internal/messenger/delivery/model"
	_interface "github.com/Point-AI/backend/internal/messenger/domain/interface"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
)

type MessengerController struct {
	messengerService _interface.MessengerService
	websocketService _interface.WebsocketService
	config           *config.Config
}

func NewMessengerController(cfg *config.Config, messengerService _interface.MessengerService, websocketService _interface.WebsocketService) *MessengerController {
	return &MessengerController{
		messengerService: messengerService,
		websocketService: websocketService,
		config:           cfg,
	}
}

// ChatWSHandler handles WebSocket connections for real-time messaging.
// @Summary Handles WebSocket connections.
// @Tags Messenger
// @Produce json
// @Param id path string true "Workspace ID"
// @Param userId path string true "User ID"
// @Success 200 {object} model.SuccessResponse "Connection upgraded successfully"
// @Failure 400 {object} model.ErrorResponse "Bad request, user not valid in workspace"
// @Failure 500 {object} model.ErrorResponse "Internal server error, failed to upgrade connection"
// @Router /messenger/ws/{id} [get]
func (mc *MessengerController) ChatWSHandler(c echo.Context) error {
	userId, _ := c.Get("userId").(primitive.ObjectID)
	workspaceId := c.QueryParam("id")

	if err := mc.messengerService.HandleChatWS(userId, workspaceId, c.Response(), c.Request()); err != nil {
		return c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "error happened while upgrading the connection"})
	}

	return c.JSON(http.StatusOK, model.SuccessResponse{Message: "connection upgraded successfully"})
}

// ReassignTicketToTeam reassigns a support ticket to a different team.
// @Summary Reassigns a support ticket to a team.
// @Tags Messenger
// @Accept json
// @Produce json
// @Param ticket_id path string true "Ticket ID"
// @Param request body model.ReassignTicketToTeamRequest true "details"
// @Success 200 {object} model.SuccessResponse "Ticket successfully reassigned to team"
// @Failure 400 {object} model.ErrorResponse "Invalid request parameters"
// @Failure 500 {object} model.ErrorResponse "Internal server error, failed to reassign ticket"
// @Router /messenger/ticket/reassign/team [post]
func (mc *MessengerController) ReassignTicketToTeam(c echo.Context) error {
	var request model.ReassignTicketToTeamRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "invalid request parameters"})
	}

	userId := c.Request().Context().Value("userId").(primitive.ObjectID)
	if err := mc.messengerService.ReassignTicketToTeam(userId, request.ChatId, request.TicketId, request.WorkspaceId, request.TeamName); err != nil {
		return c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, model.SuccessResponse{Message: "ticket reassigned successfully"})
}

// ReassignTicketToMember reassigns a support ticket to a different team member.
// @Summary Reassigns a support ticket to a team member.
// @Tags Messenger
// @Accept json
// @Produce json
// @Param request body model.ReassignTicketToUserRequest true "details"
// @Success 200 {object} model.SuccessResponse "Ticket successfully reassigned to member"
// @Failure 400 {object} model.ErrorResponse "Invalid request parameters"
// @Failure 500 {object} model.ErrorResponse "Internal server error, failed to reassign ticket"
// @Router /messenger/ticket/reassign/member [post]
func (mc *MessengerController) ReassignTicketToMember(c echo.Context) error {
	var request model.ReassignTicketToUserRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "invalid request parameters"})
	}

	userId := c.Request().Context().Value("userId").(primitive.ObjectID)
	if err := mc.messengerService.ReassignTicketToUser(userId, request.ChatId, request.TicketId, request.WorkspaceId, request.Email); err != nil {
		return c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, model.SuccessResponse{Message: "ticket reassigned successfully"})
}

// UpdateChatInfo updates the information of a chat in the messenger.
// @Summary Updates chat information.
// @Tags Messenger
// @Accept json
// @Produce json
// @Param request body model.UpdateChatInfoRequest true "details"
// @Success 200 {object} model.SuccessResponse "Chat information updated successfully"
// @Failure 400 {object} model.ErrorResponse "Invalid request parameters"
// @Failure 500 {object} model.ErrorResponse "Internal server error, failed to update chat information"
// @Router /messenger/chat [put]
func (mc *MessengerController) UpdateChatInfo(c echo.Context) error {
	var request model.UpdateChatInfoRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "invalid request parameters"})
	}

	userId := c.Request().Context().Value("userId").(primitive.ObjectID)
	if err := mc.messengerService.UpdateChatInfo(userId, request.ChatId, request.Tags, request.WorkspaceId, request.Language, request.Address, request.Company, request.ClientEmail, request.ClientPhone); err != nil {
		return c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, model.SuccessResponse{Message: "ticket reassigned successfully"})
}

// ChangeTicketStatus changes the status of a support ticket.
// @Summary Changes the status of a support ticket.
// @Tags Messenger
// @Accept json
// @Produce json
// @Param request body model.ChangeTicketStatusRequest true "details"
// @Success 200 {object} model.SuccessResponse "Ticket status updated successfully"
// @Failure 400 {object} model.ErrorResponse "Invalid request parameters"
// @Failure 500 {object} model.ErrorResponse "Internal server error, failed to update ticket status"
// @Router /messenger/ticket [put]
func (mc *MessengerController) ChangeTicketStatus(c echo.Context) error {
	var request model.ChangeTicketStatusRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "invalid request parameters"})
	}

	userId := c.Request().Context().Value("userId").(primitive.ObjectID)
	if err := mc.messengerService.UpdateTicketStatus(userId, request.TicketId, request.WorkspaceId, request.Status); err != nil {
		return c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, model.SuccessResponse{Message: "ticket status updated successfully"})
}

func (mc *MessengerController) SendOk(c echo.Context) error {
	return c.JSON(http.StatusOK, model.SuccessResponse{Message: "okay"})
}

// DeleteMessage removes a message from a chat in a workspace.
// @Summary Removes a message from a chat
// @Tags Messenger
// @Accept json
// @Produce json
// @Param request body model.MessageRequest true "details"
// @Success 200 {object} model.SuccessResponse "Message deleted successfully"
// @Failure 400 {object} model.ErrorResponse "Invalid request parameters"
// @Failure 500 {object} model.ErrorResponse "Internal server error, failed to delete the message"
// @Router /messenger/message [delete]
func (mc *MessengerController) DeleteMessage(c echo.Context) error {
	var request model.MessageRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "invalid request parameters"})
	}

	userId := c.Request().Context().Value("userId").(primitive.ObjectID)
	if err := mc.messengerService.DeleteMessage(userId, request.Type, request.WorkspaceId, request.TicketId, request.MessageId, request.ChatId); err != nil {
		return c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, model.SuccessResponse{Message: "message deleted successfully"})
}

func (mc *MessengerController) ImportTelegramChats(c echo.Context) error {
	workspaceId := c.Param("id")
	var request model.TelegramChatsRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "invalid request parameters"})
	}

	err := mc.messengerService.ImportTelegramChats(workspaceId, request.Chats)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, model.SuccessResponse{Message: "messages imported successfully"})
}

func (mc *MessengerController) GetAllChats(c echo.Context) error {
	workspaceId, chatsType := c.Param("id"), c.Param("type")
	userId := c.Request().Context().Value("userId").(primitive.ObjectID)

	switch chatsType {
	case "primary":
		chats, err := mc.messengerService.GetAllPrimaryChats(userId, workspaceId)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: err.Error()})
		}

		return c.JSON(http.StatusOK, chats)
	case "all":
		chats, err := mc.messengerService.GetAllChats(userId, workspaceId)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: err.Error()})
		}

		return c.JSON(http.StatusOK, chats)
	case "unassigned":
		chats, err := mc.messengerService.GetAllUnassignedChats(userId, workspaceId)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: err.Error()})
		}

		return c.JSON(http.StatusOK, chats)
	}

	return c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "invalid type"})
}

func (mc *MessengerController) GetChatsByFolder(c echo.Context) error {
	workspaceId, folderName := c.Param("id"), c.Param("name")
	userId := c.Request().Context().Value("userId").(primitive.ObjectID)

	chats, err := mc.messengerService.GetChatsByFolder(userId, workspaceId, folderName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, chats)
}

func (mc *MessengerController) GetChat(c echo.Context) error {
	workspaceId, chatId := c.Param("id"), c.Param("chat_id")
	userId := c.Request().Context().Value("userId").(primitive.ObjectID)

	chat, err := mc.messengerService.GetChat(userId, workspaceId, chatId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, chat)
}

func (mc *MessengerController) GetMessages(c echo.Context) error {
	userId := c.Request().Context().Value("userId").(primitive.ObjectID)
	var request model.GetMessagesRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "invalid request parameters"})
	}

	messages, err := mc.messengerService.GetMessages(userId, request.WorkspaceId, request.ChatId, request.LastMessageDate)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, messages)
}

func (mc *MessengerController) GetAllTags(c echo.Context) error {
	userId := c.Request().Context().Value("userId").(primitive.ObjectID)
	workspaceId := c.Param("id")

	tags, err := mc.messengerService.GetAllTags(userId, workspaceId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, tags)
}

//func (mc *MessengerController) HandleTelegramMessage(c echo.Context) error {
//	workspaceId := c.Param("id")
//
//	chats, err := mc.messengerService.GetAllChats(userId, workspaceId)
//	if err != nil {
//		return c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: err.Error()})
//	}
//
//	return c.JSON(http.StatusOK, chats)
//}
