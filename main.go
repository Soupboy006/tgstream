package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
)

var (
	apiID   = getEnvInt("TGSTREAM_API_ID", 37572969)
	apiHash = getEnvStr("TGSTREAM_API_HASH", "d2edebcdf7c93100669251fb88d00f69")
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "play":
		if len(os.Args) < 3 {
			fmt.Println("Usage: tgstream play <telegram-link>")
			os.Exit(1)
		}
		runPlay(os.Args[2])
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("tgstream - stream Telegram videos without downloading")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  tgstream play <telegram-link>   Stream a video from a message link")
}

func runPlay(link string) {
	parsed, err := parseLink(link)
	if err != nil {
		log.Fatal("bad link: ", err)
	}

	sessionStorage := &session.FileStorage{
		Path: sessionPath(),
	}

	client := telegram.NewClient(apiID, apiHash, telegram.Options{
		SessionStorage: sessionStorage,
	})

	ctx := context.Background()

	err = client.Run(ctx, func(ctx context.Context) error {
		if err := ensureLoggedIn(ctx, client); err != nil {
			return fmt.Errorf("login: %w", err)
		}

		api := client.API()

		var doc *tg.Document
		var channelID, accessHash int64

		switch parsed.Kind {
		case KindPrivateChannel:
			fmt.Println("Resolving private channel...")
			channelID = parsed.ChannelID
			accessHash, err = findChannelAccessHash(ctx, api, channelID)
			if err != nil {
				return err
			}
			fmt.Println("Fetching message...")
			doc, err = fetchDocumentFromChannel(ctx, api, channelID, accessHash, parsed.MsgID)
			if err != nil {
				return err
			}

		case KindPublicChannel:
			fmt.Printf("Resolving @%s...\n", parsed.Username)
			peer, err2 := resolveUsername(ctx, api, parsed.Username)
			if err2 != nil {
				return err2
			}

			fmt.Println("Fetching message...")
			switch p := peer.(type) {
			case *tg.InputPeerChannel:
				channelID = p.ChannelID
				accessHash = p.AccessHash
				doc, err = fetchDocumentFromChannel(ctx, api, channelID, accessHash, parsed.MsgID)
			case *tg.InputPeerUser:
				doc, err = fetchDocumentFromUser(ctx, api, p.UserID, p.AccessHash, parsed.MsgID)
			default:
				return fmt.Errorf("unsupported peer type")
			}
			if err != nil {
				return err
			}
		}

		fmt.Printf("Found video: %.1f MB, mime=%s\n", float64(doc.Size)/1024/1024, doc.MimeType)

		playURL := startStreamServer(ctx, api, channelID, accessHash, doc)
		fmt.Println("Streaming ready:", playURL)
		fmt.Println("Press Ctrl+C to stop.")

		select {}
	})

	if err != nil {
		log.Fatal(err)
	}
}

func sessionPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "tgstream_session.json"
	}
	dir := home + "/.tgstream"
	os.MkdirAll(dir, 0700)
	return dir + "/session.json"
}
