package middleware

import (
	"encoding/base64"
	"github.com/spf13/viper"
	"github.com/valyala/fasthttp"
	"strings"
)

//Basic Auth calls to check authentication state, then passes handler along.
func BasicAuth(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		// Get the Basic Authentication credentials
		user, ok := basicAuth(ctx)
		if ok {
			ctx.SetUserValue("auth", user)
		}
		h(ctx)
		return
	})
}

//basicAuth does the actual checking
func basicAuth(ctx *fasthttp.RequestCtx) (username string, ok bool) {
	// check for auth header
	auth := ctx.Request.Header.Peek("Authorization")
	if auth == nil {
		return
	}
	// check that auth is basic auth
	sauth := string(auth)
	prefix := "Basic "
	if !strings.HasPrefix(sauth, prefix) {
		return
	}
	// decode authstring
	dec, err := base64.StdEncoding.DecodeString(sauth[len(prefix):])
	if err != nil {
		return
	}
	// find where username:password splits
	sdec := string(dec)
	s := strings.IndexByte(sdec, ':')
	if s < 0 {
		return
	}

	// set user and password
	user := sdec[:s]
	pw := sdec[s+1:]

	// check if user : password is correct
	var C map[string]interface{}
	_ = viper.Unmarshal(&C)
	authUsers := viper.Sub("users")
	pss := authUsers.GetString(user)
	status := pss != "" && pss == pw

	return user, status
}

func CheckAuth(ctx *fasthttp.RequestCtx) {
	if user := ctx.UserValue("auth"); user != nil {
		ctx.Error("Valid Credentials", fasthttp.StatusAccepted)
		ctx.Logger().Printf("Valid Credentials for %s", user.(string))
	} else {
		ctx.Error("Invalid Credentials", fasthttp.StatusUnauthorized)
		ctx.Logger().Printf("Invalid Credentials for %s", user.(string))
	}
	return
}
