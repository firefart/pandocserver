package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

type Configuration struct {
	Server         ConfigServer       `koanf:"server"`
	Notifications  ConfigNotification `koanf:"notifications"`
	Timeout        time.Duration      `koanf:"timeout"`
	Cloudflare     bool               `koanf:"cloudflare"`
	PandocPath     string             `koanf:"pandoc_path"`
	PandocDataDir  string             `koanf:"pandoc_data_dir"`
	CommandTimeout time.Duration      `koanf:"command_timeout"`
}

type ConfigServer struct {
	Listen          string        `koanf:"listen"`
	PprofListen     string        `koanf:"listen_pprof"`
	GracefulTimeout time.Duration `koanf:"graceful_timeout"`
	RootCA          string        `koanf:"root_ca"`
	CertSubject     string        `koanf:"cert_subject"`
}

type ConfigNotification struct {
	SecretKeyHeader string                     `koanf:"secret_key_header"`
	Telegram        ConfigNotificationTelegram `koanf:"telegram"`
	Discord         ConfigNotificationDiscord  `koanf:"discord"`
	Email           ConfigNotificationEmail    `koanf:"email"`
	SendGrid        ConfigNotificationSendGrid `koanf:"sendgrid"`
	MSTeams         ConfigNotificationMSTeams  `koanf:"msteams"`
}

type ConfigNotificationTelegram struct {
	APIToken string  `koanf:"api_token"`
	ChatIDs  []int64 `koanf:"chat_ids"`
}
type ConfigNotificationDiscord struct {
	BotToken   string   `koanf:"bot_token"`
	OAuthToken string   `koanf:"oauth_token"`
	ChannelIDs []string `koanf:"channel_ids"`
}

type ConfigNotificationEmail struct {
	Sender     string   `koanf:"sender"`
	Server     string   `koanf:"server"`
	Port       int      `koanf:"port"`
	Username   string   `koanf:"username"`
	Password   string   `koanf:"password"`
	Recipients []string `koanf:"recipients"`
}

type ConfigNotificationSendGrid struct {
	APIKey        string   `koanf:"api_key"`
	SenderAddress string   `koanf:"sender_address"`
	SenderName    string   `koanf:"sender_name"`
	Recipients    []string `koanf:"recipients"`
}

type ConfigNotificationMSTeams struct {
	Webhooks []string `koanf:"webhooks"`
}

var defaultConfig = Configuration{
	Server: ConfigServer{
		Listen:          "127.0.0.1:8000",
		PprofListen:     "127.0.0.1:1234",
		GracefulTimeout: 10 * time.Second,
	},
	CommandTimeout: 1 * time.Minute,
	PandocPath:     "/usr/local/bin/pandoc",
	PandocDataDir:  "/.pandoc",
	Timeout:        5 * time.Second,
	Cloudflare:     false,
}

func GetConfig(f string) (Configuration, error) {
	var k = koanf.NewWithConf(koanf.Conf{
		Delim: ".",
	})

	if err := k.Load(structs.Provider(defaultConfig, "koanf"), nil); err != nil {
		return Configuration{}, err
	}

	if err := k.Load(env.Provider("PANDOC_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, "PANDOC_")), "_", ".", -1)
	}), nil); err != nil {
		return Configuration{}, err
	}

	if f != "" {
		if err := k.Load(file.Provider(f), json.Parser()); err != nil {
			return Configuration{}, err
		}
	}

	var config Configuration
	if err := k.Unmarshal("", &config); err != nil {
		return Configuration{}, err
	}

	if config.Notifications.SecretKeyHeader == "" {
		return Configuration{}, fmt.Errorf("please supply a secret key header in the config")
	}

	return config, nil
}
