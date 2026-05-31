package authentication

import (
	"fmt"
	"strings"

	"tg/internal/config"
	"tg/internal/tdlib"
)

func RequireAuthenticatedSession() error {
	tdjson, err := tdlib.LoadTDJSON()
	if err != nil {
		return err
	}
	defer tdjson.Close()

	tdlib.ConfigureLogging(tdjson)
	clientID := tdjson.CreateClientID()

	state, err := tdlib.FetchAuthorizationState(tdjson, clientID)
	if err != nil {
		return err
	}
	if state != tdlib.AuthStateReady {
		return fmt.Errorf("not logged in (run `tg login`)")
	}

	return nil
}

func ResolveIdentityLabel(configuration config.Config, configPath string) (string, config.Config, error) {
	if configuration.Username != "" {
		return "@" + configuration.Username, configuration, nil
	}
	if configuration.PhoneNumber != "" {
		return configuration.PhoneNumber, configuration, nil
	}

	tdjson, err := tdlib.LoadTDJSON()
	if err != nil {
		return "", configuration, err
	}
	defer tdjson.Close()

	tdlib.ConfigureLogging(tdjson)
	clientID := tdjson.CreateClientID()

	user, err := tdlib.FetchCurrentUser(tdjson, clientID)
	if err != nil {
		return "", configuration, err
	}

	updatedConfig, err := PersistIdentity(configuration, configPath, user)
	if err != nil {
		return "", configuration, err
	}

	return BuildIdentityLabelFromUser(user), updatedConfig, nil
}

func PersistIdentity(configuration config.Config, configPath string, user tdlib.User) (config.Config, error) {
	configuration.Username = user.Username
	configuration.PhoneNumber = user.PhoneNumber
	if err := config.Save(configuration, configPath); err != nil {
		return configuration, err
	}

	return configuration, nil
}

func BuildIdentityLabelFromUser(user tdlib.User) string {
	if user.Username != "" {
		return "@" + user.Username
	}

	fullName := strings.TrimSpace(strings.Join([]string{user.FirstName, user.LastName}, " "))
	if fullName != "" {
		return fullName
	}

	if user.PhoneNumber != "" {
		return user.PhoneNumber
	}

	return fmt.Sprintf("user:%d", user.ID)
}
