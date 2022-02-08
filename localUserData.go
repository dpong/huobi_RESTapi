package huobiapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

type SpotUserDataBranch struct {
	spotAccount        spotAccountBranch
	accountID          int
	cancel             *context.CancelFunc
	HttpUpdateInterval int
}

type spotAccountBranch struct {
	sync.RWMutex
	Data           *GetAccountDataResponse
	LastUpdated    time.Time
	spotsnapShoted bool
}

type HuobiAuthSubscribeMessage struct {
	Op    string `json:"op"`
	Cid   string `json:"cid, omitempty"`
	Topic string `json:"topic"`
}

type HuobiSpotSocketSub struct {
	Action string `json:"action"`
	Ch     string `json:"ch"`
	Params struct {
		Authtype         string `json:"authType"`
		Accesskey        string `json:"accessKey"`
		Signaturemethod  string `json:"signatureMethod"`
		Signatureversion string `json:"signatureVersion"`
		Timestamp        string `json:"timestamp"`
		Signature        string `json:"signature"`
	} `json:"params,omitempty"`
}

type HuobiAuthPing struct {
	Action string `json:"action"`
	Data   struct {
		Ts float64 `json:"ts"`
	} `json:"data"`
}

func (u *SpotUserDataBranch) Close() {
	(*u.cancel)()
}

func (u *SpotUserDataBranch) SetAccountID(input int) {
	u.accountID = input
}

// default is 60 sec
func (u *SpotUserDataBranch) SetHttpUpdateInterval(input int) {
	u.HttpUpdateInterval = input
}

func (u *SpotUserDataBranch) SpotAccount() *GetAccountDataResponse {
	u.spotAccount.RLock()
	defer u.spotAccount.RUnlock()
	return u.spotAccount.Data
}

func (c *Client) SpotUserData(id int, logger *log.Logger) *SpotUserDataBranch {
	var u SpotUserDataBranch
	ctx, cancel := context.WithCancel(context.Background())
	u.cancel = &cancel
	u.HttpUpdateInterval = 60
	u.accountID = id
	userData := make(chan map[string]interface{}, 100)
	errCh := make(chan error, 5)
	// stream user data
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := huobiSpotUserData(ctx, c.key, c.secret, logger, &userData); err == nil {
					return
				}
				errCh <- errors.New("Reconnect websocket")
				time.Sleep(time.Second)
			}
		}
	}()
	// update snapshot with steady interval
	go func() {
		snap := time.NewTicker(time.Second * time.Duration(u.HttpUpdateInterval))
		defer snap.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-snap.C:
				if err := u.getSpotAccountSnapShot(c); err != nil {
					message := fmt.Sprintf("fail to getSpotAccountSnapShot() with err: %s", err.Error())
					logger.Errorln(message)
				}

			default:
				time.Sleep(time.Second)
			}
		}
	}()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				u.maintainSpotUserData(ctx, c, &userData, &errCh)
				logger.Warningln("Refreshing huobi spot local user data.")
				time.Sleep(time.Second)
			}
		}
	}()
	// wait for connecting
	time.Sleep(time.Second * 5)
	return &u
}

func (u *SpotUserDataBranch) getSpotAccountSnapShot(client *Client) error {
	u.spotAccount.Lock()
	defer u.spotAccount.Unlock()
	u.spotAccount.spotsnapShoted = false
	res, err := client.GetAccountData(u.accountID)
	if err != nil {
		return err
	}
	u.spotAccount.Data = res
	u.spotAccount.spotsnapShoted = true
	u.spotAccount.LastUpdated = time.Now()
	return nil
}

func (u *SpotUserDataBranch) lastUpdateTime() time.Time {
	u.spotAccount.RLock()
	defer u.spotAccount.RUnlock()
	return u.spotAccount.LastUpdated
}

func (u *SpotUserDataBranch) maintainSpotUserData(
	ctx context.Context,
	client *Client,
	userData *chan map[string]interface{},
	errCh *chan error,
) {
	innerErr := make(chan error, 1)
	// get the first snapshot to initial data struct
	if err := u.getSpotAccountSnapShot(client); err != nil {
		return
	}
	// self check
	go func() {
		check := time.NewTicker(time.Second * 10)
		defer check.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-innerErr:
				return
			case <-check.C:
				last := u.lastUpdateTime()
				if time.Now().After(last.Add(time.Second * time.Duration(u.HttpUpdateInterval))) {
					*errCh <- errors.New("spot user data out of date")
					return
				}
			default:
				time.Sleep(time.Second)
			}
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-(*errCh):
			innerErr <- errors.New("restart")
			return
		default:
			message := <-(*userData)
			data, ok := message["data"].(map[string]interface{})
			if !ok {
				continue
			}
			id, ok := data["accountId"].(float64)
			if !ok {
				continue
			}
			if id != float64(u.accountID) {
				continue
			}
			accountType, ok := data["accountType"].(string)
			if !ok {
				continue
			}
			balance, ok := data["balance"].(string)
			if !ok {
				continue
			}
			currency, ok := data["currency"].(string)
			if !ok {
				continue
			}
			u.updateSpotAccountData(currency, accountType, balance)
		}
	}
}

func (u *SpotUserDataBranch) updateSpotAccountData(currency, accountType, balance string) {
	u.spotAccount.Lock()
	defer u.spotAccount.Unlock()
	for idx, data := range u.spotAccount.Data.Data.List {
		if currency == data.Currency && accountType == data.Type {
			u.spotAccount.Data.Data.List[idx].Balance = balance
			break
		}
	}
	u.spotAccount.LastUpdated = time.Now()
}

