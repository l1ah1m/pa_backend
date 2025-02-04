package utils

import (
	"bytes"
	"errors"
	"github.com/Point-AI/backend/internal/system/domain/entity"
	"image"
)

func ValidateWorkspaceId(projectId string) error {
	if len(projectId) < 6 || len(projectId) > 30 {
		return errors.New("project ID must be between 6 and 30 characters")
	}

	for _, char := range projectId {
		if !isValidCharacter(char) {
			return errors.New("project ID can only contain lowercase alphanumeric characters and hyphen (-)")
		}
	}

	return nil
}

func isValidCharacter(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= '0' && char <= '9') ||
		char == '-'
}

func ValidatePhoto(photoBytes []byte) ([]byte, error) {
	_, _, err := image.Decode(bytes.NewReader(photoBytes))
	if err != nil {
		return nil, err
	}
	//if len(photoBytes) > 1024*1024 {
	//	return nil, errors.New("photo size cannot exceed 1MB")
	//}

	//if len(photoBytes) > 1024*512 {
	//	img, _, err := image.Decode(bytes.NewReader(photoBytes))
	//	if err != nil {
	//		return nil, err
	//	}
	//	var compressed bytes.Buffer
	//
	//	err = jpeg.Encode(&compressed, img, &jpeg.Options{Quality: 50})
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	return compressed.Bytes(), nil
	//}

	return photoBytes, nil

	//bounds := img.Bounds()
	//width := bounds.Dx()
	//height := bounds.Dy()
	//
	//if width != height {
	//	return errors.New("photo must be square")
	//}
	//
	//if width > 256 || height > 256 {
	//	return errors.New("photo dimensions cannot exceed 256x256 pixels")
	//}
}

func ValidateTeamRoles(team map[string]string) (map[string]entity.WorkspaceRole, error) {
	userRoles := make(map[string]entity.WorkspaceRole)
	for email, role := range team {
		switch role {
		case string(entity.RoleAdmin), string(entity.RoleAgent), string(entity.RoleOwner):
			userRoles[email] = entity.WorkspaceRole(role)
		default:
			userRoles[email] = entity.RoleAgent
		}
	}

	return userRoles, nil
}

func ValidateTeamNames(teams []string) error {
	teamMap := make(map[string]bool)

	for _, t := range teams {
		if _, exists := teamMap[t]; exists {
			return errors.New("duplicate team found: %s")
		}
		teamMap[t] = true
	}

	return nil
}
