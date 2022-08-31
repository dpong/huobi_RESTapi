package huobiapi

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

const NullPrice = "null"

type StreamTickerBranch struct {
	bid    tobBranch
	ask    tobBranch
	cancel *context.CancelFunc
	reCh   chan error

	contractSize decimal.Decimal
}

type tobBranch struct {
	mux       sync.RWMutex
	price     string
	qty       string
	timeStamp time.Time
}

// ex: symbol = BTC-USDT
func PerpStreamTicker(symbol string, logger *log.Logger) *StreamTickerBranch {
	return localStreamTicker("swap", symbol, logger)
}

// ex: symbol = btcusdt
func SpotStreamTicker(symbol string, logger *log.Logger) *StreamTickerBranch {
	return localStreamTicker("spot", symbol, logger)
}

func localStreamTicker(product, symbol string, logger *log.Logger) *StreamTickerBranch {
	var s StreamTickerBranch
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = &cancel
	ticker := make(chan map[string]interface{}, 50)
	errCh := make(chan error, 5)
	// initial data with rest api first
	s.initialWithSpotDetail(product, symbol)
	if product == "swap" {
		client := New("", "", "", false)
		res, err := client.Perps("")
		if err != nil {
			logger.Errorf("fail to get perp info, fail to init stream ticker")
			return nil
		}
		for _, item := range res.Data {
			if item.ContractCode == symbol {
				s.contractSize = decimal.NewFromFloat(item.ContractSize)
			}
		}
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := huobiTickerSocket(ctx, product, symbol, "ticker", logger, &ticker, &errCh); err == nil {
					return
				} else {
					logger.Warningf("Reconnect %s %s ticker stream with err: %s\n", symbol, product, err.Error())
				}
			}
		}
	}()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := s.maintainStreamTicker(ctx, product, symbol, &ticker, &errCh); err == nil {
					return
				} else {
					logger.Warningf("Refreshing %s %s ticker stream with err: %s\n", symbol, product, err.Error())
				}
			}
		}
	}()
	return &s
}

func (s *StreamTickerBranch) Close() {
	(*s.cancel)()
	s.bid.mux.Lock()
	s.bid.price = NullPrice
	s.bid.mux.Unlock()
	s.ask.mux.Lock()
	s.ask.price = NullPrice
	s.ask.mux.Unlock()
}

func (s *StreamTickerBranch) GetBid() (price, qty string, timeStamp time.Time, ok bool) {
	s.bid.mux.RLock()
	defer s.bid.mux.RUnlock()
	price = s.bid.price
	qty = s.bid.qty
	if price == NullPrice || price == "" {
		return price, qty, s.bid.timeStamp, false
	}
	return price, qty, s.bid.timeStamp, true
}

func (s *StreamTickerBranch) GetAsk() (price, qty string, timeStamp time.Time, ok bool) {
	s.ask.mux.RLock()
	defer s.ask.mux.RUnlock()
	price = s.ask.price
	qty = s.ask.qty
	if price == NullPrice || price == "" {
		return price, qty, s.ask.timeStamp, false
	}
	return price, qty, s.ask.timeStamp, true
}

func (s *StreamTickerBranch) updateBidData(price, qty string, stamp time.Time) {
	s.bid.mux.Lock()
	defer s.bid.mux.Unlock()
	s.bid.price = price
	s.bid.qty = qty
	s.bid.timeStamp = stamp
}

func (s *StreamTickerBranch) updateAskData(price, qty string, stamp time.Time) {
	s.ask.mux.Lock()
	defer s.ask.mux.Unlock()
	s.ask.price = price
	s.ask.qty = qty
	s.ask.timeStamp = stamp
}

