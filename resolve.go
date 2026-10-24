package main

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"
)

const dialogsPageSize = 100

// findChannelAccessHash searches your dialogs for a private channel matching channelID.
func findChannelAccessHash(ctx context.Context, api *tg.Client, channelID int64) (int64, error) {
	offsetPeer := tg.InputPeerClass(&tg.InputPeerEmpty{})
	offsetDate, offsetID := 0, 0

	for page := 0; page < 100; page++ {
		result, err := api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
			OffsetPeer: offsetPeer,
			OffsetDate: offsetDate,
			OffsetID:   offsetID,
			Limit:      dialogsPageSize,
		})
		if err != nil {
			return 0, fmt.Errorf("get dialogs: %w", err)
		}

		var dialogs []tg.DialogClass
		var chats []tg.ChatClass
		var users []tg.UserClass
		var messages []tg.MessageClass
		switch d := result.(type) {
		case *tg.MessagesDialogs:
			dialogs, chats, users, messages = d.Dialogs, d.Chats, d.Users, d.Messages
		case *tg.MessagesDialogsSlice:
			dialogs, chats, users, messages = d.Dialogs, d.Chats, d.Users, d.Messages
		default:
			return 0, fmt.Errorf("unexpected dialogs type: %T", result)
		}

		for _, c := range chats {
			if ch, ok := c.(*tg.Channel); ok && ch.ID == channelID {
				return ch.AccessHash, nil
			}
		}

		if len(dialogs) < dialogsPageSize {
			break
		}
		last, ok := dialogs[len(dialogs)-1].(*tg.Dialog)
		if !ok || last.TopMessage == offsetID {
			break
		}
		nextPeer, ok := inputPeerFromDialog(last, chats, users)
		if !ok {
			break
		}
		offsetPeer = nextPeer
		offsetID = last.TopMessage
		offsetDate = messageDate(messages, last.TopMessage)
	}

	return 0, fmt.Errorf("channel %d not found in your dialogs (are you a member?)", channelID)
}

func inputPeerFromDialog(d *tg.Dialog, chats []tg.ChatClass, users []tg.UserClass) (tg.InputPeerClass, bool) {
	switch p := d.Peer.(type) {
	case *tg.PeerChannel:
		for _, c := range chats {
			if ch, ok := c.(*tg.Channel); ok && ch.ID == p.ChannelID {
				return ch.AsInputPeer(), true
			}
		}
	case *tg.PeerChat:
		for _, c := range chats {
			if ch, ok := c.(*tg.Chat); ok && ch.ID == p.ChatID {
				return ch.AsInputPeer(), true
			}
		}
	case *tg.PeerUser:
		for _, u := range users {
			if usr, ok := u.(*tg.User); ok && usr.ID == p.UserID {
				return usr.AsInputPeer(), true
			}
		}
	}
	return nil, false
}

func messageDate(messages []tg.MessageClass, id int) int {
	for _, mc := range messages {
		if m, ok := mc.(*tg.Message); ok && m.ID == id {
			return m.Date
		}
	}
	return 0
}

// resolveUsername resolves a public @username to a peer (channel, group, or user).
func resolveUsername(ctx context.Context, api *tg.Client, username string) (tg.InputPeerClass, error) {
	result, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: username,
	})
	if err != nil {
		return nil, fmt.Errorf("resolve username @%s: %w", username, err)
	}

	if len(result.Chats) > 0 {
		switch ch := result.Chats[0].(type) {
		case *tg.Channel:
			return &tg.InputPeerChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash}, nil
		case *tg.Chat:
			return &tg.InputPeerChat{ChatID: ch.ID}, nil
		}
	}
	if len(result.Users) > 0 {
		if u, ok := result.Users[0].(*tg.User); ok {
			return &tg.InputPeerUser{UserID: u.ID, AccessHash: u.AccessHash}, nil
		}
	}

	return nil, fmt.Errorf("@%s did not resolve to a channel, group, or user", username)
}

// fetchDocument fetches a message by peer and returns its Document.
func fetchDocument(ctx context.Context, api *tg.Client, peer tg.InputPeerClass, msgID int) (*tg.Document, error) {
	if ch, ok := peer.(*tg.InputPeerChannel); ok {
		return fetchDocumentFromChannel(ctx, api, ch.ChannelID, ch.AccessHash, msgID)
	}
	return fetchDocumentFromUser(ctx, api, msgID)
}

// fetchDocumentFromChannel fetches a message from a channel (private or public) and returns its Document.
func fetchDocumentFromChannel(ctx context.Context, api *tg.Client, channelID, accessHash int64, msgID int) (*tg.Document, error) {
	result, err := api.ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{
		Channel: &tg.InputChannel{ChannelID: channelID, AccessHash: accessHash},
		ID:      []tg.InputMessageClass{&tg.InputMessageID{ID: msgID}},
	})
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}

	m, ok := result.(*tg.MessagesChannelMessages)
	if !ok || len(m.Messages) == 0 {
		return nil, fmt.Errorf("message not found")
	}

	return extractDocument(m.Messages[0])
}

// fetchDocumentFromUser fetches a message from a user DM or basic group.
func fetchDocumentFromUser(ctx context.Context, api *tg.Client, msgID int) (*tg.Document, error) {
	result, err := api.MessagesGetMessages(ctx, []tg.InputMessageClass{&tg.InputMessageID{ID: msgID}})
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}

	var messages []tg.MessageClass
	switch m := result.(type) {
	case *tg.MessagesMessages:
		messages = m.Messages
	case *tg.MessagesMessagesSlice:
		messages = m.Messages
	default:
		return nil, fmt.Errorf("unexpected messages type: %T", result)
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("message not found")
	}

	return extractDocument(messages[0])
}

func extractDocument(mc tg.MessageClass) (*tg.Document, error) {
	msg, ok := mc.(*tg.Message)
	if !ok {
		return nil, fmt.Errorf("message was deleted or inaccessible")
	}

	docMedia, ok := msg.Media.(*tg.MessageMediaDocument)
	if !ok {
		return nil, fmt.Errorf("message has no document/video")
	}

	doc, ok := docMedia.Document.(*tg.Document)
	if !ok {
		return nil, fmt.Errorf("unexpected document type")
	}

	return doc, nil
}
