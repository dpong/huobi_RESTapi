package huobiapi

import (
	"bytes"
	"context"
	"errors"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

type SpotUserDataBranch struct {
	account            AccountBranch
	accountID          int
	cancel             *context.CancelFunc
	httpUpdateInterval int
	errs               chan error
	trades             chan TradeData
}

type AccountBranch struct {
	sync.RWMutex
	Data *GetAccountDataResponse
}

type TradeData struct {
	Symbol    string
	Side      string
	Oid       string
	IsMaker   bool
	Price     decimal.Decimal
	Qty       decimal.Decimal
	Fee       decimal.Decimal
	TimeStamp time.Time
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

func (u *SpotUserDataBranch) AccountData() (*GetAccountDataResponse, error) {
	u.account.RLock()
	defer u.account.RUnlock()
	return u.account.Data, u.readerrs()
}

func (u *SpotUserDataBranch) ReadTrade() (TradeData, error) {
	if data, ok := <-u.trades; ok {
		return data, nil
	}
	return TradeData{}, errors.New("trade channel already closed.")
}

func (u *SpotUserDataBranch) SetAccountID(input int) {
	u.accountID = input
}

// default is 60 sec
func (u *SpotUserDataBranch) SetHttpUpdateInterval(input int) {
	u.httpUpdateInterval = input
}

func (c *Client) SpotLocalUserData(id int, logger *log.Logger) *SpotUserDataBranch {
	var u SpotUserDataBranch
	ctx, cancel := context.WithCancel(context.Background())
	u.cancel = &cancel
	u.httpUpdateInterval = 60
	u.accountID = id
	u.initialChannels()
	userData := make(chan map[string]interface{}, 100)
	// stream user data
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := spotUserData(ctx, c.key, c.secret, logger, &userData); err == nil {
					return
				}
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
				if err := u.maintainSpotUserData(ctx, c, &userData); err == nil {
					return
				} else {
					logger.Warningf("Refreshing huobi spot local user data with err: %s", err.Error())
				}
			}
		}
	}()
	// wait for connecting
	time.Sleep(time.Second * 5)
	return &u
}

func (u *SpotUserDataBranch) getAccountSnapShot(client *Client) error {
	u.account.Lock()
	defer u.account.Unlock()
	res, err := client.GetAccountData(u.accountID)
	if err != nil {
		return err
	}
	u.account.Data = res
	return nil
}

func (u *SpotUserDataBranch) maintainSpotUserData(
	ctx context.Context,
	client *Client,
	userData *chan map[string]interface{},
) error {
	// get the first snapshot to initial data struct
	if err := u.getAccountSnapShot(client); err != nil {
		return err
	}
	// update snapshot with steady interval
	go func() {
		snap := time.NewTicker(time.Second * time.Duration(u.httpUpdateInterval))
		defer snap.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-snap.C:
				if err := u.getAccountSnapShot(client); err != nil {
					u.insertErr(err)
				}
			default:
				time.Sleep(time.Second)
			}
		}
	}()
	for {
		select {
		case <-ctx.Done():
			close(u.errs)
			close(u.trades)
			return nil
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
	u.account.Lock()
	defer u.account.Unlock()
	for idx, data := range u.account.Data.Data.List {
		if currency == data.Currency && accountType == data.Type {
			u.account.Data.Data.List[idx].Balance = balance
			break
		}
	}
}

func spotUserData(ctx context.Context, key, secret string, logger *log.Logger, mainCh *chan map[string]interface{}) error {
	var w huobiWebsocket
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
	defer w.Conn.Close()
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
	read := time.NewTicker(time.Millisecond * 100)
	defer read.Stop()
	for {
		select {
		case <-ctx.Done():
			w.outHuobiErr()
			message := "Huobi User Data closing..."
			logger.Warningln(message)
			return errors.New(message)
		case <-read.C:
			if w.Conn == nil {
				w.outHuobiErr()
				message := "Huobi User Data reconnect..."
				logger.Warningln(message)
				return errors.New(message)
			}
			_, buf, err := w.Conn.ReadMessage()
			if err != nil {
				w.outHuobiErr()
				message := "Huobi User Data reconnect..."
				logger.Warningln(message)
				return errors.New(message)
			}
			res, err1 := DecodingMap(buf, logger)
			if err1 != nil {
				w.outHuobiErr()
				message := "Huobi User Data reconnect..."
				logger.Warningln(message, err1)
				return err1
			}
			err2 := w.HandleHuobiSpotUserData(&res, mainCh, logger)
			if err2 != nil {
				w.outHuobiErr()
				message := "Huobi User Data reconnect..."
				logger.Warningln(message, err2)
				return err2
			}
			if err := w.Conn.SetReadDeadline(time.Now().Add(time.Second * duration)); err != nil {
				return err
			}
		default:
			time.Sleep(time.Millisecond * 50)
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

func (w *huobiWebsocket) HandleHuobiSpotUserData(res *map[string]interface{}, mainCh *chan map[string]interface{}, logger *log.Logger) error {
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

func (u *SpotUserDataBranch) initialChannels() {
	// 5 err is allowed
	u.errs = make(chan error, 5)
	u.trades = make(chan TradeData, 100)
}

func (u *SpotUserDataBranch) insertErr(input error) {
	if len(u.errs) == cap(u.errs) {
		<-u.errs
	}
	u.errs <- input
}

func (u *SpotUserDataBranch) insertTrade(input *TradeData) {
	if len(u.trades) == cap(u.trades) {
		<-u.trades
	}
	u.trades <- *input
}

func (u *SpotUserDataBranch) readerrs() error {
	var buffer bytes.Buffer
	for {
		select {
		case err, ok := <-u.errs:
			if ok {
				buffer.WriteString(err.Error())
				buffer.WriteString(", ")
			} else {
				buffer.WriteString("errs chan already closed, ")
			}
		default:
			if buffer.Cap() == 0 {
				return nil
			}
			return errors.New(buffer.String())
		}
	}
}