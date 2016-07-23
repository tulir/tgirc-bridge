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
	"github.com/tucnak/telebot"
	"time"
)

// GoogleMaps ...
const GoogleMaps = "https://maps.google.com/maps?q=%[1]f,%[2]f&ll=%[1]f,%[2]f&z=16"

// Contact ...
const Contact = "%[1]s %[2]s (%[3]s)"

// SimpleUser is a simple wrapper for telebots Recepient
type SimpleUser struct {
	Sender string
}

// Destination ...
func (su SimpleUser) Destination() string {
	return su.Sender
}

var telegramGroup = flag.StringP("group", "g", "", "")
var telegramToken = flag.StringP("token", "t", "", "")

var telegram *telebot.Bot

var groupSU SimpleUser
var md *telebot.SendOptions

func init() {
	md = new(telebot.SendOptions)
	md.ParseMode = telebot.ModeMarkdown
}

func startTelegram() {
	// Connect to Telegram
	var err error
	telegram, err = telebot.NewBot(*telegramToken)
	if err != nil {
		logf("[DEBUG] Error connecting to Telegram: %[1]s\n", err)
		return
	}
	messages := make(chan telebot.Message)
	// Enable message listener
	telegram.Listen(messages, 1*time.Second)
	// Print "connected" message
	logf("[DEBUG] Successfully connected to Telegram!\n")

	// Listen to messages
	for message := range messages {
		go telegramMessage(message)
	}
}

func misUpload(text, fileID string) string {
	dl := CreateDownload(fileID)
	if len(dl) == 0 {
		return text
	}
	data := Download(dl)
	if len(data) == 0 {
		return text
	}
	url := MISUpload(data)
	if len(url) == 0 {
		return text
	}
	text = fmt.Sprintf("%s %s", url, text)
	return text
}

func telegramMessageData(message telebot.Message) telebot.Message {
	if len(message.Photo) > 0 && *useMis {
		message.Text = misUpload(message.Text, message.Photo[len(message.Photo)-1].FileID)
	} else if message.Sticker.Exists() {
		message.Text = misUpload(message.Text, message.Sticker.FileID)
	} else if message.Location.Latitude != 0 || message.Location.Longitude != 0 {
		message.Text = fmt.Sprintf(GoogleMaps, message.Location.Latitude, message.Location.Longitude)
	} else if message.Contact.UserID != 0 {
		message.Text = fmt.Sprintf(Contact, message.Contact.FirstName, message.Contact.LastName, message.Contact.PhoneNumber)
	} else if message.Document.Exists() && message.Document.Mime == "image/gif" {
		message.Text = misUpload(message.Text, message.Document.FileID)
	}
	return message
}

func telegramMessage(message telebot.Message) {
	message = telegramMessageData(message)
	if len(message.Text) == 0 {
		telegramLog(message)
		return
	}
	if message.IsForwarded() {
		// Type>ID|Timestamp|Username|UID|Text||ForwardTimestamp|ForwardUsername|ForwardUID
		logf("FORWARD>%[1]d|%[2]d|%[3]s|%[4]d|%[5]s§%[6]d|%[7]s|%[8]d\n",
			message.ID,
			message.Time().Unix(),
			message.Sender.Username,
			message.Sender.ID,
			message.Text,
			message.OriginalUnixtime,
			message.OriginalSender.Username,
			message.OriginalSender.ID,
		)
		ircmessage(message.Sender.Username, fmt.Sprintf("[fwd from %[2]s] %[1]s", message.Text, message.OriginalSender.Username))
	} else if message.IsReply() {
		// Type>ID|Timestamp|Username|UID|Text||ReplyID|ReplyTimestamp|ReplyUsername|ReplyUID|ReplyText
		logf("REPLY>%[1]d|%[2]d|%[3]s|%[4]d|%[5]s§%[6]d|%[7]d|%[8]s|%[9]d|%[10]s\n",
			message.ID,
			message.Time().Unix(),
			message.Sender.Username,
			message.Sender.ID,
			message.Text,
			message.ReplyTo.ID,
			message.ReplyTo.Time().Unix(),
			message.ReplyTo.Sender.Username,
			message.ReplyTo.Sender.ID,
			message.ReplyTo.Text,
		)
		ircmessage(message.Sender.Username, fmt.Sprintf("[reply to %[2]s] %[1]s", message.Text, message.ReplyTo.Sender.Username))
	} else {
		// Type>ID|Timestamp|Username|UID|Text
		logf("MESSAGE>%[1]d|%[2]d|%[3]s|%[4]d|%[5]s\n",
			message.ID,
			message.Time().Unix(),
			message.Sender.Username,
			message.Sender.ID,
			message.Text,
		)
		ircmessage(message.Sender.Username, message.Text)
	}
}

func telegramLog(message telebot.Message) {
	if message.Audio.Exists() {
		message.Text = "DATA_AUDIO"
	} else if message.Video.Exists() {
		message.Text = "DATA_VIDEO"
	} else if len(message.Photo) > 0 {
		message.Text = "DATA_PHOTO"
	} else if message.Sticker.Exists() {
		message.Text = "DATA_STICKER"
	} else if message.UserJoined.ID != 0 {
		// Type>ID|Timestamp|Username|UID
		logf("JOIN>%[1]d|%[2]d|%[3]s|%[4]d\n",
			message.ID,
			message.Time().Unix(),
			message.UserJoined.Username,
			message.UserJoined.ID,
		)
		ircmessage(message.Sender.Username, "* joined the group")
		return
	} else if message.UserLeft.ID != 0 {
		// Type>ID|Timestamp|Username|UID
		logf("LEAVE>%[1]d|%[2]d|%[3]s|%[4]d\n",
			message.ID,
			message.Time().Unix(),
			message.UserJoined.Username,
			message.UserJoined.ID,
		)
		ircmessage(message.Sender.Username, "* left the group")
		return
	} else if len(message.NewChatTitle) > 0 {
		message.Text = "Group title changed to " + message.NewChatTitle
	} else if message.Document.Exists() {
		message.Text = "DATA_DOCUMENT-" + message.Document.Mime
	} else {
		message.Text = "DATA_UNKNOWN"
	}
	// Type>ID|Timestamp|Username|UID|Text
	logf("DATA>%[1]d|%[2]d|%[3]s|%[4]d|%[5]s\n",
		message.ID,
		message.Time().Unix(),
		message.Sender.Username,
		message.Sender.ID,
		message.Text,
	)
}
