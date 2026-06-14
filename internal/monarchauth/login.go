package monarchauth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pquerna/otp/totp"

	"github.com/localitas/localitas-app-mmoney/internal/monarcherr"
	"github.com/localitas/localitas-app-mmoney/internal/monarchgql"
)

const loginEndpoint = "https://api.monarch.com/auth/login/"

type loginRequest struct {
	Username      string `json:"username"`
	Password      string `json:"password"`
	SupportsMFA   bool   `json:"supports_mfa"`
	TrustedDevice bool   `json:"trusted_device"`
	TOTP          string `json:"totp,omitempty"`
}

type loginResponse struct {
	Token string `json:"token"`
}

type Session struct {
	Email     string    `json:"email,omitempty"`
	Token     string    `json:"token,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func Authenticate(email, password, mfaCode, mfaSecret string) (*Session, error) {
	if mfaSecret != "" {
		code, err := totp.GenerateCode(mfaSecret, time.Now())
		if err != nil {
			return nil, monarcherr.New(monarcherr.InternalError, "failed to generate MFA code", monarcherr.CatInternal, false, err)
		}
		mfaCode = code
	}

	reqBody := loginRequest{
		Username:      email,
		Password:      password,
		SupportsMFA:   true,
		TrustedDevice: true,
		TOTP:          mfaCode,
	}
	body, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", loginEndpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, monarcherr.New(monarcherr.InternalError, "failed to create login request", monarcherr.CatInternal, false, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Client-Platform", "web")
	req.Header.Set("User-Agent", monarchgql.UserAgent())

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, monarcherr.New(monarcherr.NetworkUnreachable, "failed to reach Monarch API", monarcherr.CatNetwork, true, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 || resp.StatusCode == 401 {
		if mfaCode == "" && mfaSecret == "" {
			return nil, monarcherr.New(monarcherr.AuthMFARequired, "MFA code required", monarcherr.CatAuth, false, nil)
		}
		return nil, monarcherr.New(monarcherr.AuthMFAInvalid, "invalid credentials or MFA code", monarcherr.CatAuth, false, nil)
	}

	if resp.StatusCode != 200 {
		var apiErr struct {
			Detail    string `json:"detail"`
			ErrorCode string `json:"error_code"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && apiErr.Detail != "" {
			return nil, monarcherr.New(monarcherr.APIError, apiErr.Detail, monarcherr.CatAPI, false, nil)
		}
		return nil, monarcherr.New(monarcherr.APIError, fmt.Sprintf("API returned status %d", resp.StatusCode), monarcherr.CatAPI, false, nil)
	}

	var loginResp loginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return nil, monarcherr.New(monarcherr.APISchemaChanged, "failed to parse login response", monarcherr.CatAPI, false, err)
	}

	return &Session{
		Email:     email,
		Token:     loginResp.Token,
		CreatedAt: time.Now().UTC(),
	}, nil
}
