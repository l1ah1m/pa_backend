package controller

import (
	"encoding/json"
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
		config:           cfg,
	}
}

// WSHandler handles WebSocket connections for real-time messaging.
// @Summary Handles WebSocket connections.
// @Tags Messenger
// @Produce json
// @Param id path string true "Workspace ID"
// @Param userId path string true "User ID"
// @Success 200 {object} model.SuccessResponse "Connection upgraded successfully"
// @Failure 400 {object} model.ErrorResponse "Bad request, user not valid in workspace"
// @Failure 500 {object} model.ErrorResponse "Internal server error, failed to upgrade connection"
// @Router /messenger/ws/{id} [get]
func (mc *MessengerController) WSHandler(c echo.Context) error {
	userId := c.Request().Context().Value("userId").(primitive.ObjectID)
	workspaceId := c.Param("id")

	err := mc.messengerService.ValidateUserInWorkspace(userId, workspaceId)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
	}

	ws, err := mc.websocketService.UpgradeConnection(c.Response(), c.Request(), workspaceId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: err.Error()})
	}
	//mc.websocketService.

	go func() {
		defer mc.websocketService.RemoveConnection(workspaceId, ws)
		for {
			_, message, err := ws.ReadMessage()
			if err != nil {
				break
			}

			var receivedMessage model.MessageRequest
			if err := json.Unmarshal(message, &receivedMessage); err != nil {
				continue
			}

			if receivedMessage.Source == "telegramBot" {

			}
		}
	}()

	return c.JSON(http.StatusOK, model.SuccessResponse{Message: "connection upgraded successfully"})
}

// ReassignTicketToTeam reassigns a support ticket to a different team.
// @Summary Reassigns a support ticket to a team.
// @Tags Messenger
// @Accept json
// @Produce json
// @Param ticket_id path string true "Ticket ID"
// @Param id path string true "Workspace ID"
// @Param name path string true "Team name"
// @Param userId path string true "User ID"
// @Success 200 {object} model.SuccessResponse "Ticket successfully reassigned to team"
// @Failure 500 {object} model.ErrorResponse "Internal server error, failed to reassign ticket"
// @Router /messenger/ticket/reassign/team [post]
func (mc *MessengerController) ReassignTicketToTeam(c echo.Context) error {
	var request model.ReassignTicketToTeamRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "invalid request parameters"})
	}

	userId := c.Request().Context().Value("userId").(primitive.ObjectID)
	if err := mc.messengerService.ReassignTicketToTeam(userId, request.TgClientId, request.TicketId, request.WorkspaceId, request.TeamName); err != nil {
		return c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, model.SuccessResponse{Message: "ticket reassigned successfully"})
}

func (mc *MessengerController) ReassignTicketToUser(c echo.Context) error {
	var request model.ReassignTicketToUserRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "invalid request parameters"})
	}

	userId := c.Request().Context().Value("userId").(primitive.ObjectID)
	if err := mc.messengerService.ReassignTicketToUser(userId, request.TgClientId, request.TicketId, request.WorkspaceId, request.Email); err != nil {
		return c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, model.SuccessResponse{Message: "ticket reassigned successfully"})
}

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
