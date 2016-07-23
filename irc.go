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
	flag "github.com/ogier/pflag"
	goirc "github.com/thoj/go-ircevent"
	"strings"
	"time"
)

var ircChannel = flag.StringP("channel", "c", "#telegram", "")
var ircAddr = flag.StringP("address", "a", "", "")
var ircName = flag.StringP("username", "u", "logbot", "")
var ircPassword = flag.StringP("password", "p", "", "")
var ircNick = flag.StringP("nick", "n", "tgbridge", "")

var irc *goirc.Connection

func startIRC() {
	quit := make(chan bool)
	irc = goirc.IRC(*ircNick, *ircName)
	irc.UseTLS = true
	irc.QuitMessage = "Bridge/logbot shutting down..."
	irc.Version = version
	err := irc.Connect(*ircAddr)
	if err != nil {
		logf(err.Error())
	}
	irc.AddCallback("PRIVMSG", func(event *goirc.Event) {
		if event.Arguments[0] == *ircChannel {
			telegram.SendMessage(groupSU, fmt.Sprintf("*<%[1]s>* %[2]s", event.Nick, filterMarkdown(event.Message())), md)
			// Type>Timestamp|Username|Text
			logf("IRCMESSAGE>%[1]d|%[2]s|%[3]s\n",
				time.Now().Unix(),
				event.Nick,
				event.Message(),
			)
		} else {
			logf("UIRCMESSAGE>%[1]d|%[2]s|%[3]s\n",
				time.Now().Unix(),
				event.Nick,
				event.Message(),
			)
		}
	})

	irc.AddCallback("CTCP_ACTION", func(event *goirc.Event) {
		if event.Arguments[0] == *ircChannel {
			telegram.SendMessage(groupSU, fmt.Sprintf("*★ %[1]s* %[2]s", event.Nick, filterMarkdown(event.Message())), md)
			// Type>Timestamp|Username|Text
			logf("IRCACTION>%[1]d|%[2]s|%[3]s\n",
				time.Now().Unix(),
				event.Nick,
				event.Message(),
			)
		}
	})

	irc.AddCallback("001", func(event *goirc.Event) {
		irc.Join(*ircChannel)
		if len(*ircPassword) > 0 {
			irc.Privmsgf("NickServ", "IDENTIFY %s", *ircPassword)
		}
		logf("[DEBUG] Successfully connected to IRC!\n")
	})

	irc.AddCallback("DISCONNECTED", func(event *goirc.Event) {
		logf("[DEBUG] Disconnected from IRC.\n")
		quit <- true
	})

	<-quit
}

func filterMarkdown(msg string) string {
	return strings.Replace(strings.Replace(msg, "*", "\\*", -1), "_", "\\_", -1)
}

func ircmessage(user, msg string) {
	for _, line := range Split(msg) {
		irc.Privmsgf(*ircChannel, "<%s> %s", user, line)
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
