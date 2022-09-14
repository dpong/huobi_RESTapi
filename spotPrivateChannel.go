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

type spotPrivateChannelBranch struct {
	account            AccountBranch
	accountID          int
	cancel             *context.CancelFunc
	httpUpdateInterval int
	errs               chan error
	symbols            []string
	tradeSets          tradeDataMap
}

type AccountBranch struct {
	sync.RWMutex
	Data *GetAccountDataResponse
}

type UserTradeData struct {
	Symbol    string
	Side      string
	Oid       string
	OrderType string
	IsMaker   bool
	Price     decimal.Decimal
	Qty       decimal.Decimal
	Fee       decimal.Decimal
	FeeAsset  string
	TimeStamp time.Time
}

type tradeDataMap struct {
	mux sync.RWMutex
	set map[string][]UserTradeData
}

type authSubscribeMessage struct {
	Op    string `json:"op"`
	Cid   string `json:"cid, omitempty"`
	Topic string `json:"topic"`
}

type spotSocketSub struct {
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

type authPing struct {
	Action string `json:"action"`
	Data   struct {
		Ts float64 `json:"ts"`
	} `json:"data"`
}

func (c *Client) CloseSpotPrivateChannel() {
	(*c.spotPrivateChannel.cancel)()
}

// id is the account id huobi requested
// symbols are batch symbols you want to receive trade report
func (c *Client) InitSpotPrivateChannel(id int, symbols []string, logger *log.Logger) {
	c.spotPrivateChannelStream(id, symbols, logger)
}

func (c *Client) SpotAccountData() (*GetAccountDataResponse, error) {
	c.spotPrivateChannel.account.RLock()
	defer c.spotPrivateChannel.account.RUnlock()
	return c.spotPrivateChannel.account.Data, c.spotPrivateChannel.readerrs()
}

// err is no trade
// mix up with multiple symbol's trade data
func (c *Client) ReadSpotUserTrade() ([]UserTradeData, error) {
	c.spotPrivateChannel.tradeSets.mux.Lock()
	defer c.spotPrivateChannel.tradeSets.mux.Unlock()
	var result []UserTradeData
	for key, item := range c.spotPrivateChannel.tradeSets.set {
		// each symbol
		result = append(result, item...)
		// earse old data
		new := []UserTradeData{}
		c.spotPrivateChannel.tradeSets.set[key] = new
	}
	if len(result) == 0 {
		return result, errors.New("no trade data")
	}
	return result, nil
}

func (u *spotPrivateChannelBranch) SetAccountID(input int) {
	u.accountID = input
}

// default is 60 sec
func (u *spotPrivateChannelBranch) SetHttpUpdateInterval(input int) {
	u.httpUpdateInterval = input
}

// internal

func (u *spotPrivateChannelBranch) insertTrade(input *UserTradeData) {
	u.tradeSets.mux.Lock()
	defer u.tradeSets.mux.Unlock()
	if _, ok := u.tradeSets.set[input.Symbol]; !ok {
		// not in the map yet
		data := []UserTradeData{*input}
		u.tradeSets.set[input.Symbol] = data
	} else {
		// already in the map
		data := u.tradeSets.set[input.Symbol]
		data = append(data, *input)
		u.tradeSets.set[input.Symbol] = data
	}
}

func (c *Client) spotPrivateChannelStream(id int, symbols []string, logger *log.Logger) {
	u := new(spotPrivateChannelBranch)
	ctx, cancel := context.WithCancel(context.Background())
	u.cancel = &cancel
	u.httpUpdateInterval = 60
	u.accountID = id
	u.symbols = symbols
	u.tradeSets.set = make(map[string][]UserTradeData, 5)
	u.errs = make(chan error, 5)
	userData := make(chan map[string]interface{}, 100)
	// stream user data
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := spotUserData(ctx, c.key, c.secret, symbols, logger, &userData); err == nil {
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
	c.spotPrivateChannel = u
	// wait for connecting
	time.Sleep(time.Second * 5)
}

func (u *spotPrivateChannelBranch) getAccountSnapShot(client *Client) error {
	res, err := client.GetAccountData(u.accountID)
	if err != nil {
		return err
	}
	u.account.Lock()
	defer u.account.Unlock()
	u.account.Data = res
	return nil
}

func (u *spotPrivateChannelBranch) maintainSpotUserData(
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
			}
		}
	}()
	for {
		select {
		case <-ctx.Done():
			close(u.errs)
			return nil
		case message := <-(*userData):
			ch, ok := message["ch"].(string)
			if !ok {
				continue
			}
			switch {
			case strings.Contains(ch, "accounts.update"):
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
			case strings.Contains(ch, "trade.clearing"):
				if event, ok := message["eventType"].(string); ok {
					if event != "trade" {
						continue
					}
				}

				info, ok := message["data"].(map[string]interface{})
				if !ok {
					continue
				}

				data := new(UserTradeData)
				if oid, ok := info["orderId"].(float64); ok {
					data.Oid = decimal.NewFromFloat(oid).String()
				}
				if symbol, ok := info["symbol"].(string); ok {
					data.Symbol = symbol
				}
				if price, ok := info["tradePrice"].(string); ok {
					priceDec, _ := decimal.NewFromString(price)
					data.Price = priceDec
				}
				if side, ok := info["orderSide"].(string); ok {
					data.Side = side
				}
				if orderType, ok := info["orderType"].(string); ok {
					if strings.Contains(orderType, "limit") {
						data.OrderType = "limit"
					} else if strings.Contains(orderType, "market") {
						data.OrderType = "market"
					}
				}
				// if strings.EqualFold(data.Side, "buy") && data.OrderType == "market" {
				// 	// check order value
				// 	if valueStr, ok := info["orderValue"].(string); ok {
				// 		value, _ := decimal.NewFromString(valueStr)
				// 		if data.Price.IsZero() {
				// 			continue
				// 		}
				// 		data.Qty = value.Div(data.Price)
				// 	}
				// } else {
				// check order size
				if sizeStr, ok := info["tradeVolume"].(string); ok {
					size, _ := decimal.NewFromString(sizeStr)
					data.Qty = size
				}
				// }
				if agg, ok := info["aggressor"].(bool); ok {
					if !agg {
						data.IsMaker = true
					}
				}
				if feeStr, ok := info["transactFee"].(string); ok {
					fee, _ := decimal.NewFromString(feeStr)
					data.Fee = fee
				}
				if feeAsset, ok := info["feeCurrency"].(string); ok {
					data.FeeAsset = strings.ToUpper(feeAsset)
				}
				if unixTs, ok := info["tradeTime"].(float64); ok {
					data.TimeStamp = time.UnixMilli(int64(unixTs))
				}

				u.insertTrade(data)
			}

		}
	}
}

