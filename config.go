// tgirc-bridge - A Telegram <-> IRC bridge and chat logger
// Copyright (C) 2016 Tulir Asokan

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.
package main

import (
	"encoding/json"
	"io/ioutil"
)

// Config ...
type Config struct {
	Mappings map[string]string `json:"mappings"`

	Telegram Telegram `json:"telegram"`
	IRC      IRC      `json:"irc"`
	MIS      MIS      `json:"mis"`
}

// GetTelegramChannel ...
func (config *Config) GetTelegramChannel(ircChannel string) (SimpleUser, bool) {
	for key, val := range config.Mappings {
		if key == ircChannel {
			return SimpleUser{val}, true
		}
	}
	return SimpleUser{}, false
}

// GetIRCChannel ...
func (config *Config) GetIRCChannel(telegramChannel string) (string, bool) {
	for key, val := range config.Mappings {
		if val == telegramChannel {
			return key, true
		}
	}
	return "", false
}

// Telegram ...
type Telegram struct {
	Token string `json:"token"`
}

// IRC ...
type IRC struct {
	Address  string `json:"address"`
	User     string `json:"user"`
	Nick     string `json:"nick"`
	Password string `json:"password"`
	TLS      bool   `json:"tls"`
}

// MIS ...
type MIS struct {
	Address  string `json:"address"`
	Username string `json:"username"`
	Password string `json:"password"`
}

var config *Config

// LoadConfig loads the config
func LoadConfig() {
	config = &Config{}
	data, err := ioutil.ReadFile("config.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(data, config)
	if err != nil {
		panic(err)
	}
}
