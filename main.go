package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bartvanarnhem/chatgptbot/messagesource"
	"github.com/bartvanarnhem/chatgptbot/messenger"

	"github.com/joho/godotenv"
)

const (
	MessengerClientType = messenger.WhatsApp // or messenger.Telegram
)

func Getenv(key string, defaultValue string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}

	return defaultValue
}

func main() {
	// Load the .env file in the current directory
	err := godotenv.Load()
	if err != nil {
		fmt.Println("error loading environment from .env file")
	}

	// Get the targetID: the ID of the group or user that the bot should react to. These IDs are internal identifiers:
	// - For WhatsApp of the form xx@s.whatsapp.net for a user or xx@g.us for a group
	// - For Telegram a positive integer for a user or negative for a group
	//
	// You can find these IDs by simply running the bot and looking at the incoming messages (it will print sender and
	// recipient IDs).
	targetID := Getenv("TARGET_ID", "xx@g.us")

	fmt.Println("Connecting messenger... ")
	messenger, err := messenger.CreateMessenger(MessengerClientType)
	if err != nil {
		fmt.Printf("error initializing Telegram client: %s\n", err)
		return
	}

	fmt.Println("Connecting message source... ")
	messageSource, err := messagesource.NewChatGPTMessageSource()
	if err != nil {
		fmt.Printf("error initializing ChatGPT client: %s\n", err)
		return
	}

	// getAndResponse is a small helper func to get the response from the messageSource and send it to a specified
	// recipient using the messenger
	getAndSendResponse := func(recipient string, message string) {
		response, err := messageSource.GetResponse(message)
		if err != nil {
			fmt.Printf("error calling message source: %s\n", err)
			return
		}

		fmt.Printf("Responding: %s\n", response)

		err = messenger.SendMessage(recipient, response)
		if err != nil {
			fmt.Printf("error sending message: %s\n", err)
			return
		}
	}

	// Link them up
	messenger.OnMessage(func(sender string, recipient string, isGroup bool, message string) {
		fmt.Printf("Incoming message from %s to %s: %s\n", sender, recipient, message)

		// If this is a group message and the group is the target ip, reply back to the group
		if isGroup && recipient == targetID {
			getAndSendResponse(recipient, message)
		}

		// If not a group but the sender is the target id, reply back to the sender
		if !isGroup && sender == targetID {
			getAndSendResponse(sender, message)
		}

		// Else just ignore the message
	})

	// Listen to Ctrl+C (you can also do something else that prevents the program from exiting)
	fmt.Printf(
		"Everything is set up. Listening for messages and responding to ID %s... (Press Ctrl+C to quit)\n",
		targetID,
	)

	c := make(chan os.Signal)
	//nolint:govet,staticcheck
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	messenger.Disconnect()
}
