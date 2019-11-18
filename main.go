package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/gookit/color"
)

type message struct {
	username []byte
	message  []byte
	id       int
}

func errPrintln(a ...interface{}) {
	color.New(color.FgRed, color.BgBlack).Println(a)
}

func printNewMessages(conn net.Conn, messages chan message) {
	printedMessages := make([]int, 0)
	for {
		curMessage := <-messages

		found := false
		for _, id := range printedMessages {
			if id == curMessage.id {
				found = true
				break
			}
		}
		if found {
			messages <- curMessage
			continue
		}

		printedMessages = append(printedMessages, curMessage.id)
		_, err := conn.Write([]byte(fmt.Sprintf("<%s>: %s", curMessage.username, curMessage.message)))
		if err != nil {
			errPrintln(err)
			break
		}
	}
}

func handleConnection(conn net.Conn, messages chan message, numClients *int, numClientsMutex *sync.Mutex, numMessages *int, numMessagesMutex *sync.Mutex) {
	fmt.Println("Handling connection ", conn.LocalAddr())

	username := make([]byte, 0)
	usernameBuff := make([]byte, 32)

	_, err := conn.Write([]byte("Please type a username: "))
	if err != nil {
		errPrintln(err)
	}

	for {
		n, err := conn.Read(usernameBuff)
		username = append(username, usernameBuff[:n]...)
		if err == io.EOF || bytes.Compare(username[len(username)-1:], []byte("\n")) == 0 {
			break
		}
		if err != nil {
			errPrintln(err)
			break
		}
	}
	username = username[:len(username)-1]

	_, err = conn.Write([]byte(fmt.Sprintf("Welcome to the matrix, %s\n\n", username)))
	if err != nil {
		errPrintln(err)
	}

	go printNewMessages(conn, messages)

	curMessage := make([]byte, 0)
	curMessageBuff := make([]byte, 32)

	for {
		isError := false
		for {
			n, err := conn.Read(curMessageBuff)
			curMessage = append(curMessage, curMessageBuff[:n]...)
			if err == io.EOF || bytes.Compare(curMessage[len(curMessage)-1:], []byte("\n")) == 0 {
				break
			}
			if err != nil {
				errPrintln(err)
				isError = true
				break
			}
		}
		if isError {
			break
		}
		numMessagesMutex.Lock()
		numClientsMutex.Lock()
		*numMessages++
		for i := 0; i < *numClients; i++ {
			messages <- message{username, curMessage, *numMessages}
		}
		numClientsMutex.Unlock()
		numMessagesMutex.Unlock()
		curMessage = make([]byte, 0)
	}
}

func main() {
	ln, err := net.Listen("tcp", ":8080")

	numClientsMutex := &sync.Mutex{}
	numClients := 0
	numMessagesMutex := &sync.Mutex{}
	numMessages := 0

	if err != nil {
		errPrintln(err)
	}
	messages := make(chan message)
	for {
		conn, err := ln.Accept()
		if err != nil {
			errPrintln(err)
		}
		numClientsMutex.Lock()
		numClients += 1
		numClientsMutex.Unlock()
		go handleConnection(conn, messages, &numClients, numClientsMutex, &numMessages, numMessagesMutex)
	}
}
