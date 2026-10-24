package main

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type LinkKind int

const (
	KindPrivateChannel LinkKind = iota // t.me/c/{channelID}/{msgID}
	KindPublicChannel                  // t.me/{username}/{msgID}
)

type ParsedLink struct {
	Kind      LinkKind
	ChannelID int64  // set for KindPrivateChannel
	Username  string // set for KindPublicChannel
	MsgID     int
}

func parseLink(link string) (*ParsedLink, error) {
	link = strings.TrimSpace(link)

	if strings.HasPrefix(link, "tg://") {
		return parseTGLink(link)
	}

	link = strings.TrimPrefix(link, "https://")
	link = strings.TrimPrefix(link, "http://")

	// Share links often carry ?single, ?thread=N or a #fragment.
	if i := strings.IndexAny(link, "?#"); i >= 0 {
		link = link[:i]
	}

	link = strings.TrimPrefix(link, "www.")
	link = strings.TrimPrefix(link, "t.me/")
	parts := strings.Split(strings.Trim(link, "/"), "/")

	// Private channel format: c/{channelID}/{msgID}
	if len(parts) == 3 && parts[0] == "c" {
		channelID, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid channel id: %s", parts[1])
		}
		msgID, err := strconv.Atoi(parts[2])
		if err != nil {
			return nil, fmt.Errorf("invalid message id: %s", parts[2])
		}
		return &ParsedLink{Kind: KindPrivateChannel, ChannelID: channelID, MsgID: msgID}, nil
	}

	// Public channel/group/DM format: {username}/{msgID}
	if len(parts) == 2 {
		username := strings.TrimPrefix(parts[0], "@")
		if username == "" {
			return nil, fmt.Errorf("missing username in %s", link)
		}
		msgID, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid message id: %s", parts[1])
		}
		return &ParsedLink{Kind: KindPublicChannel, Username: username, MsgID: msgID}, nil
	}

	return nil, fmt.Errorf("unrecognized link format: %s (expected t.me/c/ID/MSG or t.me/username/MSG)", link)
}

func parseTGLink(link string) (*ParsedLink, error) {
	u, err := url.Parse(link)
	if err != nil || u.Host != "resolve" {
		return nil, fmt.Errorf("unsupported tg:// link: %s (expected tg://resolve?domain=USERNAME&post=MSG)", link)
	}

	q := u.Query()
	username := strings.TrimPrefix(q.Get("domain"), "@")
	post := q.Get("post")
	if username == "" || post == "" {
		return nil, fmt.Errorf("tg://resolve needs domain and post: %s", link)
	}

	msgID, err := strconv.Atoi(post)
	if err != nil {
		return nil, fmt.Errorf("invalid message id: %s", post)
	}

	return &ParsedLink{Kind: KindPublicChannel, Username: username, MsgID: msgID}, nil
}
