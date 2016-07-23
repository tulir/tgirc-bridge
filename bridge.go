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
	"bufio"
	flag "github.com/ogier/pflag"
	"os"
	"os/signal"
	"syscall"
)

var terminalInput = flag.BoolP("stdin", "i", false, "")

const version = "Telegram-IRC Bridge 0.3"

func init() {
	flag.Parse()
	groupSU = SimpleUser{Sender: *telegramGroup}
}

func main() {
	go startTelegram()
	go startIRC()

	interruptlist := func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		shutdown()
	}

	if *terminalInput {
		go interruptlist()
		reader := bufio.NewReader(os.Stdin)
		for {
			text, _ := reader.ReadString('\n')
			text = text[:len(text)-1]
			if text[0:1] == "/" {
				text = text[1:]
				if text == "stop" {
					shutdown()
				} else {
					logf("Unknown command: /%s", text)
				}
			} else {
				irc.Privmsg(*ircChannel, text)
				telegram.SendMessage(groupSU, text, md)
			}
		}
	} else {
		interruptlist()
	}
}

func shutdown() {
	stopIRC()
	stopLogger()
	os.Exit(0)
}
