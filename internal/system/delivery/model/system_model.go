package model

import "time"

// Requests

type CreateWorkspaceRequest struct {
	Name        string `json:"name"`
	Logo        []byte `json:"logo"`
	WorkspaceId string `json:"workspace_id"`
}

type CreateTeamRequest struct {
	TeamName    string            `json:"team_name"`
	Members     map[string]string `json:"members"`
	Logo        []byte            `json:"logo"`
	WorkspaceId string            `json:"workspace_id"`
}

type UpdateTeamRequest struct {
	TeamId      string `json:"team_id"`
	NewTeamName string `json:"team_name"`
	Logo        []byte `json:"logo"`
	WorkspaceId string `json:"workspace_id"`
}

type AddTeamMembersRequest struct {
	TeamId      string            `json:"team_id"`
	WorkspaceId string            `json:"workspace_id"`
	Members     map[string]string `json:"members"`
}

type AddWorkspaceMemberRequest struct {
	Team        map[string]string `json:"team"`
	WorkspaceId string            `json:"workspace_id"`
}

type UpdateWorkspaceMemberRequest struct {
	Team        map[string]string `json:"team"`
	WorkspaceId string            `json:"workspace_id"`
}

type UpdateWorkspaceRequest struct {
	Name        string `json:"name"`
	Logo        []byte `json:"logo"`
	WorkspaceId string `json:"workspace_id"`
}

type EditFoldersRequest struct {
	WorkspaceId string              `json:"workspace_id"`
	Folders     map[string][]string `json:"folders"`
}

// Responses

type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Message string `json:"message"`
}

type WorkspaceResponse struct {
	Name        string `json:"name"`
	Logo        []byte `json:"logo"`
	WorkspaceId string `json:"workspace_id"`
}

type UserResponse struct {
	Email                string        `json:"email"`
	FullName             string        `json:"name"`
	Role                 string        `json:"role"`
	Logo                 []byte        `json:"logo"`
	InTeam               bool          `json:"in_team"`
	AverageResponseTime  time.Duration `json:"average_response_time"`
	AverageRequestsCount int           `json:"average_requests_count"`
	Status               string        `json:"status"`
	Teams                []TeamData    `json:"teams"`
}

type TeamData struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

type FoldersResponse struct {
	Folders map[string][]string `json:"folders"`
}

type TeamResponse struct {
	TeamId      string   `json:"team_id"`
	TeamName    string   `json:"team_name"`
	MemberCount int      `json:"member_count"`
	AdminNames  []string `json:"admin_names"`
	ChatCount   int      `json:"chat_count"`
	Logo        []byte   `json:"logo"`
}
