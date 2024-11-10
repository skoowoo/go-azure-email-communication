package azure_email_communication

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

type Payload struct {
	SenderAddress string     `json:"senderAddress"`
	Recipients    Recipients `json:"recipients"`
	Content       Content    `json:"content"`
}

type Content struct {
	Subject   string `json:"subject"`
	PlainText string `json:"plainText"`
	Html      string `json:"html"`
}

type Recipients struct {
	To  []Account `json:"to"`
	Cc  []Account `json:"cc"`
	Bcc []Account `json:"bcc"`
}

type Account struct {
	Address     string `json:"address"`
	DisplayName string `json:"displayName"`
}

type Client struct {
	client    *resty.Client
	mailFrom  string
	endpoint  string
	accessKey string
}

type Option func(*Client)

func WithMailFrom(mailFrom string) Option {
	return func(client *Client) {
		client.mailFrom = mailFrom
	}
}

func WithEndpoint(endpoint, accessKey string) Option {
	return func(client *Client) {
		client.endpoint = endpoint
		client.accessKey = accessKey
	}
}

func NewClient(options ...Option) (*Client, error) {
	c := &Client{
		client: resty.New(),
	}
	for _, option := range options {
		option(c)
	}
	if c.mailFrom == "" {
		return nil, fmt.Errorf("mailFrom is required, but not provided")
	}
	if c.endpoint == "" {
		return nil, fmt.Errorf("endpoint is required, but not provided")
	}
	if c.accessKey == "" {
		return nil, fmt.Errorf("accessKey is required, but not provided")
	}
	return c, nil
}

func (c *Client) SendMail(to, subject, content string) error {
	payload := Payload{
		SenderAddress: c.mailFrom,
		Recipients: Recipients{
			To: []Account{
				{
					Address: to,
				},
			},
		},
		Content: Content{
			Subject: subject,
			Html:    content,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	date, bodyHash, authHeader, err := GenerateAuthInfo(
		"POST",
		strings.TrimPrefix(c.endpoint, "https://"),
		"/emails:send",
		map[string][]string{"api-version": {"2023-03-31"}},
		c.accessKey,
		body)
	if err != nil {
		return err
	}

	resp, err := c.client.R().
		SetHeader("x-ms-date", date).
		SetHeader("x-ms-content-sha256", bodyHash).
		SetHeader("Authorization", authHeader).
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		Post(c.endpoint + "/emails:send?api-version=2023-03-31")
	if err != nil {
		return err
	}
	if resp.StatusCode() == 429 {
		return fmt.Errorf("rate limit exceeded")
	}
	if resp.StatusCode()/100 != 2 {
		return fmt.Errorf("failed to send email: %s %s", resp.Status(), resp.String())
	}
	return nil
}

const signed_header_prefix = "HMAC-SHA256 SignedHeaders=x-ms-date;host;x-ms-content-sha256&Signature="

func GenerateAuthInfo(
	method string,
	host string,
	path string,
	query map[string][]string,
	accessKey string,
	body []byte,
) (string, string, string, error) {
	// Get the current date in the HTTP format.
	date := time.Now().UTC().Format(http.TimeFormat)

	// Compute the content hash.
	contentHash := computeContentHash(body)

	// Compute the signature.
	builder := strings.Builder{}
	builder.WriteString(path)
	if len(query) > 0 {
		builder.WriteString("?")
	}
	for k, values := range query {
		for _, v := range values {
			builder.WriteString(k)
			builder.WriteString("=")
			builder.WriteString(v)
			builder.WriteString("&")
		}
	}
	pathAndQuery := strings.TrimRight(builder.String(), "?&")

	stringToSign := method + "\n" +
		pathAndQuery + "\n" +
		date + ";" + host + ";" + contentHash
	signature := computeSignature(stringToSign, accessKey)
	authHeader := signed_header_prefix + signature

	return date, contentHash, authHeader, nil
}

func computeContentHash(content []byte) string {
	sha256 := sha256.New()
	sha256.Write(content)
	hashedBytes := sha256.Sum(nil)
	base64EncodedBytes := base64.StdEncoding.EncodeToString(hashedBytes)
	return base64EncodedBytes
}

func computeSignature(stringToSign string, accessKey string) string {
	decodedSecret, _ := base64.StdEncoding.DecodeString(accessKey)
	hash := hmac.New(sha256.New, decodedSecret)
	hash.Write([]byte(stringToSign))
	hashedBytes := hash.Sum(nil)
	encodedSignature := base64.StdEncoding.EncodeToString(hashedBytes)
	return encodedSignature
}
