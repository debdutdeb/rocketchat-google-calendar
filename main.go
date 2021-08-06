package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

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

func getEvents(dueInMinutes time.Duration) ([]*calendar.Event, error) {

	var events []*calendar.Event
	now := time.Now()

	ctx := context.Background()
	b, err := ioutil.ReadFile("credentials.json")
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

func main() {

	webhookUrl := os.Args[1]

	events, err := getEvents(time.Minute * time.Duration(30))

	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	if len(events) > 0 {
		var data []byte
		for _, event := range events {
			data, err = event.MarshalJSON()
			if err != nil {
				log.Fatalf("failed to create json: %v", err)
			}
			fmt.Println(string(data))
			_, err = http.Post(webhookUrl, "application/json", bytes.NewBuffer(data))
			if err != nil {
				log.Fatalf("ERROR: %v", err)
			}
		}
	}
}
