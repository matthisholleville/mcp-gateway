package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetURI(t *testing.T) {
	for _, test := range []struct {
		name     string
		username string
		password string
		uri      string
		expected string
	}{
		{
			name:     "with uri",
			username: "",
			password: "",
			uri:      "postgresql://mcp-gateway:changeme@localhost:5439/mcp-gateway?sslmode=disable",
			expected: "postgresql://mcp-gateway:changeme@localhost:5439/mcp-gateway?sslmode=disable",
		},
		{
			name:     "with username and password",
			username: "mcp-gateway",
			password: "changeme",
			uri:      "postgresql://postgres:postgres@localhost:5439/mcp-gateway?sslmode=disable",
			expected: "postgresql://mcp-gateway:changeme@localhost:5439/mcp-gateway?sslmode=disable",
		},
		{
			name:     "with username",
			username: "mcp-gateway",
			password: "",
			uri:      "postgresql://postgres:postgres@localhost:5439/mcp-gateway?sslmode=disable",
			expected: "postgresql://mcp-gateway:postgres@localhost:5439/mcp-gateway?sslmode=disable",
		},
		{
			name:     "with password",
			username: "",
			password: "changeme",
			uri:      "postgresql://postgres:postgres@localhost:5439/mcp-gateway?sslmode=disable",
			expected: "postgresql://postgres:changeme@localhost:5439/mcp-gateway?sslmode=disable",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			uri, err := GetURI(test.username, test.password, test.uri)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, uri)
		})
	}
}
