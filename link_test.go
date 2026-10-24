package main

import "testing"

func TestParseLink(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ParsedLink
		wantErr bool
	}{
		{name: "public", input: "https://t.me/somechannel/42", want: ParsedLink{Kind: KindPublicChannel, Username: "somechannel", MsgID: 42}},
		{name: "public query", input: "https://t.me/somechannel/42?single", want: ParsedLink{Kind: KindPublicChannel, Username: "somechannel", MsgID: 42}},
		{name: "public fragment", input: "t.me/somechannel/42#comment", want: ParsedLink{Kind: KindPublicChannel, Username: "somechannel", MsgID: 42}},
		{name: "public at sign", input: "https://t.me/@somechannel/42", want: ParsedLink{Kind: KindPublicChannel, Username: "somechannel", MsgID: 42}},
		{name: "private", input: "https://t.me/c/123456789/42", want: ParsedLink{Kind: KindPrivateChannel, ChannelID: 123456789, MsgID: 42}},
		{name: "private query", input: "https://t.me/c/123456789/42?thread=7", want: ParsedLink{Kind: KindPrivateChannel, ChannelID: 123456789, MsgID: 42}},
		{name: "tg resolve", input: "tg://resolve?domain=somechannel&post=42", want: ParsedLink{Kind: KindPublicChannel, Username: "somechannel", MsgID: 42}},
		{name: "tg resolve at sign", input: "tg://resolve?domain=@somechannel&post=42", want: ParsedLink{Kind: KindPublicChannel, Username: "somechannel", MsgID: 42}},
		{name: "missing message", input: "https://t.me/somechannel", wantErr: true},
		{name: "bad channel id", input: "https://t.me/c/abc/42", wantErr: true},
		{name: "empty username", input: "https://t.me//42", wantErr: true},
		{name: "bad tg link", input: "tg://openmessage?chat_id=1&message_id=2", wantErr: true},
		{name: "wrong host", input: "https://example.com/somechannel/42", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLink(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseLink(%q) = %+v, want error", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseLink(%q) error: %v", tt.input, err)
			}
			if *got != tt.want {
				t.Fatalf("parseLink(%q) = %+v, want %+v", tt.input, *got, tt.want)
			}
		})
	}
}
