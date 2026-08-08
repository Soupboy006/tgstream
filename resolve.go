package main

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"
)

// findChannelAccessHash searches your dialogs for a private channel matching channelID.
func findChannelAccessHash(ctx context.Context, api *tg.Client, channelID int64) (int64, error) {
	result, err := api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		OffsetPeer: &tg.InputPeerEmpty{},
		Limit:      200,
	})
	if err != nil {
		return 0, fmt.Errorf("get dialogs: %w", err)
	}

	var chats []tg.ChatClass
	switch d := result.(type) {
	case *tg.MessagesDialogs:
		chats = d.Chats
	case *tg.MessagesDialogsSlice:
		chats = d.Chats
	default:
		return 0, fmt.Errorf("unexpected dialogs type: %T", result)
	}

	for _, c := range chats {
		if ch, ok := c.(*tg.Channel); ok && ch.ID == channelID {
			return ch.AccessHash, nil
		}
	}

	return 0, fmt.Errorf("channel %d not found in your dialogs (are you a member?)", channelID)
}

// resolveUsername resolves a public @username to a peer (channel, chat, or user).
func resolveUsername(ctx context.Context, api *tg.Client, username string) (tg.InputPeerClass, error) {
	result, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: username,
	})
	if err != nil {
		return nil, fmt.Errorf("resolve username @%s: %w", username, err)
	}

	if len(result.Chats) > 0 {
		if ch, ok := result.Chats[0].(*tg.Channel); ok {
			return &tg.InputPeerChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash}, nil
		}
	}
	if len(result.Users) > 0 {
		if u, ok := result.Users[0].(*tg.User); ok {
			return &tg.InputPeerUser{UserID: u.ID, AccessHash: u.AccessHash}, nil
		}
	}

	return nil, fmt.Errorf("@%s did not resolve to a channel or user", username)
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

// fetchDocumentFromUser fetches a message from a user DM and returns its Document.
func fetchDocumentFromUser(ctx context.Context, api *tg.Client, userID, accessHash int64, msgID int) (*tg.Document, error) {
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
