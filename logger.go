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
	"fmt"
	flag "github.com/ogier/pflag"
	"os"
	"time"
)

var terminalOutput = flag.BoolP("stdout", "o", false, "")

var day int
var log chan []byte
var stop chan bool

func init() {
	go open()
}

func open() {
	fmt.Println("[DEBUG] Opening log file", time.Now().Format("2006-01-02"))
	day = time.Now().Day()

	// Open the file
	newfile, err := os.OpenFile(time.Now().Format("2006-01-02")+".log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		panic(err)
	}
	if newfile == nil {
		panic(os.ErrInvalid)
	}

	file := newfile
	writer := bufio.NewWriter(file)
	log = make(chan []byte, 32)
	if stop == nil {
		stop = make(chan bool, 1)
	}

	ioLoop(file, writer)
}

func ioLoop(file *os.File, writer *bufio.Writer) {
	var lines int
	for {
		select {
		case msg, ok := <-log:
			if !ok {
				continue
			}
			if day != time.Now().Day() {
				writer.Flush()
				file.Sync()
				file.Close()
				go open()
				return
			}
			if *terminalOutput {
				os.Stdout.Write(msg)
			}
			// Write it to the log file.
			_, err := writer.Write(msg)
			if err != nil {
				panic(err)
			}
			lines++
			// Flush the file if needed
			if lines == 5 {
				lines = 0
				writer.Flush()
			}
		case <-stop:
			writer.Flush()
			file.Sync()
			file.Close()
			return
		}
	}
}

func logf(message string, args ...interface{}) {
	log <- []byte(fmt.Sprintf(message, args...))
}

func stopLogger() {
	stop <- true
}
