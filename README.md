# ChatGPTBot
This repository contains a Go implementation for creating a WhatsApp or Telegram bot to interact with the ChatGPT OpenAI model. When running it will respond to messages sent by a specific user or sent to a specific group (to be configured at the top in `main.go`).

Some context about the services being used:
* ChatGPT is the underlying OpenAI model that is used to generate responses to user queries (see https://openai.com/blog/chatgpt/) 
* The WhatsApp linked devices functionality is used to communicate over WhatsApp. This is done using the https://github.com/tulir/whatsmeow Go library.
* For Telegram the Bot API is used (see https://core.telegram.org/bots/api). For this the Bot API Go bindings are used. See https://github.com/go-telegram-bot-api/telegram-bot-api.

Note: everything in this repository is meant for personal and educational use only.

## Getting Started
The below instructions assume that you have:
* Go 1.19+ installed (if not, follow the instructions at https://go.dev/doc/install)
* An OpenAI Chat account (if not, create one at https://chat.openai.com/auth/login)
* An active WhatsApp or Telegram client that you can use to link devices or create needed API keys

You can run the bot by executing the following command from within the root of the repository:

```bash
go run .
```

Note: authenticating to the ChatGPT API is a bit convoluted since the official OpenAI GPT-3 API doesn't support ChatGPT (at time of writing). When you run the bot it will attempt to fire up a browser session to intercept your JWT token (for this you need `chromium-browser` installed). If this is failing, you need to manually extract a JWT token of the ChatGPT session (login to https://chat.openai.com/auth/login and use your browser's develop tools to copy the JWT token).

## Authenticating to messengers
* Linking the app as a device in WhatsApp is as easy as using WhatsApp for web. On the first run the a QR-code will be printed to the terminal that needs to be scanned on your phone to give acces (WhatsApp -> Settings -> Linked Devices -> Link a device)
* For Telegram follow the instructions at https://core.telegram.org/bots/tutorial#obtain-your-bot-token to obtain a bot API token

