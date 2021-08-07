/* Some of the code is copied from - https://developers.google.com/calendar/api/quickstart/go */

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

func extractNumber(str string) int {
	num, err := strconv.Atoi(strings.TrimRightFunc(str, func(r rune) bool {
		if r == 'm' || r == 's' || r == 'h' {
			return true
		}
		return false
	}))
	if err != nil {
		log.Fatal(err)
	}
	return num
}

func getTime(waitFor string) time.Duration {
	switch waitFor[len(waitFor)-1] {
	case 'm':
		return time.Duration(extractNumber(waitFor)) * time.Minute
	case 's':
		return time.Duration(extractNumber(waitFor)) * time.Second
	case 'h':
		return time.Duration(extractNumber(waitFor)) * time.Hour
	default:
		if num, err := strconv.Atoi(waitFor); err != nil {
			log.Fatal(err)
		} else {
			return time.Duration(num) * time.Second
		}
	}
	return 0
}

func getClient(config *oauth2.Config) *http.Client {
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

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

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func getEvents(credentialsFile string, dueInMinutes time.Duration) ([]*calendar.Event, error) {

	var events []*calendar.Event
	now := time.Now()

	ctx := context.Background()
	b, err := ioutil.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file: %v", err)
	}

	config, err := google.ConfigFromJSON(b, calendar.CalendarReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Calendar client: %v", err)
	}

	eventsItem, err := srv.Events.List("primary").
		ShowDeleted(false).
		ShowHiddenInvitations(false).
		SingleEvents(true).
		TimeMin(now.Format(time.RFC3339)).
		TimeMax(now.Add(dueInMinutes).Format(time.RFC3339)).
		MaxResults(10).
		OrderBy("startTime").
		TimeZone("UTC").
		Do()
	if err != nil {
		return nil, fmt.Errorf("unable to get list of events: %v", err)
	}
	{
		var startTime time.Time
		for _, event := range eventsItem.Items {
			startTime, _ = time.Parse(time.RFC3339, event.Start.DateTime)
			if (startTime.Unix() - now.Unix()) > 0 {
				events = append(events, event)
			}
		}
	}
	return events, nil
}

func do(events []*calendar.Event, webhookUrl string) {
	if len(events) > 0 {
		for _, event := range events {
			data, err := event.MarshalJSON()
			if err != nil {
				log.Fatalf("failed to create json: %v", err)
			}
			_, err = http.Post(webhookUrl, "application/json", bytes.NewBuffer(data))
			if err != nil {
				log.Fatalf("ERROR: %v", err)
			}
		}
	}
}

func main() {
	var (
		webhookUrl,
		credentialsFile,
		waitFor string
		eventInMax int
	)
	flag.StringVar(&webhookUrl, "webhook", "", "Enter the webhook url you got from Rocket.Chat.")
	flag.StringVar(&credentialsFile, "credentials", "credentials.json", "Enter path to the credentials file.")
	flag.StringVar(&waitFor, "waitfor", "5m", "Time to wait before attempting POST to Rocket.Chat webhook. 'm', s', 'h' suffixes available.")
	flag.IntVar(&eventInMax, "eventin", 30, "The upper limit of upcoming event start time (in minutes, defaults to 30). Lower bound is the time of API access.")
	flag.Parse()

	if webhookUrl == "" {
		log.Fatal("[ERROR] You must pass a webhook url. Read more: https://docs.rocket.chat/guides/rocket.chat-administrator-guides/administration/integrations/google-calendar")
	}

	ticker := time.NewTicker(getTime(waitFor))
	for {
		<-ticker.C

		events, err := getEvents(credentialsFile, time.Minute*time.Duration(eventInMax))
		if err != nil {
			log.Fatalf("ERROR: %v", err)
		}

		do(events, webhookUrl)
	}

}
