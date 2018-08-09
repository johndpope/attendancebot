package main

import (
	"github.com/kelseyhightower/envconfig"
	toml "github.com/sioncojp/tomlssm"
)

type Config struct {
	BotToken          string
	VerificationToken string
	BotID             string
	ChannelID         string
}

type envConfig struct {
	BotToken          string `envconfig:"BOT_TOKEN"`
	VerificationToken string `envconfig:"VERIFICATION_TOKEN"`
	BotID             string `envconfig:"BOT_ID"`
	ChannelID         string `envconfig:"CHANNEL_ID"`
}

type tomlConfig struct {
	BotToken          string `toml:"bot_token"`
	VerificationToken string `toml:"verification_token"`
	BotID             string `toml:"bot_id"`
	ChannelID         string `toml:"channel_id"`
}

func LoadConfig(path, region string) (*Config, error) {
	var config Config

	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		sugar.Errorf("Failed to process env var: %s", err)
		return nil, err
	}

	tc, err := loadToml(path, region)
	if err != nil {
		sugar.Errorf("Failed to load 'config.toml': %s", err)
		return nil, err
	}

	config.BotToken = tc.BotToken
	if env.BotToken != "" {
		config.BotToken = env.BotToken
	}
	config.VerificationToken = tc.VerificationToken
	if env.VerificationToken != "" {
		config.VerificationToken = env.VerificationToken
	}
	config.BotID = tc.BotID
	if env.BotID != "" {
		config.BotID = env.BotID
	}
	config.ChannelID = tc.ChannelID
	if env.ChannelID != "" {
		config.ChannelID = env.ChannelID
	}

	return &config, nil
}

func loadToml(path, region string) (*tomlConfig, error) {
	var config tomlConfig
	if _, err := toml.DecodeFile(path, &config, region); err != nil {
		return nil, err
	}
	return &config, nil
}
