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
}

type tobBranch struct {
	mux   sync.RWMutex
	price string
	qty   string
}

// func SwapStreamTicker(symbol string, logger *log.Logger) *StreamTickerBranch {
// 	return localStreamTicker("swap", symbol, logger)
// }

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

func (s *StreamTickerBranch) GetBid() (price, qty string, ok bool) {
	s.bid.mux.RLock()
	defer s.bid.mux.RUnlock()
	price = s.bid.price
	qty = s.bid.qty
	if price == NullPrice || price == "" {
		return price, qty, false
	}
	return price, qty, true
}

func (s *StreamTickerBranch) GetAsk() (price, qty string, ok bool) {
	s.ask.mux.RLock()
	defer s.ask.mux.RUnlock()
	price = s.ask.price
	qty = s.ask.qty
	if price == NullPrice || price == "" {
		return price, qty, false
	}
	return price, qty, true
}

func (s *StreamTickerBranch) updateBidData(price, qty string) {
	s.bid.mux.Lock()
	defer s.bid.mux.Unlock()
	s.bid.price = price
	s.bid.qty = qty
}

func (s *StreamTickerBranch) updateAskData(price, qty string) {
	s.ask.mux.Lock()
	defer s.ask.mux.Unlock()
	s.ask.price = price
	s.ask.qty = qty
}

func (s *StreamTickerBranch) maintainStreamTicker(
	ctx context.Context,
	product, symbol string,
	ticker *chan map[string]interface{},
	errCh *chan error,
) error {
	lastUpdate := time.Now()
	for {
		select {
		case <-ctx.Done():
			return nil
		case message := <-(*ticker):
			var bidPrice, askPrice, bidQty, askQty string
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
			s.updateBidData(bidPrice, bidQty)
			s.updateAskData(askPrice, askQty)
			lastUpdate = time.Now()
		default:
			if time.Now().After(lastUpdate.Add(time.Second * 10)) {
				// 10 sec without updating
				err := errors.New("reconnect because of time out")
				*errCh <- err
				return err
			}
			time.Sleep(time.Millisecond * 100)
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
	var w huobiWebsocket
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
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}
	logger.Infof("Huobi %s %s ticker socket connected.\n", symbol, product)
	w.Conn = conn
	defer conn.Close()
	send, err := getHuobiSubscribeMessageForTicker(product, channel, symbol)
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
			if conn == nil {
				d := w.outHuobiErr()
				*mainCh <- d
				message := "Huobi reconnect..."
				logger.Infoln(message)
				return errors.New(message)
			}
			_, buf, err := conn.ReadMessage()
			if err != nil {
				d := w.outHuobiErr()
				*mainCh <- d
				message := "Huobi reconnect..."
				logger.Infoln(message)
				return errors.New(message)
			}
			res, err1 := huobiDecodingMap(&buf, logger)
			if err1 != nil {
				d := w.outHuobiErr()
				*mainCh <- d
				message := "Huobi reconnect..."
				logger.Infoln(message, err1)
				return err1
			}
			err2 := w.handleHuobiSocketData(product, &res, mainCh)
			if err2 != nil {
				d := w.outHuobiErr()
				*mainCh <- d
				message := "Huobi reconnect..."
				logger.Infoln(message, err2)
				return err2
			}
			if err := w.Conn.SetReadDeadline(time.Now().Add(time.Second * duration)); err != nil {
				return err
			}
		}
	}
}

func getHuobiSubscribeMessageForTicker(product, channel, symbol string) ([]byte, error) {
	var buffer bytes.Buffer
	switch product {
	case "spot":
		switch channel {
		case "ticker":
			buffer.WriteString("market.")
			buffer.WriteString(symbol)
			buffer.WriteString(".ticker")
		default:
			return nil, errors.New("not supported channel, cancel socket connection")
		}
	default:
		return nil, errors.New("not supported product, cancel socket connection")
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
