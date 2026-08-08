package main

import (
	"context"
	"fmt"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

func ensureLoggedIn(ctx context.Context, client *telegram.Client) error {
	status, err := client.Auth().Status(ctx)
	if err != nil {
		return err
	}

	if status.Authorized {
		return nil
	}

	fmt.Println("Not logged in yet. Let's log you in.")

	flow := auth.NewFlow(termAuth{}, auth.SendCodeOptions{})
	return client.Auth().IfNecessary(ctx, flow)
}

type termAuth struct{}

func (termAuth) Phone(_ context.Context) (string, error) {
	fmt.Print("Enter phone number (with country code, e.g. +91...): ")
	var phone string
	fmt.Scanln(&phone)
	return phone, nil
}

func (termAuth) Password(_ context.Context) (string, error) {
	fmt.Print("Enter 2FA password (leave blank if none): ")
	var pass string
	fmt.Scanln(&pass)
	return pass, nil
}

func (termAuth) Code(_ context.Context, _ *tg.AuthSentCode) (string, error) {
	fmt.Print("Enter code sent to your Telegram: ")
	var code string
	fmt.Scanln(&code)
	return code, nil
}

func (termAuth) AcceptTermsOfService(_ context.Context, _ tg.HelpTermsOfService) error {
	return nil
}

func (termAuth) SignUp(_ context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, fmt.Errorf("sign up not supported")
}
