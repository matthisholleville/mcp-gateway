// Package util provides cmd utility functions for the MCP Gateway.
//
//nolint:revive // we need to keep the functions as is for the cmd package
package util

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// MustBindPFlag attempts to bind a specific key to a pflag (as used by cobra) and panics
// if the binding fails with a non-nil error.
func MustBindPFlag(key string, flag *pflag.Flag) {
	if err := viper.BindPFlag(key, flag); err != nil {
		panic("failed to bind pflag: " + err.Error())
	}
}

// MustBindEnv binds an environment variable to a viper key.
func MustBindEnv(input ...string) {
	if err := viper.BindEnv(input...); err != nil {
		panic("failed to bind env key: " + err.Error())
	}
}