func (s *StreamTickerBranch) maintainStreamTicker(
	ctx context.Context,
	product, symbol string,
	ticker *chan map[string]interface{},
	errCh *chan error,
) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case data := <-(*ticker):
			message := data["tick"].(map[string]interface{})
			var stamp time.Time
			if ts, ok := data["ts"].(float64); ok {
				stamp = time.UnixMilli(int64(ts))
			}
			var bidPrice, askPrice, bidQty, askQty string
			switch product {
			case "spot":
				if bid, ok := message["bid"].(float64); ok {
					bidDec := decimal.NewFromFloat(bid)
					bidPrice = bidDec.String()
				} else {
					bidPrice = NullPrice
				}
				if ask, ok := message["ask"].(float64); ok {
					askDec := decimal.NewFromFloat(ask)
					askPrice = askDec.String()
				} else {
					askPrice = NullPrice
				}
				if bidqty, ok := message["bidSize"].(float64); ok {
					bidQtyDec := decimal.NewFromFloat(bidqty)
					bidQty = bidQtyDec.String()
				}
				if askqty, ok := message["askSize"].(float64); ok {
					askQtyDec := decimal.NewFromFloat(askqty)
					askQty = askQtyDec.String()
				}
			case "swap":
				if bid, ok := message["bid"].([]interface{}); ok {
					bidDec := decimal.NewFromFloat(bid[0].(float64))
					bidPrice = bidDec.String()
					bidQtyDec := decimal.NewFromFloat(bid[1].(float64))
					intoSize := bidQtyDec.Mul(s.contractSize)
					bidQty = intoSize.String()
				} else {
					bidPrice = NullPrice
				}
				if ask, ok := message["ask"].([]interface{}); ok {
					askDec := decimal.NewFromFloat(ask[0].(float64))
					askPrice = askDec.String()
					askQtyDec := decimal.NewFromFloat(ask[1].(float64))
					intoSize := askQtyDec.Mul(s.contractSize)
					askQty = intoSize.String()
				} else {
					askPrice = NullPrice
				}
			}
			s.updateBidData(bidPrice, bidQty, stamp)
			s.updateAskData(askPrice, askQty, stamp)
		}
	}
}

func huobiTickerSocket(
	ctx context.Context,
	product, symbol, channel string,
	logger *log.Logger,
	mainCh *chan map[string]interface{},
	errCh *chan error,
) error {
	var w wS
	var duration time.Duration = 30
	w.Logger = logger
	w.OnErr = false
	var url string
	symbol = strings.ToLower(symbol)
	switch product {
	case "swap":
		url = "wss://api.hbdm.com/linear-swap-ws"
	case "spot":
		if channel == "orderbook" {
			url = "wss://api.huobi.pro/feed"
		} else {
			url = "wss://api.huobi.pro/ws"
		}
	default:
		return errors.New("not supported product, cancel socket connection")
	}
	// wait 5 second, if the hand shake fail, will terminate the dail
	dailCtx, _ := context.WithDeadline(ctx, time.Now().Add(time.Second*5))
	conn, _, err := websocket.DefaultDialer.DialContext(dailCtx, url, nil)
	if err != nil {
		return err
	}
	logger.Infof("Huobi %s %s ticker socket connected.\n", symbol, product)
	w.Conn = conn
	defer conn.Close()
	send, err := getHuobiSubscribeMessageForTicker(channel, symbol)
	if err != nil {
		return err
	}
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
		case err := <-*errCh:
			return err
		default:
			_, buf, err := w.Conn.ReadMessage()
			if err != nil {
				return err
			}
			res, err1 := huobiDecodingMap(&buf, logger)
			if err1 != nil {
				return err1
			}
			err2 := w.handleHuobiSocketData(product, &res, mainCh)
			if err2 != nil {
				return err2
			}
			if err := w.Conn.SetReadDeadline(time.Now().Add(time.Second * duration)); err != nil {
				return err
			}
		}
	}
}

func getHuobiSubscribeMessageForTicker(channel, symbol string) ([]byte, error) {
	var buffer bytes.Buffer
	switch channel {
	case "ticker":
		buffer.WriteString("market.")
		buffer.WriteString(symbol)
		buffer.WriteString(".bbo")
	default:
		return nil, errors.New("not supported channel, cancel socket connection")
	}
	sub := huobiSubscribeMessage{
		Sub: buffer.String(),
	}
	message, err := json.Marshal(sub)
	if err != nil {
		return nil, err
	}
	return message, nil
}

func (s *StreamTickerBranch) initialWithSpotDetail(product, symbol string) error {
	switch product {
	case "spot":
		client := New("", "", "", false)
		res, err := client.GetSpotDetail(symbol)
		if err != nil {
			return err
		}
		if !strings.EqualFold(res.Status, "ok") {
			return errors.New("return is not ok when intial spot detail")
		}
		// 0 => price, 1 => qty
		ts := time.UnixMilli(res.Ts)
		s.updateBidData(decimal.NewFromFloat(res.Tick.Bid[0]).String(), decimal.NewFromFloat(res.Tick.Bid[1]).String(), ts)
		s.updateAskData(decimal.NewFromFloat(res.Tick.Ask[0]).String(), decimal.NewFromFloat(res.Tick.Ask[1]).String(), ts)
	case "swap":
		//
	default:
		return errors.New("not supported product to initial spot detail")
	}

	return nil
}
