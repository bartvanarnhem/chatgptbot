package messenger

import (
	"fmt"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
)

const (
	BotAPITokenEnvName = "TELEGRAM_BOT_API_TOKEN"
	UpdatesTimeout     = 60
)

var (
	ErrNoTokenSpecified = errors.New("no bot API token available (please set the TELEGRAM_BOT_API_TOKEN environment variable)")
)

// TelegramMessenger uses the Telegram Bot API to send and receive Telegram messages. To obtain a bot token follow the
// instructions at https://core.telegram.org/bots/tutorial#obtain-your-bot-token.
//
// By default your bot can be added to any group. If you want to disable this use /setjoingroups
// Also by default, in group chats, your bot will only receive messages that either start with a / (bot command) or
// mention your bot's username. This can be disabled (making it receive all messages) by using /setprivacy.
type TelegramMessenger struct {
	baseMessenger
	bot *tgbotapi.BotAPI
}

// Ensure we implement the Messenger interface
var _ Messenger = (*TelegramMessenger)(nil)

func NewTelegramMessenger() (*TelegramMessenger, error) {
	messenger := TelegramMessenger{}

	err := messenger.init()
	if err != nil {
		return nil, err
	}

	messenger.consumeUpdatesAsync()

	return &messenger, nil
}

func (messenger *TelegramMessenger) consumeUpdatesAsync() {
	// Spawn a go func in the background to wait for messsages and call the message handler
	go func() {
		u := tgbotapi.NewUpdate(0)
		u.Timeout = UpdatesTimeout

		updates := messenger.bot.GetUpdatesChan(u)

		for update := range updates {
			message := update.Message
			if message != nil { // If we got a message
				if messenger.messageHandler != nil {
					messenger.messageHandler(
						fmt.Sprintf("%d", message.From.ID),
						fmt.Sprintf("%d", message.Chat.ID),
						message.Chat.Type == "group",
						message.Text,
					)
				}
			}
		}
	}()
}

func (messenger *TelegramMessenger) init() error {
	var err error

	token := os.Getenv(BotAPITokenEnvName)

	if token == "" {
		return ErrNoTokenSpecified
	}

	messenger.bot, err = tgbotapi.NewBotAPI(token)
	if err != nil {
		return errors.Wrap(err, "creating new Telegram Bot")
	}

	return nil
}

func (messenger *TelegramMessenger) Disconnect() {
	// Nothing to do
}

func (messenger *TelegramMessenger) SendMessage(recipient string, message string) error {
	id, err := strconv.ParseInt(recipient, 10, 64)
	if err != nil {
		return errors.Wrap(err, "parsing recipient as int")
	}

	msg := tgbotapi.NewMessage(id, message)
	_, err = messenger.bot.Send(msg)
	return err
}
