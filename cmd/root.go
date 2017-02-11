package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"

	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"os"
)

const (
	configFileName = "auth.json"
	appKey         = "sdjksdjkfhsdjkd"
	appSecret      = "23kj423jk2k3j4j"
	dropboxScheme  = "dropbox"

	tokenPersonal = "personal"
)

type TokenMap map[string]map[string]string

var config dropbox.Config

func readTokens(filePath string) (TokenMap, error) {
	b, err := ioutil.ReadFile(filePath)

	if err != nil {
		return nil, err
	}

	var tokens TokenMap
	if json.Unmarshal(b, &tokens) != nil {
		return nil, err
	}
	return tokens, nil
}

func writeTokens(filePath string, tokens TokenMap) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(filePath), 0700)
		if err != nil {
			return
		}
	}

	b, err := json.Marshal(tokens)
	if err != nil {
		return
	}

	if err = ioutil.WriteFile(filePath, b, 0600); err != nil {
		return
	}

}

func initDbx(cmd *cobra.Command, args []string) (err error) {
	dir, err := homedir.Dir()
	domain := ""

	if err != nil {
		return
	}

	configFilePath := path.Join(dir, ".config", "boxit", configFileName)
	conf := &oauth2.Config{
		ClientID:     appKey,
		ClientSecret: appSecret,
		Endpoint:     dropbox.OAuthEndpoint(domain),
	}

	tokenMap, err := readTokens(configFilePath)

	if tokenMap == nil {
		tokenMap = make(TokenMap)
	}

	if tokenMap[domain] == nil {
		tokenMap[domain] = make(map[string]string)
	}

	tokens := tokenMap[domain]

	if err != nil || tokens[tokenPersonal] == "" {
		fmt.Printf("Go to %v\n", conf.AuthCodeURL("state"))

		var code string
		if _, err = fmt.Scan(&code); err != nil {
			return
		}

		var token *oauth2.Token
		token, tokenerr := conf.Exchange(oauth2.NoContext, code)
		if tokenerr != nil {
			return
		}
		tokens[tokenPersonal] = token.AccessToken
		writeTokens(configFilePath, tokenMap)
	}

	config = dropbox.Config{tokens[tokenPersonal], false, "", ""}

	return
}

var RootCmd = &cobra.Command{
	Use:               "boxit",
	Short:             "Dropbox uploader 1",
	Long:              "",
	SilenceUsage:      true,
	PersistentPreRunE: initDbx,
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}
