package messenger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	qrterminal "github.com/mdp/qrterminal/v3"
)

const (
	DeviceStorePath        = "db/whatsapp.db"
	DeviceNotLinkedMessage = "Device is not linked yet. Please go to WhatsApp -> Settings -> Linked Devices and scan the QR code"
)

// WhatsAppMessenger uses the whatsmeow package to create a client that allows sending and receiving of messages through
// WhatsApp. The multi-device functionality of WhatsApp is used to talk to the API.
type WhatsAppMessenger struct {
	baseMessenger
	client *whatsmeow.Client
}

// Ensure we implement the Messenger interface
var _ Messenger = (*WhatsAppMessenger)(nil)

func NewWhatsAppMessenger() (*WhatsAppMessenger, error) {
	messenger := WhatsAppMessenger{}

	err := messenger.init()
	if err != nil {
		return nil, err
	}

	return &messenger, nil
}

func (messenger *WhatsAppMessenger) init() error {
	deviceStore, err := messenger.createDeviceStore()

	if err != nil {
		return errors.Wrap(err, "initializing client")
	}

	log := waLog.Stdout("Client", "DEBUG", true)
	messenger.client = whatsmeow.NewClient(deviceStore, log)
	messenger.client.AddEventHandler(messenger.eventHandler)

	if messenger.client.Store.ID == nil {
		// If the device is not yet linked, link it and connect
		err := messenger.linkDevice()
		if err != nil {
			return errors.Wrap(err, "linking device")
		}
	} else {
		// Already logged in, just connect
		err = messenger.client.Connect()
		if err != nil {
			return errors.Wrap(err, "connecting client")
		}
	}

	return nil
}

func (messenger *WhatsAppMessenger) Disconnect() {
	messenger.client.Disconnect()
}

func (messenger *WhatsAppMessenger) createDeviceStore() (*store.Device, error) {
	// Make sure the store parent's dir exists
	err := os.MkdirAll(filepath.Dir(DeviceStorePath), os.ModePerm)

	if err != nil {
		return nil, errors.Wrap(err, "creating device store parent directory")
	}

	log := waLog.Stdout("Database", "DEBUG", true)

	container, err := sqlstore.New("sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on", DeviceStorePath), log)

	if err != nil {
		return nil, errors.Wrap(err, "creating device store")
	}

	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		return nil, errors.Wrap(err, "creating device store")
	}

	return deviceStore, nil
}

// linkDevice will allow a user to link a new device by outputting a QR code on the terminal that can be used to
// authenticate this device to WhatsApp by scanning the code on your phone
func (messenger *WhatsAppMessenger) linkDevice() error {
	ctx := context.Background()
	qrChan, err := messenger.client.GetQRChannel(ctx)
	if err != nil {
		return errors.Wrap(err, "getting QR channel")
	}

	err = messenger.client.Connect()
	if err != nil {
		return errors.Wrap(err, "connecting client")
	}

	for evt := range qrChan {
		if evt.Event == "code" {
			fmt.Println(DeviceNotLinkedMessage)

			qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
		}
	}

	return nil
}

func (messenger *WhatsAppMessenger) SendMessage(recipient string, message string) error {
	jid, err := parseJID(recipient)
	if err != nil {
		return errors.Wrap(err, "parsing JID")
	}

	ctx := context.Background()
	_, err = messenger.client.SendMessage(ctx, jid, "", &waProto.Message{
		Conversation: proto.String(message),
	})

	return err
}

func (messenger *WhatsAppMessenger) eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		if messenger.messageHandler != nil {
			message := v.Message.GetConversation()
			sender := v.Info.Sender.String()
			recipient := v.Info.Chat.String()

			messenger.messageHandler(sender, recipient, v.Info.IsGroup, message)
		}
	}
}

func parseJID(jid string) (types.JID, error) {
	if jid[0] == '+' {
		jid = jid[1:]
	}

	if !strings.ContainsRune(jid, '@') {
		return types.NewJID(jid, types.DefaultUserServer), nil
	} else {
		recipient, err := types.ParseJID(jid)
		if err != nil {
			return types.JID{}, errors.Wrap(err, "parsing JID")
		} else if recipient.User == "" {
			return types.JID{}, errors.New("invalid JID, no service specified")
		}
		return recipient, nil
	}
}
