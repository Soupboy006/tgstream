package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gotd/contrib/middleware/floodwait"
	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
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
	fmt.Println()
	fmt.Println("Required environment:")
	fmt.Println("  TGSTREAM_API_ID     Telegram API ID from https://my.telegram.org")
	fmt.Println("  TGSTREAM_API_HASH   Telegram API hash from https://my.telegram.org")
}

func runPlay(link string) {
	parsed, err := parseLink(link)
	if err != nil {
		log.Fatal("bad link: ", err)
	}

	apiID, err := requiredEnvInt("TGSTREAM_API_ID")
	if err != nil {
		log.Fatal(err)
	}
	apiHash, err := requiredEnvStr("TGSTREAM_API_HASH")
	if err != nil {
		log.Fatal(err)
	}

	sessionStorage := &session.FileStorage{
		Path: sessionPath(),
	}

	waiter := floodwait.NewSimpleWaiter()

	client := telegram.NewClient(apiID, apiHash, telegram.Options{
		SessionStorage: sessionStorage,
		Middlewares: []telegram.Middleware{
			waiter,
		},
	})

	ctx := context.Background()

	err = client.Run(ctx, func(ctx context.Context) error {
		if err := ensureLoggedIn(ctx, client); err != nil {
			return fmt.Errorf("login: %w", err)
		}

		api := client.API()

		var peer tg.InputPeerClass
		switch parsed.Kind {
		case KindPrivateChannel:
			fmt.Println("Resolving private channel...")
			accessHash, err := findChannelAccessHash(ctx, api, parsed.ChannelID)
			if err != nil {
				return err
			}
			peer = &tg.InputPeerChannel{ChannelID: parsed.ChannelID, AccessHash: accessHash}

		case KindPublicChannel:
			fmt.Printf("Resolving @%s...\n", parsed.Username)
			var err error
			peer, err = resolveUsername(ctx, api, parsed.Username)
			if err != nil {
				return err
			}
		}

		fmt.Println("Fetching message...")
		doc, err := fetchDocument(ctx, api, peer, parsed.MsgID)
		if err != nil {
			return err
		}

		fmt.Printf("Found video: %.1f MB, mime=%s\n", float64(doc.Size)/1024/1024, doc.MimeType)

		refresh := func(ctx context.Context) (*tg.Document, error) {
			return fetchDocument(ctx, api, peer, parsed.MsgID)
		}
		playURL, err := startStreamServer(api, refresh, doc)
		if err != nil {
			return err
		}
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
