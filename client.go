package huobiapi

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// Please do not send more than 10 requests per second. Sending requests more frequently will result in HTTP 429 errors.
type Client struct {
	key, secret        string
	subaccount         string
	HTTPC              *http.Client
	aws                bool
	AccountID          int
	spotPrivateChannel *spotPrivateChannelBranch
	perpPrivateChannel *perpPrivateChannelBranch
}

func New(key, secret, subaccount string, isAws bool) *Client {
	hc := &http.Client{
		Timeout: 10 * time.Second,
	}
	return &Client{
		key:        key,
		secret:     secret,
		subaccount: subaccount,
		HTTPC:      hc,
		aws:        isAws,
	}
}

func (p *Client) newRequest(product, method, spath string, body []byte, params *map[string]string, auth bool) (*http.Request, error) {
	q := url.Values{}
	if params != nil {
		for k, v := range *params {
			q.Add(k, v)
		}
	}
	host := p.HostHub(product)
	url, err := p.sign(host, method, spath, &q, auth)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, url, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func MakeHMAC(secret, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}

func (c *Client) sendRequest(product, method, spath string, body []byte, params *map[string]string, auth bool) (*http.Response, error) {
	req, err := c.newRequest(product, method, spath, body, params, auth)
	if err != nil {
		return nil, err
	}
	res, err := c.HTTPC.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		//c.Logger.Printf("status: %s", res.Status)
		buf := new(bytes.Buffer)
		buf.ReadFrom(res.Body)
		s := buf.String()
		return nil, fmt.Errorf("faild to get data. with error: %s", s)
	}
	return res, nil
}

func decode(res *http.Response, out interface{}) error {
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	err := json.Unmarshal([]byte(body), &out)
	if err == nil {
		return nil
	}
	return err
}

func responseLog(res *http.Response) string {
	b, _ := httputil.DumpResponse(res, true)
	return string(b)
}
func requestLog(req *http.Request) string {
	b, _ := httputil.DumpRequest(req, true)
	return string(b)
}

func (c *Client) sign(host, method, spath string, q *url.Values, auth bool) (string, error) {
	var buffer bytes.Buffer
	if !auth {
		buffer.WriteString("https://")
		buffer.WriteString(host)
		buffer.WriteString(spath)
		if (*q).Encode() == "" {
			return buffer.String(), nil
		}
		buffer.WriteString("?")
		buffer.WriteString(q.Encode())
		return buffer.String(), nil
	}
	if c.key != "" {
		q.Add("AccessKeyId", c.key)
	}
	q.Add("SignatureMethod", "HmacSHA256")
	q.Add("SignatureVersion", "2")
	q.Add("Timestamp", time.Now().UTC().Format("2006-01-02T15:04:05"))
	buffer.WriteString(method)
	buffer.WriteString("\n")
	buffer.WriteString(host)
	buffer.WriteString("\n")
	buffer.WriteString(spath)
	buffer.WriteString("\n")
	buffer.WriteString(q.Encode())
	signature, err := GetParamHmacSHA256Base64Sign(c.secret, buffer.String())
	if err != nil {
		return "", err
	}
	buffer.Reset()
	buffer.WriteString("https://")
	buffer.WriteString(host)
	buffer.WriteString(spath)
	buffer.WriteString("?")
	buffer.WriteString(q.Encode())
	buffer.WriteString("&Signature=")
	buffer.WriteString(url.QueryEscape(signature))
	return buffer.String(), nil
}

func GetParamHmacSHA256Base64Sign(secret, params string) (string, error) {
	mac := hmac.New(sha256.New, []byte(secret))
	_, err := mac.Write([]byte(params))
	if err != nil {
		return "", err
	}
	signByte := mac.Sum(nil)
	return base64.StdEncoding.EncodeToString(signByte), nil
}

func (c *Client) HostHub(product string) (host string) {
	switch product {
	case "spot":
		if c.aws {
			host = "api-aws.huobi.pro"
		} else {
			host = "api.huobi.pro"
		}
	case "swap":
		if c.aws {
			host = "api.hbdm.vn"
		} else {
			host = "api.hbdm.com"
		}
	}
	return host
}
