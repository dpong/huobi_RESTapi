package huobiapi

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

type OrderBookBranch struct {
	Bids          BookBranch
	Asks          BookBranch
	LastUpdatedId decimal.Decimal
	SnapShoted    bool
	Cancel        *context.CancelFunc
}

type BookBranch struct {
	mux  sync.RWMutex
	Book [][]string
}

func (o *OrderBookBranch) UpdateNewComing(message *map[string]interface{}) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		// bid
		bids, ok := (*message)["bids"].([]interface{})
		if !ok {
			return
		}
		for _, bid := range bids {
			price := decimal.NewFromFloat(bid.([]interface{})[0].(float64))
			qty := decimal.NewFromFloat(bid.([]interface{})[1].(float64))
			o.DealWithBidPriceLevel(price, qty)
		}
	}()
	go func() {
		defer wg.Done()
		// ask
		asks, ok := (*message)["asks"].([]interface{})
		if !ok {
			return
		}
		for _, ask := range asks {
			price := decimal.NewFromFloat(ask.([]interface{})[0].(float64))
			qty := decimal.NewFromFloat(ask.([]interface{})[1].(float64))
			o.DealWithAskPriceLevel(price, qty)
		}
	}()
	wg.Wait()
}

func (o *OrderBookBranch) DealWithBidPriceLevel(price, qty decimal.Decimal) {
	o.Bids.mux.Lock()
	defer o.Bids.mux.Unlock()
	l := len(o.Bids.Book)
	for level, item := range o.Bids.Book {
		bookPrice, _ := decimal.NewFromString(item[0])
		switch {
		case price.GreaterThan(bookPrice):
			// insert level
			if qty.IsZero() {
				// ignore
				return
			}
			o.Bids.Book = append(o.Bids.Book, []string{})
			copy(o.Bids.Book[level+1:], o.Bids.Book[level:])
			o.Bids.Book[level] = []string{price.String(), qty.String()}
			return
		case price.LessThan(bookPrice):
			if level == l-1 {
				// insert last level
				if qty.IsZero() {
					// ignore
					return
				}
				o.Bids.Book = append(o.Bids.Book, []string{price.String(), qty.String()})
				return
			}
			continue
		case price.Equal(bookPrice):
			if qty.IsZero() {
				// delete level
				o.Bids.Book = append(o.Bids.Book[:level], o.Bids.Book[level+1:]...)
				return
			}
			o.Bids.Book[level][1] = qty.String()
			return
		}
	}
}

func (o *OrderBookBranch) DealWithAskPriceLevel(price, qty decimal.Decimal) {
	o.Asks.mux.Lock()
	defer o.Asks.mux.Unlock()
	l := len(o.Asks.Book)
	for level, item := range o.Asks.Book {
		bookPrice, _ := decimal.NewFromString(item[0])
		switch {
		case price.LessThan(bookPrice):
			// insert level
			if qty.IsZero() {
				// ignore
				return
			}
			o.Asks.Book = append(o.Asks.Book, []string{})
			copy(o.Asks.Book[level+1:], o.Asks.Book[level:])
			o.Asks.Book[level] = []string{price.String(), qty.String()}
			return
		case price.GreaterThan(bookPrice):
			if level == l-1 {
				// insert last level
				if qty.IsZero() {
					// ignore
					return
				}
				o.Asks.Book = append(o.Asks.Book, []string{price.String(), qty.String()})
				return
			}
			continue
		case price.Equal(bookPrice):
			if qty.IsZero() {
				// delete level
				o.Asks.Book = append(o.Asks.Book[:level], o.Asks.Book[level+1:]...)
				return
			}
			o.Asks.Book[level][1] = qty.String()
			return
		}
	}
}

func (o *OrderBookBranch) Close() {
	(*o.Cancel)()
	o.SnapShoted = true
	o.Bids.mux.Lock()
	o.Bids.Book = [][]string{}
	o.Bids.mux.Unlock()
	o.Asks.mux.Lock()
	o.Asks.Book = [][]string{}
	o.Asks.mux.Unlock()
}

// return bids, ready or not
func (o *OrderBookBranch) GetBids() ([][]string, bool) {
	o.Bids.mux.RLock()
	defer o.Bids.mux.RUnlock()
	if len(o.Bids.Book) == 0 || !o.SnapShoted {
		return [][]string{}, false
	}
	book := o.Bids.Book
	return book, true
}

// return asks, ready or not
func (o *OrderBookBranch) GetAsks() ([][]string, bool) {
	o.Asks.mux.RLock()
	defer o.Asks.mux.RUnlock()
	if len(o.Asks.Book) == 0 || !o.SnapShoted {
		return [][]string{}, false
	}
	book := o.Asks.Book
	return book, true
}

