package messagesource

import (
	"fmt"
	"os"
	"strings"

	chatgpt "github.com/golang-infrastructure/go-ChatGPT"
	"github.com/pkg/errors"
)

const (
	JWTTokenEnvName = "CHATGPT_JWT_TOKEN"
)

// ChatGPTMessageSource is a message source backed by ChatGPT
//
// Note:
// Since the ChatGPT engine (text-davinci-002-render) is not yet available using the official OpenAI GPT-3 API (at time
// of writing) we use the go-ChatGPT library that just calls the same endpoints as otherwise would be used when
// interacting with ChatGPT at https://chat.openai.com/chat. This does mean authentication to the API is slightly
// complicated since we cannot use the OpenAI API token. We need to copy the JWT token from an active ChatGPT session
// which has the downside of expiring at some point. You can use the CHATGPT_JWT_TOKEN to set this token, or rely on the
// interactive browser session that this script will open. (For this to work, chromium-browser needs to be installed.)
type ChatGPTMessageSource struct {
	client      *chatgpt.ChatGPT
	tokenGetter *InteractiveJWTTokenGetter
}

// Ensure we implement the MessageSource interface
var _ MessageSource = (*ChatGPTMessageSource)(nil)

func NewChatGPTMessageSource() (*ChatGPTMessageSource, error) {
	source := ChatGPTMessageSource{
		tokenGetter: NewInteractiveJWTTokenGetter(),
	}

	err := source.init()
	if err != nil {
		return nil, err
	}

	return &source, nil
}

func (source *ChatGPTMessageSource) getJWTToken() (string, error) {
	// Try to get the token from the environment
	token := os.Getenv(JWTTokenEnvName)
	if token != "" && source.tokenGetter.IsJWTToken(token) {
		return token, nil
	}

	// Else get a token using an interactive browser session
	fmt.Println("CHATGPT_JWT_TOKEN is not set, attempting to get a token using an interactive browser session")
	token, err := source.tokenGetter.GetToken()
	if err != nil {
		return "", errors.Wrap(err, "getting JWT token")
	}
	fmt.Printf("Token: %s\n", token)

	return token, nil
}

func (source *ChatGPTMessageSource) init() error {
	token, err := source.getJWTToken()
	if err != nil {
		return err
	}

	source.client = chatgpt.NewChatGPT(token)
	return nil
}

func (source *ChatGPTMessageSource) GetResponse(message string) (string, error) {
	talk, err := source.client.Talk(message)
	if err != nil {
		return "", err
	}

	return strings.Join(talk.Message.Content.Parts, " "), nil
}
