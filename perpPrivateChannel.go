package huobiapi

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// still working on it

type perpPrivateChannelBranch struct {
	mode               string
	account            PerpAccountBranch
	cancel             *context.CancelFunc
	httpUpdateInterval int
	errs               chan error
	tradeSets          tradeDataMap
}

type PerpAccountBranch struct {
	sync.RWMutex
	Data interface{}
}

type perpSocketAuth struct {
	Op               string `json:"op"`
	Type             string `json:"type"`
	AccessKeyID      string `json:"AccessKeyId"`
	SignatureMethod  string `json:"SignatureMethod"`
	SignatureVersion string `json:"SignatureVersion"`
	Timestamp        string `json:"Timestamp"`
	Signature        string `json:"Signature"`
}

func (c *Client) InitPerpPrivateChannel(mode string, logger *log.Logger) {
	c.perpPrivateChannelStream(mode, logger)
}

func (c *Client) perpPrivateChannelStream(mode string, logger *log.Logger) {
	u := new(perpPrivateChannelBranch)
	ctx, cancel := context.WithCancel(context.Background())
	u.cancel = &cancel
	u.httpUpdateInterval = 60
	u.mode = mode
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
				if err := perpUserData(ctx, c.key, c.secret, logger, &userData); err == nil {
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
				if err := u.maintainPerpUserData(ctx, c, &userData); err == nil {
					return
				} else {
					logger.Warningf("Refreshing huobi spot local user data with err: %s", err.Error())
				}
			}
		}
	}()
	c.perpPrivateChannel = u
	// wait for connecting
	time.Sleep(time.Second * 5)
}

// both cross and iso mode
func (u *perpPrivateChannelBranch) getAccountSnapShot(client *Client) error {
	var cont interface{}
	switch u.mode {
	case "iso":
		res, err := client.PerpIsoAccountInfo()
		if err != nil {
			return err
		}
		cont = res
	case "cross":
		res, err := client.PerpAccountInfo()
		if err != nil {
			return err
		}
		cont = res
	}
	u.account.Lock()
	defer u.account.Unlock()
	u.account.Data = cont
	return nil
}

func (u *perpPrivateChannelBranch) maintainPerpUserData(
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
			// working
			fmt.Println(message)
		}
	}
}

func perpUserData(ctx context.Context, key, secret string, logger *log.Logger, mainCh *chan map[string]interface{}) error {
	var w wS
	var duration time.Duration = 30
	w.Logger = logger
	url := "wss://api.hbdm.com/linear-swap-notification"
	host := "api.hbdm.com"
	path := "/linear-swap-notification"
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
	if send, err := getPerpAuthSubscribeMessage(host, path, key, secret); err != nil {
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
			res, err1 := perpDecodingMap(&buf, logger)
			if err1 != nil {
				message := "Huobi User Data reconnect..."
				logger.Warningln(message, err1)
				return err1
			}
			err2 := w.HandleHuobiPerpUserData(&res, mainCh, logger)
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

func perpDecodingMap(message *[]byte, logger *log.Logger) (res map[string]interface{}, err error) {
	body, err := gzip.NewReader(bytes.NewReader(*message))
	if err != nil {
		return nil, err
	}
	enflated, err := ioutil.ReadAll(body)
	if err != nil {
		logger.Println(err)
		return nil, err
	}
	err = json.Unmarshal(enflated, &res)
	if err != nil {
		logger.Println(err)
		return nil, err
	}
	return res, nil
}

func (u *perpPrivateChannelBranch) insertErr(input error) {
	if len(u.errs) == cap(u.errs) {
		<-u.errs
	}
	u.errs <- input
}

func (w *wS) subscribePerpAccountUpdateEvent(mode string) error {
	if send, err := getPerpAccountUpdateSubMessage(mode); err != nil {
		return err
	} else {
		if err := w.Conn.WriteMessage(websocket.TextMessage, send); err != nil {
			return err
		}
		w.subscribeCheck.request++
	}
	return nil
}

func getPerpAccountUpdateSubMessage(mode string) ([]byte, error) {
	var topic string
	switch mode {
	case "iso":
		topic = "accounts.$contract_code"
	case "cross":
		topic = "accounts_cross.$margin_account"
	}
	raw := authSubscribeMessage{
		Op:    "sub",
		Cid:   "5243",
		Topic: topic,
	}
	message, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	return message, nil
}

func getPerpAuthSubscribeMessage(host, path, key, secret string) ([]byte, error) {
	var buffer bytes.Buffer
	st := time.Now().UTC().Format("2006-01-02T15:04:05")
	q := url.Values{}
	q.Add("AccessKeyId", key)
	q.Add("SignatureMethod", "HmacSHA256")
	q.Add("SignatureVersion", "2")
	q.Add("Timestamp", st)
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
	auth := perpSocketAuth{
		Op:               "auth",
		Type:             "api",
		AccessKeyID:      key,
		SignatureMethod:  "HmacSHA256",
		SignatureVersion: "2",
		Timestamp:        st,
		Signature:        signature,
	}
	message, err := json.Marshal(auth)
	if err != nil {
		return nil, err
	}
	return message, nil
}

func (w *wS) HandleHuobiPerpUserData(res *map[string]interface{}, mainCh *chan map[string]interface{}, logger *log.Logger) error {
	// op, ok := (*res)["op"].(string)
	// if ok {
	// 	switch op {
	// 	case "auth":
	// 		if errCode, ok := (*res)["err-code"].(int); !ok {
	// 			return

	// 		}
	// 	}
	// }

	// if ok {
	// 	switch action {
	// 	case "ping":
	// 		data := (*res)["data"].(map[string]interface{})
	// 		ts := data["ts"].(float64)
	// 		mm := authPing{
	// 			Action: "pong",
	// 		}
	// 		mm.Data.Ts = ts
	// 		message, err := json.Marshal(mm)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		if err := w.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
	// 			return err
	// 		}
	// 	case "req":
	// 		code, okCode := (*res)["code"].(float64)
	// 		if !okCode {
	// 			return errors.New("fail to req auth on spot user data")
	// 		}
	// 		if code != 200 {
	// 			return errors.New("fail to req auth on spot user data")
	// 		}
	// 		logger.Infof("Huobi user data auth success")
	// 		// sub account update
	// 		if err := w.subscribeAccountUpdateEvent(); err != nil {
	// 			return err
	// 		}
	// 		if err := w.batchSubscribeTradeEvent(w.symbols); err != nil {
	// 			return err
	// 		}
	// 	case "sub":
	// 		code, okCode := (*res)["code"].(float64)
	// 		if !okCode {
	// 			return errors.New("fail to sub account update on spot user data")
	// 		}
	// 		if code != 200 {
	// 			return errors.New("fail to sub account update on spot user data")
	// 		}
	// 		if ch, ok := (*res)["ch"].(string); ok {
	// 			logger.Infof("Huobi private channel subscribed to %s\n", ch)
	// 			w.subscribeCheck.receive++
	// 		}
	// 	case "push":
	// 		_, okCh := (*res)["ch"].(string)
	// 		if !okCh {
	// 			return errors.New("fail to update push data on spot user data")
	// 		}
	// 		*mainCh <- (*res)
	// 	default:
	// 		// debugging
	// 		logger.Warningln(res)
	// 	}
	// }
	return nil
}