func SpotLocalOrderBook(symbol string, logger *log.Logger) *OrderBookBranch {
	var o OrderBookBranch
	ctx, cancel := context.WithCancel(context.Background())
	o.Cancel = &cancel
	bookticker := make(chan map[string]interface{}, 50)
	errCh := make(chan error, 5)
	symbol = strings.ToLower(symbol)
	go func(logger *log.Logger, bookticker *chan map[string]interface{}) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := HuobiOrderBookSocket(ctx, "spot", symbol, "orderbook", logger, bookticker); err == nil {
					return
				}
				errCh <- errors.New("Reconnect websocket")
				time.Sleep(time.Second)
			}
		}
	}(logger, &bookticker)
	go func() {
		for {
			select {
			case <-ctx.Done():
			default:
				o.MaintainOrderBook(ctx, symbol, logger, &bookticker, &errCh)
				time.Sleep(time.Second)
			}
		}
	}()
	return &o
}

func (o *OrderBookBranch) MaintainOrderBook(ctx context.Context, symbol string, logger *log.Logger, bookticker *chan map[string]interface{}, errCh *chan error) {
	var storage []map[string]interface{}

	for {
		select {
		case <-ctx.Done():
			return
		case <-(*errCh):
			return
		default:
			message := <-(*bookticker)

			if len(message) != 0 {
				// for initial orderbook
				_, okReq := message["rep"].(string)
				if okReq {
					o.InitialOrderBook(&message)
				} else {
					storage = append(storage, message)
				}

				if !o.SnapShoted {
					continue
				}
				if len(storage) > 1 {
					for _, data := range storage {
						headID := decimal.NewFromFloat(data["prevSeqNum"].(float64))
						tailID := decimal.NewFromFloat(data["seqNum"].(float64))
						snapID := o.LastUpdatedId.Add(decimal.NewFromInt(1))
						//U <= lastUpdateId+1 AND u >= lastUpdateId+1.
						if headID.LessThanOrEqual(snapID) && tailID.GreaterThanOrEqual(snapID) {
							// handle incoming data
							o.UpdateNewComing(&message)
							o.LastUpdatedId = tailID
						}
						storage = storage[1:]
					}
				}

				// handle incoming data
				if okReq {
					// pass the req message
					continue
				}
				headID := decimal.NewFromFloat(message["prevSeqNum"].(float64))
				tailID := decimal.NewFromFloat(message["seqNum"].(float64))
				snapID := o.LastUpdatedId.Add(decimal.NewFromInt(1))
				if headID.LessThanOrEqual(snapID) && tailID.GreaterThanOrEqual(snapID) {
					o.UpdateNewComing(&message)
					o.LastUpdatedId = tailID
				}
			}

		}
	}
}

func (o *OrderBookBranch) InitialOrderBook(res *map[string]interface{}) {
	var wg sync.WaitGroup
	data := (*res)["data"].(map[string]interface{})
	id := decimal.NewFromFloat(data["seqNum"].(float64))
	wg.Add(2)
	go func() {
		defer wg.Done()
		// bid
		o.Bids.mux.Lock()
		defer o.Bids.mux.Unlock()
		o.Bids.Book = [][]string{}
		bids := data["bids"].([]interface{})
		for _, item := range bids {
			levelData := item.([]interface{})
			price := decimal.NewFromFloat(levelData[0].(float64))
			size := decimal.NewFromFloat(levelData[1].(float64))
			o.Bids.Book = append(o.Bids.Book, []string{price.String(), size.String()})
		}
	}()
	go func() {
		defer wg.Done()
		// ask
		o.Asks.mux.Lock()
		defer o.Asks.mux.Unlock()
		o.Asks.Book = [][]string{}
		asks := data["asks"].([]interface{})
		for _, item := range asks {
			levelData := item.([]interface{})
			price := decimal.NewFromFloat(levelData[0].(float64))
			size := decimal.NewFromFloat(levelData[1].(float64))
			o.Asks.Book = append(o.Asks.Book, []string{price.String(), size.String()})
		}
	}()
	wg.Wait()
	o.LastUpdatedId = id
	o.SnapShoted = true
}

type HuobiWebsocket struct {
	Channel       string
	OnErr         bool
	Logger        *log.Logger
	Conn          *websocket.Conn
	LastUpdatedId decimal.Decimal
}

type HuobiSubscribeMessage struct {
	Sub string `json:"sub"`
	ID  string `json:"id"`
}

type HuobiSnapShotReqMessage struct {
	Req string `json:"req"`
	ID  string `json:"id"`
}

func (w *HuobiWebsocket) OutHuobiErr() map[string]interface{} {
	w.OnErr = true
	m := make(map[string]interface{})
	return m
}

func HuobiDecodingMap(message *[]byte, logger *log.Logger) (res map[string]interface{}, err error) {
	body, err := gzip.NewReader(bytes.NewReader(*message))
	if err != nil {
		return nil, err
	}
	enflated, err := ioutil.ReadAll(body)
	if err != nil {
		(*logger).Infoln(err)
		return nil, err
	}
	err = json.Unmarshal(enflated, &res)
	if err != nil {
		(*logger).Infoln(err)
		return nil, err
	}
	return res, nil
}

