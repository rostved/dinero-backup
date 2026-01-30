package dinero

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	AuthURL = "https://authz.dinero.dk/dineroapi/oauth/token"
	BaseURL = "https://api.dinero.dk"
)

type Client struct {
	ClientID     string
	ClientSecret string
	APIKey       string
	OrgID        string
	HTTPClient   *http.Client
	Token        string
    Debug        bool
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

func NewClient(clientID, clientSecret, apiKey, orgID string) *Client {
	return &Client{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		APIKey:       apiKey,
		OrgID:        orgID,
		HTTPClient:   &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) SetDebug(debug bool) {
    c.Debug = debug
}

func (c *Client) Authenticate() error {
    if c.Debug {
        log.Println("Authenticating...")
    }
	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("scope", "read")
	data.Set("username", c.APIKey)
	data.Set("password", c.APIKey)

	req, err := http.NewRequest("POST", AuthURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", c.ClientID, c.ClientSecret)))
	req.Header.Add("Authorization", "Basic "+auth)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed: %s", string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return err
	}

	c.Token = tokenResp.AccessToken
    if c.Debug {
        log.Println("Authenticated successfully.")
    }
	return nil
}

func (c *Client) doRequest(method, endpoint string, params url.Values, stream bool) (*http.Response, error) {
	if c.Token == "" {
		if err := c.Authenticate(); err != nil {
			return nil, err
		}
	}

	fullURL := fmt.Sprintf("%s%s", BaseURL, strings.Replace(endpoint, "{organizationId}", c.OrgID, 1))
    
    if c.Debug {
        log.Printf("Request: %s %s\n", method, fullURL)
        if params != nil {
             log.Printf("Params: %v\n", params)
        }
    }

	req, err := http.NewRequest(method, fullURL, nil)
	if err != nil {
		return nil, err
	}

	if params != nil {
		req.URL.RawQuery = params.Encode()
	}

	req.Header.Add("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Retry on 401
	if resp.StatusCode == http.StatusUnauthorized {
        if c.Debug {
            log.Println("401 Unauthorized, refreshing token...")
        }
		if err := c.Authenticate(); err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.Token)
		resp, err = c.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}
	}

    if resp.StatusCode >= 400 {
        body, _ := io.ReadAll(resp.Body)
        resp.Body.Close()
        return nil, fmt.Errorf("request failed with status code %d: %s", resp.StatusCode, string(body))
    }

	return resp, nil
}

func (c *Client) Get(endpoint string, params url.Values) ([]byte, error) {
	resp, err := c.doRequest("GET", endpoint, params, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (c *Client) GetStream(endpoint string) (io.ReadCloser, error) {
	resp, err := c.doRequest("GET", endpoint, nil, true)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (c *Client) GetPDF(endpoint string) (io.ReadCloser, error) {
	if c.Token == "" {
		if err := c.Authenticate(); err != nil {
			return nil, err
		}
	}

	fullURL := fmt.Sprintf("%s%s", BaseURL, strings.Replace(endpoint, "{organizationId}", c.OrgID, 1))

	if c.Debug {
		log.Printf("Request PDF: GET %s\n", fullURL)
	}

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+c.Token)
	req.Header.Add("Accept", "application/octet-stream")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		if c.Debug {
			log.Println("401 Unauthorized, refreshing token...")
		}
		if err := c.Authenticate(); err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.Token)
		resp, err = c.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("request failed with status code %d: %s", resp.StatusCode, string(body))
	}

	return resp.Body, nil
}