// type UserTradeData struct {
// 	TimeStamp time.Time
// }

func (u *spotPrivateChannelBranch) updateSpotAccountData(currency, accountType, balance string) {
	u.account.Lock()
	defer u.account.Unlock()
	for idx, data := range u.account.Data.Data.List {
		if currency == data.Currency && accountType == data.Type {
			u.account.Data.Data.List[idx].Balance = balance
			break
		}
	}
}

func spotUserData(ctx context.Context, key, secret string, symbols []string, logger *log.Logger, mainCh *chan map[string]interface{}) error {
	var w wS
	var duration time.Duration = 30
	w.symbols = symbols
	w.Logger = logger
	url := "wss://api.huobi.pro/ws/v2"
	host := "api.huobi.pro"
	path := "/ws/v2"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}
	logger.Println("Connected:", url)
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

	// make sure subscription all settled in 10 sec
	go func() {
		time.Sleep(time.Second * 10)
		if w.subscribeCheck.receive != w.subscribeCheck.request {
			w.Conn.SetReadDeadline(time.Now().Add(time.Millisecond * 500))
		}
	}()

	for {
		select {
		case <-ctx.Done():
			message := "Huobi User Data closing..."
			logger.Warningln(message)
			return errors.New(message)
		default:
			if w.Conn == nil {
				message := "Huobi User Data reconnect..."
				logger.Warningln(message)
				return errors.New(message)
			}
			_, buf, err := w.Conn.ReadMessage()
			if err != nil {
				message := "Huobi User Data reconnect..."
				logger.Warningln(message)
				return errors.New(message)
			}
			res, err1 := decodingMap(buf, logger)
			if err1 != nil {
				message := "Huobi User Data reconnect..."
				logger.Warningln(message, err1)
				return err1
			}
			err2 := w.HandleHuobiSpotUserData(&res, mainCh, logger)
			if err2 != nil {
				message := "Huobi User Data reconnect..."
				logger.Warningln(message, err2)
				return err2
			}
			if err := w.Conn.SetReadDeadline(time.Now().Add(time.Second * duration)); err != nil {
				return err
			}
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
	auth := spotSocketSub{
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

func (w *wS) subscribeAccountUpdateEvent() error {
	if send, err := getSpotAccountUpdateSubMessage(); err != nil {
		return err
	} else {
		if err := w.Conn.WriteMessage(websocket.TextMessage, send); err != nil {
			return err
		}
		w.subscribeCheck.request++
	}
	return nil
}

func getSpotAccountUpdateSubMessage() ([]byte, error) {
	raw := spotSocketSub{
		Action: "sub",
		Ch:     "accounts.update",
	}
	message, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	return message, nil
}

func (w *wS) batchSubscribeTradeEvent(symbols []string) error {
	for _, symbol := range symbols {
		if send, err := getSpotTradeSubMessage(symbol); err != nil {
			return err
		} else {
			if err := w.Conn.WriteMessage(websocket.TextMessage, send); err != nil {
				return err
			}
			w.subscribeCheck.request++
		}
	}
	return nil
}

func getSpotTradeSubMessage(symbol string) ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString("trade.clearing#")
	buffer.WriteString(strings.ToLower(symbol))
	buffer.WriteString("#0")
	raw := spotSocketSub{
		Action: "sub",
		Ch:     buffer.String(),
	}
	message, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	return message, nil
}

func (w *wS) HandleHuobiSpotUserData(res *map[string]interface{}, mainCh *chan map[string]interface{}, logger *log.Logger) error {
	action, ok := (*res)["action"].(string)
	if ok {
		switch action {
		case "ping":
			data := (*res)["data"].(map[string]interface{})
			ts := data["ts"].(float64)
			mm := authPing{
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
			if err := w.subscribeAccountUpdateEvent(); err != nil {
				return err
			}
			if err := w.batchSubscribeTradeEvent(w.symbols); err != nil {
				return err
			}
		case "sub":
			code, okCode := (*res)["code"].(float64)
			if !okCode {
				return errors.New("fail to sub account update on spot user data")
			}
			if code != 200 {
				return errors.New("fail to sub account update on spot user data")
			}
			if ch, ok := (*res)["ch"].(string); ok {
				logger.Infof("Huobi private channel subscribed to %s\n", ch)
				w.subscribeCheck.receive++
			}
		case "push":
			_, okCh := (*res)["ch"].(string)
			if !okCh {
				return errors.New("fail to update push data on spot user data")
			}
			*mainCh <- (*res)
		default:
			// debugging
			logger.Warningln(res)
		}
	}
	return nil
}

func decodingMap(message []byte, logger *log.Logger) (res map[string]interface{}, err error) {
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

func (u *spotPrivateChannelBranch) insertErr(input error) {
	if len(u.errs) == cap(u.errs) {
		<-u.errs
	}
	u.errs <- input
}

func (u *spotPrivateChannelBranch) readerrs() error {
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
