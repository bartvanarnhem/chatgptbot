package messagesource

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/dgrijalva/jwt-go"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const (
	SimulateUserAgent       = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/105.0.0.0 Safari/537.36"
	AuthHeaderTokenPrefix   = "Bearer"
	LoginURL                = "https://chat.openai.com/auth/login"
	InteractiveLoginTimeout = time.Second * 30
)

var (
	ErrInteractiveGetTokenTimeout = errors.New("timeout while getting token (did you login?)")
)

// isJWTToken checks if the passed string is a valid JWT token. We only check if the token can be parsed, no further
// validation of keys or claims is done.
func isJWTToken(token string) bool {
	tokenParsed, _ := jwt.Parse(token, nil)
	return tokenParsed != nil
}

func parseTokenFromAuthHeader(header string) (string, bool) {
	if header[0:len(AuthHeaderTokenPrefix)] == AuthHeaderTokenPrefix {
		return header[len(AuthHeaderTokenPrefix)+1:], true
	}

	return "", false
}

// chatGPTGetJWTTokenInteractive attempts to retrieve a JWT token from an interactive ChatGPT session by firing up
// a browser using chromedm and intercepting the bearer token after login
func chatGPTGetJWTTokenInteractive() (string, error) {
	// Configure chromedp to not be headless, set a sane user-agent and a datadir
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("enable-automation", false),
		chromedp.UserDataDir(path.Join(os.TempDir(), "_chromedp")),
		chromedp.UserAgent(SimulateUserAgent),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(
		allocCtx,
		chromedp.WithLogf(log.Printf),
	)
	defer cancel()

	// Listen for network requests that have an authorization header. If we find a bearer token, signal this using the
	// gotToken channel
	gotToken := make(chan string, 1)
	chromedp.ListenTarget(taskCtx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *network.EventRequestWillBeSentExtraInfo:
			if authHeader, ok := ev.Headers["authorization"]; ok {
				if token, ok := parseTokenFromAuthHeader(fmt.Sprintf("%s", authHeader)); ok {
					gotToken <- token
				}
			}
		}
	})

	// Fire up the browser
	err := chromedp.Run(taskCtx, chromedp.Navigate(LoginURL))
	if err != nil {
		return "", err
	}

	// Wait until we either intercept a token or there is a timeout
	select {
	case token := <-gotToken:
		return token, nil
	case <-time.After(InteractiveLoginTimeout):
		return "", ErrInteractiveGetTokenTimeout
	}
}
