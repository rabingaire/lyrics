package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
)

var (
	spotifyClientID     string
	spotifyClientSecret string
	geniusClientSecret  string
)

const redirectURI = "http://localhost:8080/callback"

type authHandler struct {
	state string
	ch    chan *oauth2.Token
	auth  spotify.Authenticator
}

type response struct {
	Lyrics string `json:"lyrics"`
}

func (a *authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tok, err := a.auth.Token(a.state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}

	if st := r.FormValue("state"); st != a.state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, a.state)
	}

	fmt.Fprintf(w, "Login successfully. Please return to your terminal.")

	a.ch <- tok
}

func main() {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	tokenPath := filepath.Join(usr.HomeDir, ".lyrics")

	auth := spotify.NewAuthenticator(
		redirectURI,
		spotify.ScopeUserReadCurrentlyPlaying,
	)
	auth.SetAuthInfo(spotifyClientID, spotifyClientSecret)

	token, err := readToken(tokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			state, err := generateRandomString(32)
			if err != nil {
				panic(err)
			}

			ch := make(chan *oauth2.Token)

			http.Handle("/callback", &authHandler{state: state, ch: ch, auth: auth})
			go http.ListenAndServe(":8080", nil)

			url := auth.AuthURL(state)
			openbrowser(url)
			// fmt.Println("Please log in to Spotify by visiting the following page in your browser:", url)

			tok := <-ch

			if err := saveToken(tok, tokenPath); err != nil {
				panic(err)
			}

			// read token one more time
			token, err = readToken(tokenPath)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatal(err)
		}
	}

	client := auth.NewClient(token)
	current, err := client.PlayerCurrentlyPlaying()
	if err != nil {
		panic(err)
	}
	if current.Playing {
		getLyrics(current.Item.Name, current.Item.Artists[0].Name)
	} else {
		fmt.Println("No track playing")
	}

}

func generateRandomString(s int) (string, error) {
	b, err := generateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b), err
}

func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func saveToken(tok *oauth2.Token, tokenPath string) error {
	f, err := os.OpenFile(tokenPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	return enc.Encode(tok)
}

func readToken(tokenPath string) (*oauth2.Token, error) {
	content, err := ioutil.ReadFile(tokenPath)
	if err != nil {
		return nil, err
	}

	var tok oauth2.Token
	if err := json.Unmarshal(content, &tok); err != nil {
		return nil, err
	}

	return &tok, nil
}

func openbrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}

func getLyrics(trackName string, artistName string) {
	url := fmt.Sprintf("https://songlyricsfree.herokuapp.com/?api_key=%s&title=%s&artist=%s", geniusClientSecret, url.QueryEscape(trackName), url.QueryEscape(artistName))
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		var track response
		if err := json.Unmarshal(bodyBytes, &track); err != nil {
			panic(err)
		}
		fmt.Println("######################")
		fmt.Println(fmt.Sprintf("%s - %s", trackName, artistName))
		fmt.Println("######################")
		fmt.Println()
		fmt.Println(track.Lyrics)
	}
}
