// Package utils provides utility functions for the storage package.
package utils //nolint:revive // this is a utility package and we need to keep the package name short

import (
	"fmt"
	"net/url"
)

// GetURI gets the URI for the storage backend.
func GetURI(inputUser, inputPassword, uri string) (string, error) {
	if inputUser != "" || inputPassword != "" {
		parsed, err := url.Parse(uri)
		if err != nil {
			return "", fmt.Errorf("parse postgres connection uri: %w", err)
		}
		username := ""
		switch {
		case inputUser != "":
			username = inputUser
		case parsed.User != nil:
			username = parsed.User.Username()
		default:
			username = ""
		}
		switch {
		case inputPassword != "":
			parsed.User = url.UserPassword(username, inputPassword)
		case parsed.User != nil:
			if password, ok := parsed.User.Password(); ok {
				parsed.User = url.UserPassword(username, password)
			} else {
				parsed.User = url.User(username)
			}
		default:
			parsed.User = url.User(username)
		}

		return parsed.String(), nil
	}
	return uri, nil
}
