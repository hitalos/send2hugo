package auth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/hitalos/send2hugo/config"
	"github.com/labstack/echo"
)

type credentials struct {
	Username string `json:"username" form:"username" query:"username"`
	Password string `json:"password" form:"password" query:"password"`
}
type authResponse struct {
	Data string `json:"access_token"`
}

// Login http handler that receives username, password and returns plain text token
func Login(auth config.AuthConfig) echo.HandlerFunc {
	return func(c echo.Context) error {
		cred := credentials{}
		if err := c.Bind(&cred); err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "error binding credencials")
		}
		qs := url.Values{}
		qs.Add("grant_type", "password")
		qs.Add("client_id", auth.ClientID)
		qs.Add("client_secret", auth.ClientSecret)
		qs.Add("username", cred.Username)
		qs.Add("password", cred.Password)

		body := strings.NewReader(qs.Encode())
		resp, err := http.Post(auth.EndPoint, "application/x-www-form-urlencoded", body)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, echo.ErrUnauthorized)
		}
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "error reading request body")
		}
		authResp := authResponse{}

		if err := json.Unmarshal(b, &authResp); err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "error decoding JSON")
		}
		if authResp.Data != "" {
			token := jwt.New(jwt.SigningMethodHS256)
			claims := token.Claims.(jwt.MapClaims)
			claims["id"] = cred.Username
			claims["exp"] = time.Now().Add(time.Hour * time.Duration(auth.TokenDuration)).Unix()
			claims["iss"] = fmt.Sprintf("%s://%s", c.Scheme(), c.Request().Host)
			authResp.Data, err = token.SignedString([]byte(auth.JwtSecret))
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "error on generate token")
			}
			return c.JSON(http.StatusOK, authResp)
		}
		return echo.NewHTTPError(http.StatusUnauthorized, "wrong or empty credentials")
	}
}
