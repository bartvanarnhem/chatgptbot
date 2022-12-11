package messagesource

type MessageSource interface {
	GetResponse(message string) (string, error)
}
