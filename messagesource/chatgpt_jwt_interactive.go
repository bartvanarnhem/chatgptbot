package messagesource

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
)

const (
	SimulateUserAgent       = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/105.0.0.0 Safari/537.36"
	AuthHeaderTokenPrefix   = "Bearer"
	LoginURL                = "https://chat.openai.com/auth/login"
	InteractiveLoginTimeout = time.Second * 30
	CacheJWTTokenPath       = "db/_cached_jwt_token"
)

var (
	ErrInteractiveGetTokenTimeout = errors.New("timeout while getting token (did you login?)")
)

type InteractiveJWTTokenGetter struct {
}

func NewInteractiveJWTTokenGetter() *InteractiveJWTTokenGetter {
	return &InteractiveJWTTokenGetter{}
}

// isJWTToken checks if the passed string is a valid JWT token. We only check if the token can be parsed, no further
// validation of keys or claims is done.
func (getter *InteractiveJWTTokenGetter) IsJWTToken(token string) bool {
	tokenParsed, _ := jwt.Parse(token, nil)
	return tokenParsed != nil
}

func (getter *InteractiveJWTTokenGetter) parseTokenFromAuthHeader(header string) (string, bool) {
	if header[0:len(AuthHeaderTokenPrefix)] == AuthHeaderTokenPrefix {
		return header[len(AuthHeaderTokenPrefix)+1:], true
	}

	return "", false
}

func (getter *InteractiveJWTTokenGetter) getTokenFromCache() (string, bool, error) {
	if _, err := os.Stat(CacheJWTTokenPath); err == nil {
		fileContent, err := os.ReadFile(CacheJWTTokenPath)
		if err != nil {
			return "", false, errors.Wrap(err, "reading token cache file for reading")
		}

		token := string(fileContent)

		// If the token is not expired yet, return it
		tokenParsed, _ := jwt.Parse(string(token), nil)
		if tokenParsed != nil {
			type VerifyClaimsInterface interface {
				VerifyExpiresAt(cmp int64, req bool) bool
			}

			if claims, ok := tokenParsed.Claims.(VerifyClaimsInterface); ok {
				nowUnix := time.Now().UTC().Unix()
				if claims.VerifyExpiresAt(nowUnix, true) {
					return token, true, nil
				}
			} else {
				return "", false, errors.New("cannot get StandardClaims from token")
			}
		}
	}

	return "", false, nil
}

func (getter *InteractiveJWTTokenGetter) cacheToken(token string) error {
	// Make sure the store parent's dir exists
	err := os.MkdirAll(filepath.Dir(CacheJWTTokenPath), os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "creating token cache parent directory")
	}

	f, _ := os.OpenFile(CacheJWTTokenPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return errors.Wrap(err, "opening token cache file")
	}
	defer f.Close()

	_, err = f.WriteString(token)
	if err != nil {
		return errors.Wrap(err, "writing cache token file")
	}

	return nil
}

// chatGPTGetJWTTokenInteractive attempts to retrieve a JWT token from an interactive ChatGPT session by firing up
// a browser using chromedm and intercepting the bearer token after login
func (getter *InteractiveJWTTokenGetter) GetToken() (string, error) {
	// If cached, return this instead
	token, ok, err := getter.getTokenFromCache()
	if ok && err == nil {
		return token, nil
	}

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
				token, ok := getter.parseTokenFromAuthHeader(fmt.Sprintf("%s", authHeader))
				if ok && getter.IsJWTToken(token) {
					gotToken <- token
				}
			}
		}
	})

	// Fire up the browser
	err = chromedp.Run(taskCtx, chromedp.Navigate(LoginURL))
	if err != nil {
		return "", err
	}

	// Wait until we either intercept a token or there is a timeout
	select {
	case token := <-gotToken:
		// Before returning, cache the token for a followup run
		err := getter.cacheToken(token)
		if err != nil {
			fmt.Printf("Warning: unable to cache token: %s\n", err)
		}

		return token, nil
	case <-time.After(InteractiveLoginTimeout):
		return "", ErrInteractiveGetTokenTimeout
	}
}
