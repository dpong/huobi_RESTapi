package huobiapi

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"
)

const SpotBuy = "buy-limit"
const SpotSell = "sell-limit"
const SpotBuyMaker = "buy-limit-maker"
const SpotSellMaker = "sell-limit-maker"
const SpotSource = "spot-api"

type SpotPlaceOrderResponse struct {
	Status string `json:"status,omitempty"`
	Data   string `json:"data,omitempty"`
}
type SpotPlaceOrderOpts struct {
	AccountID     string `json:"account-id"`
	Symbol        string `json:"symbol"`
	Type          string `json:"type"`
	Amount        string `json:"amount"`
	Price         string `json:"price,omitempty"`
	Source        string `json:"source"`
	ClientOrderId string `json:"client-order-id,omitempty"`
	StopPrice     string `json:"stop-price,omitempty"`
	Operator      string `json:"operator,omitempty"`
}

// types :buy-market, sell-market, buy-limit, sell-limit, buy-ioc, sell-ioc, buy-limit-maker, sell-limit-maker, buy-stop-limit, sell-stop-limit, buy-limit-fok, sell-limit-fok, buy-stop-limit-fok, sell-stop-limit-fok
// source: spot-api, margin-api,super-margin-api,c2c-margin-api
// for buy market order, it's order value
func (p *Client) SpotPlaceOrder(opts SpotPlaceOrderOpts) (*SpotPlaceOrderResponse, error) {
	body, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	res, err := p.sendRequest("spot", http.MethodPost, "/v1/order/orders/place", body, nil, true)
	if err != nil {
		return nil, err
	}
	var result SpotPlaceOrderResponse
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (p *Client) SpotCancelOrder(orderID string) (*SpotPlaceOrderResponse, error) {
	var buffer bytes.Buffer
	buffer.WriteString("/v1/order/orders/")
	buffer.WriteString(orderID)
	buffer.WriteString("/submitcancel")
	res, err := p.sendRequest("spot", http.MethodPost, buffer.String(), nil, nil, true)
	if err != nil {
		return nil, err
	}
	var result SpotPlaceOrderResponse
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

type GetSpotOrderDetailResponse struct {
	Status string `json:"status"`
	Data   []struct {
		Symbol            string `json:"symbol"`
		FeeCurrency       string `json:"fee-currency"`
		Source            string `json:"source"`
		OrderID           int64  `json:"order-id"`
		Price             string `json:"price"`
		CreatedAt         int64  `json:"created-at"`
		Role              string `json:"role"`
		MatchID           int    `json:"match-id"`
		FilledAmount      string `json:"filled-amount"`
		FilledFees        string `json:"filled-fees"`
		FilledPoints      string `json:"filled-points"`
		FeeDeductCurrency string `json:"fee-deduct-currency"`
		FeeDeductState    string `json:"fee-deduct-state"`
		TradeID           int    `json:"trade-id"`
		ID                int64  `json:"id"`
		Type              string `json:"type"`
	} `json:"data"`
}

func (p *Client) GetSpotOrderDetail(orderID string) (*GetSpotOrderDetailResponse, error) {
	var buffer bytes.Buffer
	buffer.WriteString("/v1/order/orders/")
	buffer.WriteString(orderID)
	buffer.WriteString("/matchresults")
	res, err := p.sendRequest("spot", http.MethodGet, buffer.String(), nil, nil, true)
	if err != nil {
		return nil, err
	}
	var result GetSpotOrderDetailResponse
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

type SpotPlaceBatchOrdersOpts struct {
	AccountID     string `json:"account-id"`
	Symbol        string `json:"symbol"`
	Type          string `json:"type"`
	Amount        string `json:"amount"`
	Price         string `json:"price,omitempty"`
	Source        string `json:"source"`
	ClientOrderID string `json:"client-order-id,omitempty"`
}

type SpotPlaceBatchOrdersResponse struct {
	Status string `json:"status"`
	Data   []struct {
		OrderID       int64  `json:"order-id,omitempty"`
		ClientOrderID string `json:"client-order-id,omitempty"`
		ErrCode       string `json:"err-code,omitempty"`
		ErrMsg        string `json:"err-msg,omitempty"`
	} `json:"data"`
}

// max 10 orders
func (p *Client) SpotPlaceBatchOrders(opts []SpotPlaceBatchOrdersOpts) (*SpotPlaceBatchOrdersResponse, error) {
	body, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	res, err := p.sendRequest("spot", http.MethodPost, "/v1/order/batch-orders", body, nil, true)
	if err != nil {
		return nil, err
	}
	var result SpotPlaceBatchOrdersResponse
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

type SpotGetAllOpenOrdersResponse struct {
	Status string `json:"status"`
	Data   []struct {
		Symbol           string `json:"symbol"`
		Source           string `json:"source"`
		Price            string `json:"price"`
		CreatedAt        int64  `json:"created-at"`
		Amount           string `json:"amount"`
		AccountID        int    `json:"account-id"`
		FilledCashAmount string `json:"filled-cash-amount"`
		ClientOrderID    string `json:"client-order-id"`
		FilledAmount     string `json:"filled-amount"`
		FilledFees       string `json:"filled-fees"`
		ID               int64  `json:"id"`
		State            string `json:"state"`
		Type             string `json:"type"`
	} `json:"data"`
}

// if want all symbols, symbol = ""
func (p *Client) SpotGetAllOpenOrders(accountID int, symbol string) (*SpotGetAllOpenOrdersResponse, error) {
	params := make(map[string]string)
	params["account-id"] = strconv.Itoa(accountID)
	if symbol != "" {
		params["symbol"] = strings.ToLower(symbol)
	}
	res, err := p.sendRequest("spot", http.MethodGet, "/v1/order/openOrders", nil, &params, true)
	if err != nil {
		return nil, err
	}
	var result SpotGetAllOpenOrdersResponse
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

type SpotCancelBatchOrdersOpts struct {
	OrderIds []int64 `json:"order-ids"`
}

type SpotCancelBatchOrdersResponse struct {
	Status string `json:"status"`
	Data   struct {
		Success []string `json:"success"`
		Failed  []struct {
			ErrMsg        string `json:"err-msg"`
			OrderState    int    `json:"order-state"`
			OrderID       string `json:"order-id"`
			ErrCode       string `json:"err-code"`
			ClientOrderID string `json:"client-order-id"`
		} `json:"failed"`
	} `json:"data"`
}

// max 50 orders
func (p *Client) SpotCancelBatchOrders(opts SpotCancelBatchOrdersOpts) (*SpotCancelBatchOrdersResponse, error) {
	body, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	res, err := p.sendRequest("spot", http.MethodPost, "/v1/order/orders/batchcancel", body, nil, true)
	if err != nil {
		return nil, err
	}
	var result SpotCancelBatchOrdersResponse
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
