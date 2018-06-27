package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config runtime variables
type Config struct {
	Port          int        `json:"port"`
	ContentFolder string     `json:"content_folder"`
	StaticFolder  string     `json:"static_folder"`
	MimeTypes     []string   `json:"mimetypes"` // mimetypes allowed to upload
	Auth          AuthConfig `json:"auth"`
}

// AuthConfig authentication variables
type AuthConfig struct {
	EndPoint      string `json:"endpoint"`
	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"client_secret"`
	TokenDuration int    `json:"token_duration"` // token expiration time in hours
	JwtSecret     string `json:"jwt_secret"`     // string to sign tokens
}

// New returns a new config loading variables from filename or environment
func New(filename string) (Config, error) {
	c := Config{}
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	defer file.Close()

	if !os.IsNotExist(err) {
		decoder := json.NewDecoder(file)
		err = decoder.Decode(&c)
	}

	if c.Port == 0 {
		if c.Port, err = strconv.Atoi(os.Getenv("PORT")); err != nil {
			c.Port = 8000
		}
	}
	if c.Auth.EndPoint == "" {
		c.Auth.EndPoint = os.Getenv("AUTH_ENDPOINT")
	}
	if c.Auth.ClientID == "" {
		c.Auth.ClientID = os.Getenv("AUTH_CLIENT_ID")
	}
	if c.Auth.ClientSecret == "" {
		c.Auth.ClientSecret = os.Getenv("AUTH_CLIENT_SECRET")
	}
	if c.Auth.TokenDuration == 0 {
		c.Auth.TokenDuration, err = strconv.Atoi(os.Getenv("TOKEN_DURATION"))
		if err != nil {
			c.Auth.TokenDuration = 24
		}
	}
	if c.ContentFolder == "" {
		c.ContentFolder = os.Getenv("CONTENT_FOLDER")
		if c.ContentFolder == "" {
			c.ContentFolder = "content"
		}

	}
	if c.StaticFolder == "" {
		c.StaticFolder = os.Getenv("STATIC_FOLDER")
		if c.StaticFolder == "" {
			c.StaticFolder = "public"
		}
	}

	if len(c.MimeTypes) == 0 {
		if len(strings.TrimSpace(os.Getenv("MIMETYPES"))) > 0 {
			c.MimeTypes = strings.Split(os.Getenv("MIMETYPES"), ",")
		} else {
			c.MimeTypes = []string{"application/pdf", "image/png", "image/jpeg"}
		}
	}
	return c, err
}

// AuthEnabled returns true if config has authentication variables
func (c Config) AuthEnabled() bool {
	return c.Auth.EndPoint != "" && c.Auth.ClientID != "" && c.Auth.ClientSecret != "" && c.Auth.JwtSecret != ""
}