func huobiSpotUserData(ctx context.Context, key, secret string, logger *log.Logger, mainCh *chan map[string]interface{}) error {
	var w HuobiWebsocket
	var duration time.Duration = 30
	w.Logger = logger

	url := "wss://api.huobi.pro/ws/v2"
	host := "api.huobi.pro"
	path := "/ws/v2"

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}
	log.Println("Connected:", url)
	w.Conn = conn
	defer conn.Close()
	if err := w.Conn.SetReadDeadline(time.Now().Add(time.Second * duration)); err != nil {
		return err
	}

	// auth
	if send, err := getAuthSubscribeMessage(host, path, key, secret); err != nil {
		return err
	} else {
		if err := w.Conn.WriteMessage(websocket.TextMessage, send); err != nil {
			return err
		}
	}
	read := time.NewTicker(time.Millisecond * 50)
	defer read.Stop()
	for {
		select {
		case <-ctx.Done():
			w.OutHuobiErr()
			message := "Huobi closing..."
			logger.Warningln(message)
			return errors.New(message)
		case <-read.C:
			if conn == nil {
				w.OutHuobiErr()
				message := "Huobi reconnect..."
				logger.Warningln(message)
				return errors.New(message)
			}
			_, buf, err := conn.ReadMessage()
			if err != nil {
				w.OutHuobiErr()
				message := "Huobi reconnect..."
				logger.Warningln(message)
				return errors.New(message)
			}
			res, err1 := DecodingMap(buf, logger)
			if err1 != nil {
				w.OutHuobiErr()
				message := "Huobi reconnect..."
				logger.Warningln(message, err1)
				return err1
			}
			err2 := w.HandleHuobiSpotUserData(&res, mainCh, logger)
			if err2 != nil {
				w.OutHuobiErr()
				message := "Huobi reconnect..."
				logger.Warningln(message, err2)
				return err2
			}
			if err := w.Conn.SetReadDeadline(time.Now().Add(time.Second * duration)); err != nil {
				return err
			}
		default:
			time.Sleep(time.Millisecond * 10)
		}
	}
}

func getAuthSubscribeMessage(host, path, key, secret string) ([]byte, error) {
	var buffer bytes.Buffer
	st := time.Now().UTC().Format("2006-01-02T15:04:05")
	q := url.Values{}
	q.Add("accessKey", key)
	q.Add("signatureMethod", "HmacSHA256")
	q.Add("signatureVersion", "2.1")
	q.Add("timestamp", st)
	buffer.WriteString("GET\n")
	buffer.WriteString(host)
	buffer.WriteString("\n")
	buffer.WriteString(path)
	buffer.WriteString("\n")
	buffer.WriteString(q.Encode())
	signature, err := GetParamHmacSHA256Base64Sign(secret, buffer.String())
	if err != nil {
		return nil, err
	}
	auth := HuobiSpotSocketSub{
		Action: "req",
		Ch:     "auth",
	}
	auth.Params.Authtype = "api"
	auth.Params.Accesskey = key
	auth.Params.Signaturemethod = "HmacSHA256"
	auth.Params.Signatureversion = "2.1"
	auth.Params.Timestamp = st
	auth.Params.Signature = signature
	message, err := json.Marshal(auth)
	if err != nil {
		return nil, err
	}
	return message, nil
}

func getSpotAccountUpdateSubMessage() ([]byte, error) {
	raw := HuobiSpotSocketSub{
		Action: "sub",
		Ch:     "accounts.update",
	}
	message, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	return message, nil
}

func (w *HuobiWebsocket) HandleHuobiSpotUserData(res *map[string]interface{}, mainCh *chan map[string]interface{}, logger *log.Logger) error {
	action, ok := (*res)["action"].(string)
	if ok {
		switch action {
		case "ping":
			data := (*res)["data"].(map[string]interface{})
			ts := data["ts"].(float64)
			mm := HuobiAuthPing{
				Action: "pong",
			}
			mm.Data.Ts = ts
			message, err := json.Marshal(mm)
			if err != nil {
				return err
			}
			if err := w.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return err
			}
		case "req":
			code, okCode := (*res)["code"].(float64)
			if !okCode {
				return errors.New("fail to req auth on spot user data")
			}
			if code != 200 {
				return errors.New("fail to req auth on spot user data")
			}
			logger.Infof("Huobi user data auth success")
			// sub account update
			if send, err := getSpotAccountUpdateSubMessage(); err != nil {
				return err
			} else {
				if err := w.Conn.WriteMessage(websocket.TextMessage, send); err != nil {
					return err
				}
			}
		case "sub":
			code, okCode := (*res)["code"].(float64)
			if !okCode {
				return errors.New("fail to sub account update on spot user data")
			}
			if code != 200 {
				return errors.New("fail to sub account update on spot user data")
			}
			logger.Infof("Huobi user data subscribe to account updating.")
		case "push":
			ch, okCh := (*res)["ch"].(string)
			if !okCh {
				return errors.New("fail to update push data on spot user data")
			}
			if strings.Contains(ch, "accounts.update") {
				*mainCh <- (*res)
			}
		default:
			// debugging
			logger.Warningln(res)
		}
	}
	return nil
}

func DecodingMap(message []byte, logger *log.Logger) (res map[string]interface{}, err error) {
	if message == nil {
		err = errors.New("the incoming message is nil")
		logger.Println(err)
		return nil, err
	}
	err = json.Unmarshal(message, &res)
	if err != nil {
		logger.Println(err)
		return nil, err
	}
	return res, nil
}
