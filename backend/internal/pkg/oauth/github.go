package oauth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// —— 配置 ——

type GitHubConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

// —— GitHub API 返回结构 ——

// GitHubUser 是 GitHub API 返回的用户信息
type GitHubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// GitHubEmail 用于 /user/emails 接口
type GitHubEmail struct {
	Email      string `json:"email"`
	Primary    bool   `json:"primary"`
	Verified   bool   `json:"verified"`
	Visibility string `json:"visibility"`
}

// —— 客户端 ——

type Client struct {
	config GitHubConfig
}

func NewClient(config GitHubConfig) *Client {
	return &Client{config: config}
}

// GetAuthURL 生成 GitHub OAuth 授权页面的 URL
// state 参数用于防 CSRF，回调时会原样返回
func (c *Client) GetAuthURL(state string) string {
	u, err := url.Parse("https://github.com/login/oauth/authorize")
	if err != nil {
		// 硬编码的 GitHub URL，理论上不会出错；防御性处理
		return ""
	}
	q := u.Query()
	q.Set("client_id", c.config.ClientID)
	q.Set("redirect_uri", c.config.RedirectURI)
	q.Set("state", state)
	q.Set("scope", "user:email")
	u.RawQuery = q.Encode()
	return u.String()
}

// ExchangeCode 用授权码换取 GitHub 用户信息
// 1. 用 code 换取 access_token
// 2. 用 access_token 获取用户信息
func (c *Client) ExchangeCode(code string) (*GitHubUser, error) {
	accessToken, err := c.getAccessToken(code)
	if err != nil {
		return nil, fmt.Errorf("换取 access_token 失败: %w", err)
	}

	user, err := c.getUser(accessToken)
	if err != nil {
		return nil, fmt.Errorf("获取 GitHub 用户信息失败: %w", err)
	}

	// 如果用户资料里没有邮箱，拉取邮箱列表
	if user.Email == "" {
		email, err := c.getPrimaryEmail(accessToken)
		if err == nil {
			user.Email = email
		}
	}

	return user, nil
}

// getAccessToken POST https://github.com/login/oauth/access_token
// 参数放在 request body 中（OAuth 2.0 RFC 6749 §4.1.3 规范要求）
func (c *Client) getAccessToken(code string) (string, error) {
	data := url.Values{}
	data.Set("client_id", c.config.ClientID)
	data.Set("client_secret", c.config.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", c.config.RedirectURI)

	body := strings.NewReader(data.Encode())

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub 返回 HTTP %d", resp.StatusCode)
	}

	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析 GitHub 响应失败: %w", err)
	}

	if result.Error != "" {
		return "", fmt.Errorf("%s: %s", result.Error, result.ErrorDesc)
	}

	return result.AccessToken, nil
}

// getUser GET https://api.github.com/user
func (c *Client) getUser(accessToken string) (*GitHubUser, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub 返回 HTTP %d", resp.StatusCode)
	}

	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("解析 GitHub 用户信息失败: %w", err)
	}

	return &user, nil
}

// getPrimaryEmail GET https://api.github.com/user/emails
func (c *Client) getPrimaryEmail(accessToken string) (string, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub 返回 HTTP %d", resp.StatusCode)
	}

	var emails []GitHubEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", fmt.Errorf("解析 GitHub 邮箱列表失败: %w", err)
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}

	// 如果没有标记 primary 的，取第一个已验证的
	for _, e := range emails {
		if e.Verified {
			return e.Email, nil
		}
	}

	return "", fmt.Errorf("未找到已验证的邮箱")
}

// —— 工具函数 ——

// GenerateState 生成随机的 state 字符串（16 字节 hex）
func GenerateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
