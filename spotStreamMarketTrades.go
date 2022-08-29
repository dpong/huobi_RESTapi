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
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

type StreamMarketTradesBranch struct {
	cancel       *context.CancelFunc
	product      string
	symbol       string
	tradeChan    chan map[string]interface{}
	tradesBranch struct {
		Trades []PublicTradeData
		sync.Mutex
	}
	logger *logrus.Logger
}

// taker side
type PublicTradeData struct {
	Product string
	Symbol  string
	Side    string
	Price   decimal.Decimal
	Qty     decimal.Decimal
	Time    time.Time
}

func StreamTradeSpot(symbol string, logger *logrus.Logger) *StreamMarketTradesBranch {
	Usymbol := strings.ToLower(symbol)
	return streamTrade("spot", Usymbol, logger)
}

// side: Side of the taker in the trade
func (o *StreamMarketTradesBranch) GetTrades() []PublicTradeData {
	o.tradesBranch.Lock()
	defer o.tradesBranch.Unlock()
	trades := o.tradesBranch.Trades
	o.tradesBranch.Trades = []PublicTradeData{}
	return trades
}

func (o *StreamMarketTradesBranch) Close() {
	(*o.cancel)()
	o.tradesBranch.Lock()
	defer o.tradesBranch.Unlock()
	o.tradesBranch.Trades = []PublicTradeData{}
}

// spot only for now
func streamTrade(product, symbol string, logger *logrus.Logger) *StreamMarketTradesBranch {
	o := new(StreamMarketTradesBranch)
	ctx, cancel := context.WithCancel(context.Background())
	o.cancel = &cancel
	o.product = product
	o.symbol = symbol
	o.tradeChan = make(chan map[string]interface{}, 100)
	o.logger = logger
	errCh := make(chan error, 5)
	go o.maintainSession(ctx, &errCh)
	go o.listen(ctx)
	return o
}

func (o *StreamMarketTradesBranch) listen(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case trade := <-o.tradeChan:
			data := new(PublicTradeData)
			data.Symbol = o.symbol
			data.Product = o.product

			if ts, ok := trade["ts"].(float64); ok {
				timeStamp := time.UnixMilli(int64(ts))
				data.Time = timeStamp
			}
			if priceFlo, ok := trade["price"].(float64); ok {
				priceDec := decimal.NewFromFloat(priceFlo)
				data.Price = priceDec
			}
			if qtyFlo, ok := trade["amount"].(float64); ok {
				qtyDec := decimal.NewFromFloat(qtyFlo)
				data.Qty = qtyDec
			}
			if direction, ok := trade["direction"].(string); ok {
				data.Side = direction
			}
			o.appendNewTrade(data)
		}
	}
}

func (o *StreamMarketTradesBranch) appendNewTrade(new *PublicTradeData) {
	o.tradesBranch.Lock()
	defer o.tradesBranch.Unlock()
	o.tradesBranch.Trades = append(o.tradesBranch.Trades, *new)
}

func (o *StreamMarketTradesBranch) maintainSession(ctx context.Context, errCh *chan error) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := huobiPublicTradeSocket(ctx, o.product, o.symbol, "trade", o.logger, &o.tradeChan, errCh); err == nil {
				return
			} else {
				o.logger.Warningf("reconnect Huobi %s trade stream with err: %s\n", o.symbol, err.Error())
			}
		}
	}
}

func huobiPublicTradeSocket(
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
	defer conn.Close()
	w.Conn = conn
	send, err := getHuobiSubscribeMessageForPublicTrade(channel, symbol)
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

func getHuobiSubscribeMessageForPublicTrade(channel, symbol string) ([]byte, error) {
	var buffer bytes.Buffer
	switch channel {
	case "trade":
		buffer.WriteString("market.")
		buffer.WriteString(symbol)
		buffer.WriteString(".trade.detail")
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