func HuobiOrderBookSocket(ctx context.Context, product, symbol, channel string, logger *log.Logger, mainCh *chan map[string]interface{}) error {
	var w HuobiWebsocket
	var duration time.Duration = 30
	w.Logger = logger
	w.OnErr = false
	var url string
	switch product {
	case "swap":
		url = "wss://api.hbdm.com/linear-swap-ws"
	case "spot":
		url = "wss://api.huobi.pro/feed"
	default:
		return errors.New("not supported product, cancel socket connection")
	}
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}
	(*logger).Infoln("Connected:", url)
	w.Conn = conn
	defer conn.Close()
	send, err := GetHuobiSubscribeMessage(product, channel, symbol)
	if err != nil {
		return err
	}
	go func() {
		time.Sleep(time.Second * 3)
		if send, err := GetHuobiSnapShotReqMessage(product, channel, symbol); err == nil {
			if err := w.Conn.WriteMessage(websocket.TextMessage, send); err != nil {
				(*logger).Errorf("fail to send req message with error: %s", err.Error())
				return
			}
		}
	}()

	if err := w.Conn.WriteMessage(websocket.TextMessage, send); err != nil {
		return err
	}
	if err := w.Conn.SetReadDeadline(time.Now().Add(time.Second * duration)); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			if conn == nil {
				d := w.OutHuobiErr()
				*mainCh <- d
				message := "Huobi reconnect..."
				(*logger).Infoln(message)
				return errors.New(message)
			}
			_, buf, err := conn.ReadMessage()
			if err != nil {
				d := w.OutHuobiErr()
				*mainCh <- d
				message := "Huobi reconnect..."
				(*logger).Infoln(message)
				return errors.New(message)
			}
			res, err1 := HuobiDecodingMap(&buf, logger)
			if err1 != nil {
				d := w.OutHuobiErr()
				*mainCh <- d
				message := "Huobi reconnect..."
				(*logger).Infoln(message, err1)
				return err1
			}
			err2 := w.HandleHuobiSocketData(product, &res, mainCh)
			if err2 != nil {
				d := w.OutHuobiErr()
				*mainCh <- d
				message := "Huobi reconnect..."
				(*logger).Infoln(message, err2)
				return err2
			}
			if err := w.Conn.SetReadDeadline(time.Now().Add(time.Second * duration)); err != nil {
				return err
			}
		}
	}
}

type HuobiPing struct {
	Pong float64 `json:"pong"`
}

func (w *HuobiWebsocket) HandleHuobiSocketData(product string, res *map[string]interface{}, mainCh *chan map[string]interface{}) error {
	_, ok := (*res)["ch"].(string)
	if ok {
		data, okd := (*res)["tick"].(map[string]interface{})
		if okd {
			Id := data["seqNum"].(float64)
			//preId := data["prevSeqNum"].(float64)
			newID := decimal.NewFromFloat(Id)
			if newID.LessThan(w.LastUpdatedId) {
				m := w.OutHuobiErr()
				*mainCh <- m
				return errors.New("got error when updating lastUpdateId")
			}
			w.LastUpdatedId = newID
			*mainCh <- data
			return nil
		}
	}
	_, okReq := (*res)["rep"].(string)
	if okReq {
		*mainCh <- *res
	}

	pong, ok := (*res)["ping"].(float64)
	if ok {
		mm := HuobiPing{
			Pong: pong,
		}
		message, err := json.Marshal(mm)
		if err != nil {
			return err
		}
		if err := w.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return err
		}
	}
	return nil
}

func GetHuobiSubscribeMessage(product, channel, symbol string) ([]byte, error) {
	var buffer bytes.Buffer
	switch product {
	case "spot":
		switch channel {
		case "orderbook":
			buffer.WriteString("market.")
			buffer.WriteString(symbol)
			buffer.WriteString(".mbp.150")
		default:
			return nil, errors.New("not supported channel, cancel socket connection")
		}
	default:
		return nil, errors.New("not supported product, cancel socket connection")
	}
	sub := HuobiSubscribeMessage{
		Sub: buffer.String(),
		ID:  "1234",
	}
	message, err := json.Marshal(sub)
	if err != nil {
		return nil, err
	}
	return message, nil
}

func GetHuobiSnapShotReqMessage(product, channel, symbol string) ([]byte, error) {
	var buffer bytes.Buffer
	switch product {
	case "spot":
		switch channel {
		case "orderbook":
			buffer.WriteString("market.")
			buffer.WriteString(symbol)
			buffer.WriteString(".mbp.150")
		default:
			return nil, errors.New("not supported channel, cancel socket connection")
		}
	default:
		return nil, errors.New("not supported product, cancel socket connection")
	}
	sub := HuobiSnapShotReqMessage{
		Req: buffer.String(),
		ID:  "1234",
	}
	message, err := json.Marshal(sub)
	if err != nil {
		return nil, err
	}
	return message, nil
}
