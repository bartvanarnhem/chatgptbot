package messenger

import "github.com/pkg/errors"

type Messenger interface {
	OnMessage(func(sender string, recipient string, isGroup bool, message string))
	SendMessage(recipient string, message string) error
	Disconnect()
}

type baseMessenger struct {
	messageHandler func(sender string, recipient string, isGroup bool, message string)
}

func (messenger *baseMessenger) OnMessage(handler func(sender string, recipient string, isGroup bool, message string)) {
	messenger.messageHandler = handler
}

type MessengerClientType int64

const (
	WhatsApp MessengerClientType = iota
	Telegram
)

func CreateMessenger(clientType MessengerClientType) (Messenger, error) {
	switch clientType {
	case WhatsApp:
		m, err := NewWhatsAppMessenger()
		if err != nil {
			return nil, errors.Wrap(err, "creating WhatsApp client")
		}

		return m, nil
	case Telegram:
		m, err := NewTelegramMessenger()
		if err != nil {
			return nil, errors.Wrap(err, "creating Telegram client")
		}

		return m, nil
	default:
		return nil, errors.New("unsupported messenger client type")
	}
}
