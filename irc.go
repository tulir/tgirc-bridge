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
	"fmt"
	"strconv"
	"strings"
	"time"

	goirc "github.com/thoj/go-ircevent"
)

// Telegram message format when receiving from IRC
const (
	IRCMsgFormat    = "*<%[1]s>* %[2]s"
	IRCActionFormat = "*★ %[1]s* %[2]s"
)

var irc *goirc.Connection

func startIRC() {
	quit := make(chan bool)
	irc = goirc.IRC(config.IRC.Nick, config.IRC.User)
	irc.UseTLS = config.IRC.TLS
	irc.QuitMessage = "Bridge/logbot shutting down..."
	irc.Version = version
	err := irc.Connect(config.IRC.Address)
	if err != nil {
		logf(err.Error())
	}

	callback := func(channel, nick, message, command string) {
		tgChan, ok := config.GetTelegramChannel(channel)
		if !ok {
			logf("Unidentified IRC channel: %s\n", channel)
			return
		}

		logFmt := "IRCMESSAGE"
		if command == "message" {
			telegram.SendMessage(tgChan, fmt.Sprintf(IRCMsgFormat, nick, decodeIRC(message)), md)
		} else if command == "action" {
			telegram.SendMessage(tgChan, fmt.Sprintf(IRCActionFormat, nick, decodeIRC(message)), md)
			logFmt = "IRCACTION"
		}

		logf("%[4]s>%[1]d|%[2]s|%[3]s\n", time.Now().Unix(), nick, message, logFmt)
	}

	irc.AddCallback("PRIVMSG", func(event *goirc.Event) {
		callback(event.Arguments[0], event.Nick, event.Message(), "message")
	})

	irc.AddCallback("CTCP_ACTION", func(event *goirc.Event) {
		callback(event.Arguments[0], event.Nick, event.Message(), "action")
	})

	irc.AddCallback("001", func(event *goirc.Event) {
		for key := range config.Mappings {
			irc.Join(key)
		}
		if len(config.IRC.Password) > 0 {
			irc.Privmsgf("NickServ", "IDENTIFY %s", config.IRC.Password)
		}
		logf("[DEBUG] Successfully connected to IRC!\n")
	})

	irc.AddCallback("DISCONNECTED", func(event *goirc.Event) {
		logf("[DEBUG] Disconnected from IRC.\n")
		quit <- true
	})

	<-quit
}

func decodeIRC(msg string) string {
	msg = strings.Replace(msg, "*", "\\*", -1)
	msg = strings.Replace(msg, "_", "\\_", -1)
	msg = strings.Replace(msg, "\x02", "*", -1)
	msg = strings.Replace(msg, "\x1D", "_", -1)
	return msg
}

func ircmessage(ch int64, user, msg string) {
	channel, ok := config.GetIRCChannel(strconv.FormatInt(ch, 10))
	if !ok {
		logf("Unidentified Telegram group: %d\n", ch)
		return
	}

	for _, line := range Split(msg) {
		irc.Privmsgf(channel, "<%s> %s", user, line)
	}
}

func stopIRC() {
	irc.Quit()
}

// Split a message by newlines and if the message is longer than 250
// characters, split it into smaller pieces using SplitLen
func Split(message string) []string {
	splitted := []string{message}
	if strings.ContainsRune(message, '\n') {
		splitted = strings.Split(message, "\n")
	} else if len(message) > 250 {
		for len(splitted[len(splitted)-1]) > 250 {
			if len(splitted) < 2 {
				a, b := SplitLen(splitted[0])
				if len(b) != 0 {
					splitted = []string{a, b}
				} else {
					splitted = []string{a}
				}
			} else {
				a, b := SplitLen(splitted[len(splitted)-1])
				splitted[len(splitted)-1] = a
				if len(b) != 0 {
					splitted = append(splitted, b)
				}
			}
		}
	}
	return splitted
}

// SplitLen splits a message into pieces that are less than 250 characters long.
// If the message contains a space character before the character limit,
func SplitLen(message string) (string, string) {
	if len(message) < 250 {
		return message, ""
	}
	lastIndex := -1
	for i := 0; i < 250; i++ {
		if message[i] == ' ' {
			lastIndex = i
		}
	}

	if lastIndex == -1 {
		for i := 0; i < 250; i++ {
			if message[i] == '-' || message[i] == '.' || message[i] == ',' {
				lastIndex = i
			}
		}
	} else {
		return message[:lastIndex], message[lastIndex+1:]
	}

	if lastIndex != -1 {
		return message[:lastIndex+1], message[lastIndex+1:]
	}

	return message[:250], message[250:]
}
