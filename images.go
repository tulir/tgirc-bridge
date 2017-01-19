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
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	_ "golang.org/x/image/webp"
)

// Telegram API constants
const (
	GetFile      = "https://api.telegram.org/bot%s/getFile?file_id=%s"
	DownloadFile = "https://api.telegram.org/file/bot%s/%s"
)

// Result ...
type Result struct {
	OK     bool `json:"ok"`
	Result File `json:"result"`
}

// File ...
type File struct {
	ID   string `json:"file_id"`
	Size int    `json:"file_size"`
	Path string `json:"file_path"`
}

// Download downloads the given file
func Download(name string) []byte {
	resp, err := http.DefaultClient.Get(fmt.Sprintf(DownloadFile, config.Telegram.Token, name))
	if err != nil {
		return []byte{}
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}
	}

	return data
}

// CreateDownload calls the getFile method in the Telegram API
func CreateDownload(id string) string {
	resp, err := http.DefaultClient.Get(fmt.Sprintf(GetFile, config.Telegram.Token, id))
	if err != nil {
		return ""
	}

	var data = Result{}

	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&data)
	if err != nil {
		return ""
	}

	return data.Result.Path
}

// MISData ...
type MISData struct {
	Image     string `json:"image"`
	Name      string `json:"image-name"`
	Format    string `json:"image-format,omitempty"`
	Client    string `json:"client-name"`
	Username  string `json:"username,omitempty"`
	AuthToken string `json:"auth-token,omitempty"`
	Hidden    bool   `json:"hidden"`
}

// MISResponse ...
type MISResponse struct {
	Success bool `json:"success"`
}

// MISUpload ...
func MISUpload(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	if http.DetectContentType(data) != "image/jpeg" {
		data = imageToJPG(data)
	}

	var dat = &MISData{
		Image:     base64.StdEncoding.EncodeToString(data),
		Name:      ImageName(5),
		Format:    "jpg",
		Client:    version,
		Username:  config.MIS.Username,
		AuthToken: config.MIS.Password,
		Hidden:    true,
	}

	data, err := json.Marshal(dat)
	if err != nil {
		return ""
	}

	resp, err := http.DefaultClient.Post(fmt.Sprintf("%s/%s", config.MIS.Address, "insert"), "text/json", bytes.NewReader(data))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var r = MISResponse{}
	dec := json.NewDecoder(resp.Body)
	dec.Decode(&r)

	if r.Success {
		return fmt.Sprintf("%s/%s.jpg", config.MIS.Address, dat.Name)
	}
	return ""
}

const imageNameAC = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789"
const (
	letterIdxBits = 6
	letterIdxMask = 1<<letterIdxBits - 1
	letterIdxMax  = 63 / letterIdxBits
)

var src = rand.NewSource(time.Now().UnixNano())

// ImageName generates a string matching [a-zA-Z0-9]{length}
func ImageName(length int) string {
	b := make([]byte, length)
	for i, cache, remain := 4, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(imageNameAC) {
			b[i] = imageNameAC[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

func imageToJPG(data []byte) []byte {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		fmt.Println("Error decoding image:", err)
		return data
	}
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, img, nil)
	if err != nil {
		fmt.Println("Error encoding image:", err)
		return data
	}
	return buf.Bytes()
}
