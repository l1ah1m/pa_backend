package service

import (
	"errors"
	"fmt"
	"github.com/Point-AI/backend/config"
	"github.com/Point-AI/backend/internal/system/delivery/model"
	"github.com/Point-AI/backend/internal/system/domain/entity"
	_interface "github.com/Point-AI/backend/internal/system/domain/interface"
	"github.com/Point-AI/backend/internal/system/infrastructure/model"
	infrastructureInterface "github.com/Point-AI/backend/internal/system/service/interface"
	"github.com/Point-AI/backend/utils"
	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SystemServiceImpl struct {
	systemRepo   infrastructureInterface.SystemRepository
	emailService _interface.EmailService
	fileService  _interface.FileService
	config       *config.Config
}

func NewSystemServiceImpl(cfg *config.Config, systemRepo infrastructureInterface.SystemRepository, emailService _interface.EmailService, fileService _interface.FileService) *SystemServiceImpl {
	return &SystemServiceImpl{
		systemRepo:   systemRepo,
		emailService: emailService,
		fileService:  fileService,
		config:       cfg,
	}
}

func (ss *SystemServiceImpl) CreateWorkspace(logo []byte, ownerId primitive.ObjectID, workspaceId, name string) error {
	if err := utils.ValidateWorkspaceId(workspaceId); err != nil {
		return err
	}

	//var finalLogo []byte
	var err error

	if logo != nil {
		go ss.fileService.SaveFile("wp."+workspaceId, logo)
		//if finalLogo, err = utils.ValidatePhoto(logo); err != nil {
		//	return err
		//}
	}

	_, err = ss.systemRepo.FindWorkspaceByWorkspaceId(workspaceId)
	if err != nil {
		return err
	}

	if err := ss.systemRepo.CreateWorkspace(ownerId, workspaceId, name); err != nil {
		return err
	}

	return nil
}

func (ss *SystemServiceImpl) LeaveWorkspace(workspaceId string, userId primitive.ObjectID) error {
	workspace, err := ss.systemRepo.FindWorkspaceByWorkspaceId(workspaceId)
	if err != nil {
		return err
	}

	user, err := ss.systemRepo.FindUserById(userId)
	if err != nil {
		return err
	}

	internalTeams, _ := ss.systemRepo.FindTeamsByWorkspaceId(workspace.Id)

	for _, internalTeam := range internalTeams {
		if _, exists := internalTeam.Members[user.Id]; exists {
			delete(internalTeam.Members, user.Id)
			if err = ss.systemRepo.UpdateTeam(internalTeam); err != nil {
				return err
			}
		}
	}

	return ss.systemRepo.RemoveUserFromWorkspace(workspace, userId)
}

func (ss *SystemServiceImpl) SetFirstTeam(userId primitive.ObjectID, teamId, workspaceId string) error {
	workspace, err := ss.systemRepo.FindWorkspaceByWorkspaceId(workspaceId)
	if err != nil {
		return err
	}

	if !ss.isAdmin(workspace.Team[userId]) && !ss.isOwner(workspace.Team[userId]) {
		return errors.New("unauthorised")
	}

	internalTeam, err := ss.systemRepo.FindTeamByTeamIdAndWorkspaceId(teamId, workspace.Id)
	if err != nil {
		return err
	}

	internalTeam.IsFirstTeam = true

	return ss.systemRepo.UpdateTeam(internalTeam)
}

func (ss *SystemServiceImpl) AddTeamsMembers(userId primitive.ObjectID, members map[string]string, teamId, workspaceId string) error {
	workspace, err := ss.systemRepo.FindWorkspaceByWorkspaceId(workspaceId)
	if err != nil {
		return err
	}

	if workspace.Team == nil {
		workspace.Team = make(map[primitive.ObjectID]entity.WorkspaceRole)
	}
	if workspace.PendingTeam == nil {
		workspace.PendingTeam = make(map[string]entity.WorkspaceRole)
	}

	if !ss.isAdmin(workspace.Team[userId]) && !ss.isOwner(workspace.Team[userId]) {
		return errors.New("unauthorised")
	}

	internalTeam, err := ss.systemRepo.FindTeamByTeamIdAndWorkspaceId(teamId, workspace.Id)
	if err != nil {
		return err
	}

	if internalTeam.Members == nil {
		internalTeam.Members = make(map[primitive.ObjectID]bool)
	}
	if internalTeam.PendingMembers == nil {
		internalTeam.PendingMembers = make(map[string]bool)
	}

	if members != nil {
		teamRoles, pendingTeamRoles, err := ss.systemRepo.ValidateTeam(members, userId)
		if err != nil {
			return err
		}

		for id, role := range teamRoles {
			if _, exists := workspace.Team[id]; !exists {
				workspace.Team[id] = role
				internalTeam.Members[id] = true
			} else {
				internalTeam.Members[id] = true
			}
		}

		for email, role := range pendingTeamRoles {
			emailHash, _ := utils.GenerateInvitationJWTToken(ss.config.Auth.JWTSecretKey, email)
			ss.emailService.SendWorkspaceInvitationEmail(email, fmt.Sprintf("%s/signin/confirm?id=%s&email=%s", ss.config.Website.WebURL, workspaceId, emailHash))

			if _, exists := workspace.PendingTeam[email]; !exists {
				workspace.PendingTeam[email] = role
				internalTeam.PendingMembers[email] = true
			}
		}
	}

	if err := ss.systemRepo.UpdateTeam(internalTeam); err != nil {
		return err
	}

	return ss.systemRepo.UpdateWorkspace(workspace)
}

// GetWorkspaceById TODO: update function not to return team
func (ss *SystemServiceImpl) GetWorkspaceById(workspaceId string, userId primitive.ObjectID) (infrastructureModel.Workspace, error) {
	workspace, err := ss.systemRepo.FindWorkspaceByWorkspaceId(workspaceId)
	if err != nil {
		return infrastructureModel.Workspace{}, err
	}

	if _, exists := workspace.Team[userId]; !exists {
		return infrastructureModel.Workspace{}, errors.New("user is not in the Workspace")
	}

	fmtWorkspace, err := ss.formatWorkspaces([]entity.Workspace{*workspace})
	if err != nil {
		return infrastructureModel.Workspace{}, err
	}

	return fmtWorkspace[0], nil
}

func (ss *SystemServiceImpl) GetAllWorkspaces(userId primitive.ObjectID) ([]infrastructureModel.Workspace, error) {
	workspaces, err := ss.systemRepo.FindWorkspacesByUser(userId)
	if err != nil {
		return []infrastructureModel.Workspace{}, err
	}

	fmtWorkspaces, err := ss.formatWorkspaces(*workspaces)
	if err != nil {
		return []infrastructureModel.Workspace{}, err
	}

	return fmtWorkspaces, err
}

func (ss *SystemServiceImpl) CreateTeam(userId primitive.ObjectID, workspaceId, teamName string, members map[string]string, logo []byte) error {
	workspace, err := ss.systemRepo.FindWorkspaceByWorkspaceId(workspaceId)
	if err != nil {
		return err
	}

	if !ss.isAdmin(workspace.Team[userId]) && !ss.isOwner(workspace.Team[userId]) {
		return errors.New("unauthorised")
	}

	internalTeam := ss.createTeam(workspace.Id, teamName, map[primitive.ObjectID]bool{}, map[string]bool{}, false)
	if members != nil {
		teamRoles, pendingTeamRoles, err := ss.systemRepo.ValidateTeam(members, userId)
		if err != nil {
			return err
		}

		for id, role := range teamRoles {
			if _, exists := workspace.Team[id]; !exists {
				workspace.Team[id] = role
				internalTeam.Members[id] = true
			} else {
				internalTeam.Members[id] = true
			}
		}

		for email, role := range pendingTeamRoles {
			if _, exists := workspace.PendingTeam[email]; !exists {
				workspace.PendingTeam[email] = role
				internalTeam.PendingMembers[email] = true
			}
		}
	}

	if logo != nil {
		ss.fileService.SaveFile("team."+internalTeam.TeamId, logo)
	}

	if err := ss.systemRepo.InsertNewTeam(internalTeam); err != nil {
		return err
	}

	return ss.systemRepo.UpdateWorkspace(workspace)
}

func (ss *SystemServiceImpl) DeleteTeam(userId primitive.ObjectID, workspaceId, teamId string) error {
	workspace, err := ss.systemRepo.FindWorkspaceByWorkspaceId(workspaceId)
	if err != nil {
		return err
	}

	if !ss.isOwner(workspace.Team[userId]) && !ss.isAdmin(workspace.Team[userId]) {
		return errors.New("unauthorized to make the changes")
	}

	team, err := ss.systemRepo.FindTeamByTeamIdAndWorkspaceId(teamId, workspace.Id)
	if err != nil {
		return err
	}

	if err = ss.systemRepo.UpdateChatTeamIdToNil(team.Id); err != nil {
		return err
	}

	return ss.systemRepo.DeleteTeam(team.Id)
}

func (ss *SystemServiceImpl) UpdateWorkspace(userId primitive.ObjectID, newLogo []byte, workspaceId, newWorkspaceId, newName string) error {
	workspace, err := ss.systemRepo.FindWorkspaceByWorkspaceId(workspaceId)
	if err != nil {
		return err
	}

	if !ss.isOwner(workspace.Team[userId]) && !ss.isAdmin(workspace.Team[userId]) {
		return errors.New("unauthorized to make the changes")
	}

	if newWorkspaceId != "" {
		if err := utils.ValidateWorkspaceId(workspaceId); err != nil {
			return err
		}
		if err := ss.fileService.UpdateFileName("wp."+workspace.WorkspaceId, "wp."+newWorkspaceId); err != nil {
			return err
		}
		workspace.WorkspaceId = newWorkspaceId
	}

	if newLogo != nil {
		compressedLogo, err := utils.ValidatePhoto(newLogo)
		if err != nil {
			return err
		}
		if err := ss.fileService.UpdateFile(compressedLogo, "wp."+workspace.WorkspaceId); err != nil {
			return err
		}
	}

	if newName != "" {
		workspace.Name = newName
	}

	return ss.systemRepo.UpdateWorkspace(workspace)
}

func (ss *SystemServiceImpl) AddWorkspaceMembers(userId primitive.ObjectID, team map[string]string, workspaceId string) error {
	workspace, err := ss.systemRepo.FindWorkspaceByWorkspaceId(workspaceId)
	if err != nil {
		return err
	}

	if !ss.isAdmin(workspace.Team[userId]) && !ss.isOwner(workspace.Team[userId]) {
		return errors.New("unauthorised")
	}

	teamRoles, pendingTeamRoles, err := ss.systemRepo.ValidateTeam(team, userId)
	if err != nil {
		return err
	}

	for email, _ := range pendingTeamRoles {
		emailHash, _ := utils.GenerateInvitationJWTToken(ss.config.Auth.JWTSecretKey, email)
		go ss.emailService.SendWorkspaceInvitationEmail(email, fmt.Sprintf("%s/signin/confirm?id=%s&email=%s", ss.config.Website.WebURL, workspaceId, emailHash))
	}

	return ss.systemRepo.AddUsersToWorkspace(workspace, teamRoles, pendingTeamRoles)
}

func (ss *SystemServiceImpl) UpdateWorkspaceMembers(userId primitive.ObjectID, team map[string]string, workspaceId string) error {
	workspace, err := ss.systemRepo.FindWorkspaceByWorkspaceId(workspaceId)
	if err != nil {
		return err
	}

	if ss.isAdmin(workspace.Team[userId]) || ss.isOwner(workspace.Team[userId]) {
		teamRoles, _, err := ss.systemRepo.ValidateTeam(team, userId)
		if err != nil {
			return err
		}

		if err := ss.systemRepo.UpdateUsersInWorkspace(workspace, teamRoles); err != nil {
			return err
		}
	}

	return errors.New("user does not have the permissions")
}

func (ss *SystemServiceImpl) DeleteWorkspaceMember(userId primitive.ObjectID, workspaceId, memberEmail string) error {
	workspace, err := ss.systemRepo.FindWorkspaceByWorkspaceId(workspaceId)
	if err != nil {
		return err
	}

	if !ss.isAdmin(workspace.Team[userId]) && !ss.isOwner(workspace.Team[userId]) {
		return errors.New("unauthorised")
	}

	user, err := ss.systemRepo.FindUserByEmail(memberEmail)
	if err != nil {
		return err
	}

	internalTeams, _ := ss.systemRepo.FindTeamsByWorkspaceId(workspace.Id)

	for _, internalTeam := range internalTeams {
		if _, exists := internalTeam.Members[user]; exists {
			delete(internalTeam.Members, user)
			if err = ss.systemRepo.UpdateTeam(internalTeam); err != nil {
				return err
			}
		}
	}

	return ss.systemRepo.RemoveUserFromWorkspace(workspace, user)
}

func (ss *SystemServiceImpl) DeleteWorkspaceById(workspaceId string, userId primitive.ObjectID) error {
	workspace, err := ss.systemRepo.FindWorkspaceByWorkspaceId(workspaceId)
	if err != nil {
		return err
	}

	if ss.isOwner(workspace.Team[userId]) {
		if err := ss.systemRepo.DeleteWorkspace(workspace.Id); err != nil {
			return err
		}
	}

	return errors.New("user does not have a valid permission")
}

func (ss *SystemServiceImpl) GetUserProfiles(workspaceId string, userId primitive.ObjectID) ([]infrastructureModel.User, error) {
	workspace, err := ss.systemRepo.FindWorkspaceByWorkspaceId(workspaceId)
	if err != nil {
		return nil, err
	}

	if _, exists := workspace.Team[userId]; exists {
		users, err := ss.systemRepo.GetUserProfiles(*workspace)
		if err != nil {
			return nil, err
		}

		for _, user := range *users {
			user.Logo, _ = ss.fileService.LoadFile("user." + user.Email)
		}

		return *users, nil
	}

	return nil, errors.New("user does not have a valid permission")
}

func (ss *SystemServiceImpl) RegisterTelegramIntegration(userId primitive.ObjectID, workspaceId, stage, value string) (int, error) {
	workspace, err := ss.systemRepo.FindWorkspaceByWorkspaceId(workspaceId)
	if err != nil {
		return 500, err
	}

	if _, exists := workspace.Team[userId]; !exists {
		return 500, errors.New("user does not have valid permissions")
	}

	client := resty.New()
	var resp *resty.Response

	switch stage {
	case "phone":
		reqBody := map[string]string{
			"workspace_id": workspaceId,
			"phone_number": value,
		}
		resp, err = client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Authorization", "Bearer "+ss.config.Auth.IntegrationsServerSecretKey).
			SetBody(reqBody).
			Post(ss.config.Website.IntegrationsServerURL + "/point_ai/telegram_wrapper/send_code")

		return resp.StatusCode(), err
	case "code":
		reqBody := map[string]string{
			"workspace_id": workspaceId,
			"code":         value,
		}
		resp, err = client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Authorization", "Bearer "+ss.config.Auth.IntegrationsServerSecretKey).
			SetBody(reqBody).
			Post(ss.config.Website.IntegrationsServerURL + "/point_ai/telegram_wrapper/verify_init_code")

		return resp.StatusCode(), err
	case "password":
		reqBody := map[string]string{
			"workspace_id": workspaceId,
			"password":     value,
		}
		resp, err = client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Authorization", "Bearer "+ss.config.Auth.IntegrationsServerSecretKey).
			SetBody(reqBody).
			Post(ss.config.Website.IntegrationsServerURL + "/point_ai/telegram_wrapper/verify_2fa_password")

		return resp.StatusCode(), err
	}

	return 500, errors.New("invalid authentication stage")
}

func (ss *SystemServiceImpl) formatWorkspaces(workspaces []entity.Workspace) ([]infrastructureModel.Workspace, error) {
	formattedWorkspaces := make([]infrastructureModel.Workspace, len(workspaces))
	for i, p := range workspaces {
		logo, _ := ss.fileService.LoadFile("wp." + p.WorkspaceId)
		team, _ := ss.systemRepo.FormatTeam(p.Team)

		formattedWorkspace := infrastructureModel.Workspace{
			Name:        p.Name,
			WorkspaceId: p.WorkspaceId,
			Team:        team,
			Logo:        logo,
		}

		formattedWorkspaces[i] = formattedWorkspace
	}

	return formattedWorkspaces, nil
}

func (ss *SystemServiceImpl) EditFolders(userId primitive.ObjectID, workspaceId string, folders map[string][]string) error {
	workspace, err := ss.systemRepo.FindWorkspaceByWorkspaceId(workspaceId)
	if err != nil {
		return err
	}

	if _, exists := workspace.Team[userId]; exists {
		workspace.Folders = folders
		return ss.systemRepo.UpdateWorkspace(workspace)
	}
	return errors.New("unauthorized")
}

func (ss *SystemServiceImpl) GetAllUsers(userId primitive.ObjectID, workspaceId, teamId string) ([]model.UserResponse, error) {
	workspace, err := ss.systemRepo.FindWorkspaceByWorkspaceId(workspaceId)
	if err != nil {
		return nil, err
	}

	if _, exists := workspace.Team[userId]; !exists {
		return nil, errors.New("unauthorized")
	}

	var team *entity.Team
	if teamId != "" {
		team, err = ss.systemRepo.FindTeamByTeamId(teamId)
		if err != nil {
			return nil, err
		}
	}

	var usersResponse []model.UserResponse
	for userId, _ := range workspace.Team {
		user, _ := ss.systemRepo.FindUserById(userId)
		logo, _ := ss.fileService.LoadFile("user." + user.Email)

		isMember := false
		if team != nil && team.Members != nil {
			if _, exists := team.Members[userId]; exists {
				isMember = true
			}
		}

		userTeams := ss.systemRepo.GetTeamNamesByUserId(userId)
		var teams []model.TeamData
		for _, pTeam := range userTeams {
			teams = append(teams, model.TeamData{
				Name: pTeam.TeamName,
				Id:   pTeam.TeamId,
			})
		}

		usersResponse = append(usersResponse, *ss.createUserResponse(user.Email, user.FullName, string(workspace.Team[userId]), string(user.Status), logo, isMember, teams))
	}

	return usersResponse, nil
}

func (ss *SystemServiceImpl) UpdateWorkspacePendingStatus(userId primitive.ObjectID, workspaceId string, status bool) error {
	if err := ss.systemRepo.ClearPendingStatus(userId, workspaceId); err != nil {
		return err
	}

	if err := ss.systemRepo.UpdateWorkspaceUserStatus(userId, workspaceId, status); err != nil {
		return err
	}
	return nil
}

func (ss *SystemServiceImpl) UpdateTeam(userId primitive.ObjectID, newLogo []byte, workspaceId, newTeamName, teamId string) error {
	workspace, err := ss.systemRepo.FindWorkspaceByWorkspaceId(workspaceId)
	if err != nil {
		return err
	}

	if !ss.isAdmin(workspace.Team[userId]) && !ss.isOwner(workspace.Team[userId]) {
		return errors.New("unauthorised")
	}

	team, err := ss.systemRepo.FindTeamByTeamIdAndWorkspaceId(teamId, workspace.Id)
	if err != nil {
		return err
	}

	team.TeamName = newTeamName
	if newLogo != nil {
		ss.fileService.UpdateFile(newLogo, "team."+teamId)
	}

	return ss.systemRepo.UpdateTeam(team)
}

func (ss *SystemServiceImpl) GetAllTeams(userId primitive.ObjectID, workspaceId string) ([]model.TeamResponse, int, error) {
	workspace, err := ss.systemRepo.FindWorkspaceByWorkspaceId(workspaceId)
	if err != nil {
		return nil, 500, err
	}
	if workspace == nil {
		return nil, 204, err
	}

	if _, exists := workspace.Team[userId]; !exists {
		return nil, 403, errors.New("unauthorised")
	}

	internalTeams, err := ss.systemRepo.FindTeamsByWorkspaceId(workspace.Id)
	if err != nil {
		return nil, 500, err
	}
	if internalTeams == nil {
		return nil, 204, nil
	}

	var teamsResponse []model.TeamResponse
	for _, team := range internalTeams {
		var memberCount int
		var admins []string

		for userId, _ := range team.Members {
			memberCount++
			if ss.isAdmin(workspace.Team[userId]) || ss.isOwner(workspace.Team[userId]) {
				user, _ := ss.systemRepo.FindUserById(userId)
				admins = append(admins, user.FullName)
			}
		}
		number, _ := ss.systemRepo.CountChatsByTeamId(team.Id)

		logo, _ := ss.fileService.LoadFile("team." + team.TeamId)
		teamsResponse = append(teamsResponse, *ss.createTeamResponse(team.TeamName, team.TeamId, memberCount, number, admins, logo))
	}

	return teamsResponse, 200, nil
}

func (ss *SystemServiceImpl) GetAllFolders(userId primitive.ObjectID, workspaceId string) (map[string][]string, error, int) {
	workspace, err := ss.systemRepo.FindWorkspaceByWorkspaceId(workspaceId)
	if err != nil {
		return nil, err, 500
	}

	if _, exists := workspace.Team[userId]; !exists {
		return nil, errors.New("unauthorised"), 403
	}

	return workspace.Folders, nil, 200
}

func (ss *SystemServiceImpl) createTeamResponse(teamName, teamId string, memberCount, chatCount int, adminNames []string, logo []byte) *model.TeamResponse {
	return &model.TeamResponse{
		TeamId:      teamId,
		TeamName:    teamName,
		MemberCount: memberCount,
		AdminNames:  adminNames,
		ChatCount:   chatCount,
		Logo:        logo,
	}
}

func (ss *SystemServiceImpl) createTeam(workspaceId primitive.ObjectID, teamName string, members map[primitive.ObjectID]bool, pendingMembers map[string]bool, isFirstTeam bool) *entity.Team {
	return &entity.Team{
		WorkspaceId:    workspaceId,
		TeamId:         uuid.New().String(),
		TeamName:       teamName,
		Members:        members,
		PendingMembers: pendingMembers,
		IsFirstTeam:    isFirstTeam,
	}
}

func (ss *SystemServiceImpl) createUserResponse(email, fullName, role, status string, logo []byte, inTeam bool, teams []model.TeamData) *model.UserResponse {
	return &model.UserResponse{
		Email:                email,
		FullName:             fullName,
		Role:                 role,
		Logo:                 logo,
		AverageResponseTime:  0,
		AverageRequestsCount: 0,
		Status:               status,
		Teams:                teams,
		InTeam:               inTeam,
	}
}

func (ss *SystemServiceImpl) isAdmin(userRole entity.WorkspaceRole) bool {
	return userRole == entity.RoleAdmin
}

func (ss *SystemServiceImpl) isOwner(userRole entity.WorkspaceRole) bool {
	return userRole == entity.RoleOwner
}
