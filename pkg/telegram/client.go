package telegram

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type UserInfo struct {
	ID        int64
	FirstName string
	LastName  string
	Username  string
	PhotoURL  string
}

type Client struct {
	secretKey []byte
}

func NewClient(botToken string) *Client {
	h := sha256.Sum256([]byte(botToken))
	return &Client{secretKey: h[:]}
}

// Verify validates Telegram's HMAC-SHA256 hash and returns user info.
// data must contain all fields sent by the Telegram Login Widget (excluding nothing).
func (c *Client) Verify(data map[string]string) (*UserInfo, error) {
	hash, ok := data["hash"]
	if !ok || hash == "" {
		return nil, fmt.Errorf("telegram: missing hash")
	}

	authDateStr, ok := data["auth_date"]
	if !ok || authDateStr == "" {
		return nil, fmt.Errorf("telegram: missing auth_date")
	}
	authDate, err := strconv.ParseInt(authDateStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("telegram: invalid auth_date")
	}
	if time.Now().Unix()-authDate > 86400 {
		return nil, fmt.Errorf("telegram: auth_date is stale")
	}

	var pairs []string
	for k, v := range data {
		if k != "hash" {
			pairs = append(pairs, k+"="+v)
		}
	}
	sort.Strings(pairs)
	checkString := strings.Join(pairs, "\n")

	mac := hmac.New(sha256.New, c.secretKey)
	mac.Write([]byte(checkString))
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(hash)) {
		return nil, fmt.Errorf("telegram: hash mismatch")
	}

	idStr, ok := data["id"]
	if !ok || idStr == "" {
		return nil, fmt.Errorf("telegram: missing id")
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("telegram: invalid id")
	}

	return &UserInfo{
		ID:        id,
		FirstName: data["first_name"],
		LastName:  data["last_name"],
		Username:  data["username"],
		PhotoURL:  data["photo_url"],
	}, nil
}
