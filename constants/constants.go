package constants

import (
	"os"
	"strconv"
	"strings"
)

var ApplicationId int

var TokenId string

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func init() {
	token, err := os.ReadFile("secret/token")
	check(err)
	TokenId = strings.TrimSpace(string(token))

	application, err := os.ReadFile("secret/application")
	check(err)
	ApplicationId, err = strconv.Atoi(strings.TrimSpace(string(application)))
	check(err)
}
