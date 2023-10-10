/*
	Author:			Astennu
	Created:		08.10.23

	Updated by:		Astennu
	Last Updated:	10.10.23

	Description:
	Access gmail account and gets all mails with a specified label.
	If the body of the mail is in html style it will remove all html tags.
	If a python file is specified it will give the body of the mail as input to the python file you can receive it with this python command: input_string = sys.stdin.read()
	else it will print the body in the console
*/

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/net/html"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

type clientInfo struct {
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
	LabelName    string `json:"labelName"`
	RedirectURL  string `json:"redirectURL"`
	PythonPath   string `json:"pythonPath"`
}

func main() {

	//Read content of JSON file
	fileContent, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Fatalf("Error reading JSON file: %v", err)
	}
	var configFile clientInfo

	err = json.Unmarshal(fileContent, &configFile)
	if err != nil {
		log.Fatalf("Error unmarshalling JSON: %v", err)
	}

	// Create a new Gmail client using OAuth 2.0 authentication.
	config := &oauth2.Config{
		ClientID:     configFile.ClientID,
		ClientSecret: configFile.ClientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  configFile.RedirectURL,
		Scopes:       []string{gmail.GmailReadonlyScope},
	}

	ctx := context.Background()
	client := getClient(ctx, config)

	// Create a Gmail service using the authenticated client.
	srv, err := gmail.New(client)
	if err != nil {
		log.Fatalf("Unable to create Gmail service: %v", err)
	}

	// Get the label ID for the specified label name.
	label, err := srv.Users.Labels.List("me").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve labels: %v", err)
	}

	var labelID string
	for _, l := range label.Labels {
		if l.Name == configFile.LabelName {
			labelID = l.Id
			break
		}
	}

	if labelID == "" {
		log.Fatalf("Label '%s' not found", configFile.LabelName)
	}

	// List all messages with the specified label using pagination.
	pageToken := ""
	for {
		messages, err := srv.Users.Messages.List("me").LabelIds(labelID).PageToken(pageToken).Do()
		if err != nil {
			log.Fatalf("Unable to retrieve messages: %v", err)
		}

		// Handle the bodies
		for _, message := range messages.Messages {
			body, err := getMessageBody(srv, "me", message.Id)
			if err != nil {
				log.Printf("Unable to retrieve body of the email (ID: %s): %v", message.Id, err)
			} else {
				/*
					-------------------------------------------------------
					Enter here what it should do with the body of each mail
					-------------------------------------------------------
				*/
				if configFile.PythonPath == "" {
					fmt.Printf("\nBody of the email (ID: %s):\n%s\n", message.Id, body)
				} else {
					err := runPythonScript(body)
					if err != nil {
						fmt.Printf("Error running Python script: %v\n", err)
						continue
					}
				}
			}
		}

		// Check if there are more pages.
		if messages.NextPageToken == "" {
			break
		}

		pageToken = messages.NextPageToken
	}
}

// getMessageBody retrieves the body of a specific email and converts HTML to plain text.
func getMessageBody(srv *gmail.Service, userID, messageID string) (string, error) {
	message, err := srv.Users.Messages.Get(userID, messageID).Format("full").Do()
	if err != nil {
		return "", err
	}

	if message.Payload != nil {
		var body string

		// Check if there's a plain text part in the payload.
		for _, part := range message.Payload.Parts {
			if part.MimeType == "text/plain" {
				data, err := base64.URLEncoding.DecodeString(part.Body.Data)
				if err != nil {
					return "", err
				}
				body = string(data)
				break
			}
		}

		// If there's no plain text part, use the HTML part.
		if body == "" && message.Payload.Body != nil {
			data, err := base64.URLEncoding.DecodeString(message.Payload.Body.Data)
			if err != nil {
				return "", err
			}
			body = convertHTMLToPlainText(string(data))
		}

		return body, nil
	}

	return "", nil
}

// convertHTMLToPlainText converts HTML content to plain text.
func convertHTMLToPlainText(htmlContent string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		log.Printf("Error parsing HTML: %v", err)
		return htmlContent
	}

	var plainText string
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.TextNode {
			plainText += n.Data
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)
	return plainText
}

// getClient retrieves a token from the cache or generates a new one if necessary.
func getClient(ctx context.Context, config *oauth2.Config) *http.Client {
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(ctx, tok)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser, then type the "+
		"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.Background(), code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// tokenFromFile retrieves a Token from a given file path.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// saveToken saves the token to a file path.
func saveToken(file string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.Create(file)
	if err != nil {
		log.Fatalf("Unable to cache OAuth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func runPythonScript(input string) error {
	// Command to run the Python script
	cmd := exec.Command("python", "makecsv.py")

	// Connect the standard input of the Python script to a pipe
	cmd.Stdin = strings.NewReader(input)

	// Capture the standard output of the Python script
	_, err := cmd.Output()
	if err != nil {
		return err
	}

	return nil
}
