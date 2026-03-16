/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Tokens struct {
			AccessToken  string `json:"accessToken"`
			RefreshToken string `json:"refreshToken"`
			ExpiresIn    int    `json:"expiresIn"`
		} `json:"tokens"`
		User struct {
			ID       string `json:"id"`
			Username string `json:"username"`
			Email    string `json:"email"`
			Role     string `json:"role"`
		} `json:"user"`
	} `json:"data"`
}

type Config struct {
	Server         string `yaml:"server,omitempty"`
	AccessToken    string `yaml:"access_token,omitempty"`
	RefreshToken   string `yaml:"refresh_token,omitempty"`
	TokenExpiresIn int    `yaml:"token_expires_in,omitempty"`
}

var (
	loginUsername string
	loginPassword string
)

func getConfigFilePath() string {
	if cfgFile != "" {
		return cfgFile
	}

	// Check if config exists in current directory
	if _, err := os.Stat("config.yaml"); err == nil {
		return "config.yaml"
	}

	// Otherwise use home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.yaml"
	}

	configDir := filepath.Join(home, ".myapp")
	return filepath.Join(configDir, "config.yaml")
}

func saveConfig(cfg *Config) error {
	configFile := getConfigFilePath()

	// Ensure directory exists
	configDir := filepath.Dir(configFile)
	if configDir != "" && configDir != "." {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return err
		}
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, data, 0644)
}

func loadConfig() (*Config, error) {
	configFile := getConfigFilePath()
	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "login to the nebula service",
	RunE: func(cmd *cobra.Command, args []string) error {
		server := viper.GetString("server")
		if server == "" {
			return fmt.Errorf("server address is required, please set it via --server flag or config file")
		}

		if loginUsername == "" || loginPassword == "" {
			return fmt.Errorf("username and password are required")
		}

		loginReq := LoginRequest{
			Username: loginUsername,
			Password: loginPassword,
		}

		jsonData, err := json.Marshal(loginReq)
		if err != nil {
			return fmt.Errorf("failed to marshal login request: %w", err)
		}

		loginURL := fmt.Sprintf("%s/api/auth/login", server)
		resp, err := http.Post(loginURL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("failed to send login request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("login failed with status code: %d", resp.StatusCode)
		}

		var loginResp LoginResponse
		if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
			return fmt.Errorf("failed to decode login response: %w", err)
		}

		if loginResp.Code != 0 {
			return fmt.Errorf("login failed: %s", loginResp.Message)
		}

		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		cfg.Server = server
		cfg.AccessToken = loginResp.Data.Tokens.AccessToken
		cfg.RefreshToken = loginResp.Data.Tokens.RefreshToken
		cfg.TokenExpiresIn = loginResp.Data.Tokens.ExpiresIn

		if err := saveConfig(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		viper.Set("access_token", loginResp.Data.Tokens.AccessToken)
		viper.Set("refresh_token", loginResp.Data.Tokens.RefreshToken)
		viper.Set("token_expires_in", loginResp.Data.Tokens.ExpiresIn)

		fmt.Printf("Login successful! Welcome, %s (%s)\n", loginResp.Data.User.Username, loginResp.Data.User.Role)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)

	loginCmd.Flags().StringVarP(&loginUsername, "username", "u", "", "Username for login")
	loginCmd.Flags().StringVarP(&loginPassword, "password", "p", "", "Password for login")
}
