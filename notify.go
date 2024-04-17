package main

import (
	"fmt"
	"log/slog"
	"net"
	"strconv"

	"github.com/firefart/pandocserver/internal/config"

	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/discord"
	"github.com/nikoksr/notify/service/mail"
	"github.com/nikoksr/notify/service/msteams"
	"github.com/nikoksr/notify/service/sendgrid"
	"github.com/nikoksr/notify/service/telegram"
)

func setupNotifications(configuration config.Configuration, logger *slog.Logger) (*notify.Notify, error) {
	not := notify.New()
	var services []notify.Notifier

	if configuration.Notifications.Telegram.APIToken != "" {
		logger.Info("Notifications: using telegram")
		telegramService, err := telegram.New(configuration.Notifications.Telegram.APIToken)
		if err != nil {
			return nil, fmt.Errorf("telegram setup: %w", err)
		}
		telegramService.AddReceivers(configuration.Notifications.Telegram.ChatIDs...)
		services = append(services, telegramService)
	}

	if configuration.Notifications.Discord.BotToken != "" || configuration.Notifications.Discord.OAuthToken != "" {
		logger.Info("Notifications: using discord")
		discordService := discord.New()
		if configuration.Notifications.Discord.BotToken != "" {
			if err := discordService.AuthenticateWithBotToken(configuration.Notifications.Discord.BotToken); err != nil {
				return nil, fmt.Errorf("discord bot token setup: %w", err)
			}
		} else if configuration.Notifications.Discord.OAuthToken != "" {
			if err := discordService.AuthenticateWithOAuth2Token(configuration.Notifications.Discord.OAuthToken); err != nil {
				return nil, fmt.Errorf("discord oauth token setup: %w", err)
			}
		} else {
			panic("logic error")
		}
		discordService.AddReceivers(configuration.Notifications.Discord.ChannelIDs...)
		services = append(services, discordService)
	}

	if configuration.Notifications.Email.Server != "" {
		logger.Info("Notifications: using email")
		mailHost := net.JoinHostPort(configuration.Notifications.Email.Server, strconv.Itoa(configuration.Notifications.Email.Port))
		mailService := mail.New(configuration.Notifications.Email.Sender, mailHost)
		if configuration.Notifications.Email.Username != "" && configuration.Notifications.Email.Password != "" {
			mailService.AuthenticateSMTP(
				"",
				configuration.Notifications.Email.Username,
				configuration.Notifications.Email.Password,
				configuration.Notifications.Email.Server,
			)
		}
		mailService.AddReceivers(configuration.Notifications.Email.Recipients...)
		services = append(services, mailService)
	}

	if configuration.Notifications.SendGrid.APIKey != "" {
		logger.Info("Notifications: using sendgrid")
		sendGridService := sendgrid.New(
			configuration.Notifications.SendGrid.APIKey,
			configuration.Notifications.SendGrid.SenderAddress,
			configuration.Notifications.SendGrid.SenderName,
		)
		sendGridService.AddReceivers(configuration.Notifications.SendGrid.Recipients...)
		services = append(services, sendGridService)
	}

	if configuration.Notifications.MSTeams.Webhooks != nil && len(configuration.Notifications.MSTeams.Webhooks) > 0 {
		logger.Info("Notifications: using msteams")
		msteamsService := msteams.New()
		msteamsService.AddReceivers(configuration.Notifications.MSTeams.Webhooks...)
		services = append(services, msteamsService)
	}

	not.UseServices(services...)
	return not, nil
}
