package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

var (
	BaseURL   = "https://api.todoist.com/sync/v9"
	UserAgent = "todoistbackup github.com/xinau/todoistbackup"
)

type Config struct {
	Token   string `json:"token"`
	Timeout int    `json:"timeout"`
}

func (c *Config) Validate() error {
	if len(c.Token) == 0 {
		return fmt.Errorf("token is empty")
	}
	return nil
}

type Client struct {
	client *http.Client

	baseURL   *url.URL
	userAgent string
	token     string
}

func NewClient(config *Config) (*Client, error) {
	url, err := url.Parse(BaseURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		client: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
		baseURL:   url,
		userAgent: UserAgent,
		token:     config.Token,
	}, nil
}

func (c *Client) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if err := CheckResponse(resp); err != nil {
		resp.Body.Close()
		return resp, err
	}

	return resp, nil
}

type Backup struct {
	URL     string `json:"url"`
	Version string `json:"version"`

	Metadata *Metadata
}

type Metadata struct {
	ContentDisposition string
	ContentType        string
	ETag               string
	LastModified       time.Time
	Size               int64
}

func (c *Client) DownloadBackup(ctx context.Context, backup *Backup) (io.ReadCloser, *http.Response, error) {
	resp, err := c.Get(ctx, backup.URL)
	if err != nil {
		return nil, resp, err
	}

	backup.Metadata, err = ParseMetadata(resp)
	if err != nil {
		log.Printf("warning: parsing metadata of %s: %s", backup.Version, err)
	}

	return resp.Body, resp, nil
}

func (c *Client) ListBackups(ctx context.Context) ([]*Backup, *http.Response, error) {
	resp, err := c.Get(ctx, fmt.Sprintf("%s/backups/get", c.baseURL))
	if err != nil {
		return nil, resp, err
	}
	defer resp.Body.Close()

	var val []*Backup
	return val, resp, json.NewDecoder(resp.Body).Decode(&val)
}

func CheckResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode <= 399 {
		return nil
	}
	return fmt.Errorf("unexpected http status code %s", resp.Status)
}

func ParseMetadata(resp *http.Response) (*Metadata, error) {
	data := &Metadata{
		ContentDisposition: resp.Header.Get("Content-Disposition"),
		ContentType:        resp.Header.Get("Content-Type"),
		ETag:               resp.Header.Get("ETag"),
		Size:               resp.ContentLength,
	}

	var err error

	data.LastModified, err = time.Parse(time.RFC1123, resp.Header.Get("Last-Modified"))
	if err != nil {
		return nil, fmt.Errorf("parsing header Last-Modified: %w", err)
	}

	return data, nil
}
