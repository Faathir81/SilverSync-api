package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/zmb3/spotify/v2"
	"golang.org/x/oauth2/clientcredentials"
)

func main() {
	clientID := "bc985402a3d24b9d8c28b73510baa85e"
	clientSecret := "e9553943e5f44fc39cc3e5f5ee41153c"
	ctx := context.Background()

	config := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     "https://accounts.spotify.com/api/token",
	}

	token, err := config.Token(ctx)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	fmt.Println("Got token:", token.AccessToken[:10])

	httpClient := config.Client(ctx)
	client := spotify.New(httpClient)

	track, err := client.GetTrack(ctx, spotify.ID("11dFghVXANMlKmJXsNCbNl"))
	if err != nil {
		fmt.Println("Error track:", err)
		os.Exit(1)
	}

	var artists []string
	for _, a := range track.Artists {
		artists = append(artists, a.Name)
	}
	fmt.Println("Track:", track.Name, "-", strings.Join(artists, ", "))
}
