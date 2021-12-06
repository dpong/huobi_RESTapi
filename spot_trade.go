package huobiapi

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"
)

const SpotBatchBuy = "buy-limit"
const SpotBatchSell = "sell-limit"
const SpotBatchSource = "spot-api"

type SpotPlaceOrderResponse struct {
	Data string `json:"data,omitempty"`
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
func (p *Client) SpotPlaceOrder(opts SpotPlaceOrderOpts) (spot *SpotPlaceOrderResponse, err error) {
	body, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	res, err := p.sendRequest("spot", http.MethodPost, "/v1/order/orders/place", body, nil, true)
	if err != nil {
		return nil, err
	}
	// in Close()
	err = decode(res, &spot)
	if err != nil {
		return nil, err
	}
	return spot, nil
}

func (p *Client) SpotCancelOrder(orderID string) (spot *SpotPlaceOrderResponse, err error) {
	var buffer bytes.Buffer
	buffer.WriteString("/v1/order/orders/")
	buffer.WriteString(orderID)
	buffer.WriteString("/submitcancel")
	res, err := p.sendRequest("spot", http.MethodPost, buffer.String(), nil, nil, true)
	if err != nil {
		return nil, err
	}
	// in Close()
	err = decode(res, &spot)
	if err != nil {
		return nil, err
	}
	return spot, nil
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

func (p *Client) GetSpotOrderDetail(orderID string) (spot *GetSpotOrderDetailResponse, err error) {
	var buffer bytes.Buffer
	buffer.WriteString("/v1/order/orders/")
	buffer.WriteString(orderID)
	buffer.WriteString("/matchresults")
	res, err := p.sendRequest("spot", http.MethodGet, buffer.String(), nil, nil, true)
	if err != nil {
		return nil, err
	}
	// in Close()
	err = decode(res, &spot)
	if err != nil {
		return nil, err
	}
	return spot, nil
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
		ClientOrderID string `json:"client-order-id"`
		ErrCode       string `json:"err-code,omitempty"`
		ErrMsg        string `json:"err-msg,omitempty"`
	} `json:"data"`
}

func (p *Client) SpotPlaceBatchOrders(opts []SpotPlaceBatchOrdersOpts) (spot *SpotPlaceBatchOrdersResponse, err error) {
	body, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	res, err := p.sendRequest("spot", http.MethodPost, "/v1/order/batch-orders", body, nil, true)
	if err != nil {
		return nil, err
	}
	// in Close()
	err = decode(res, &spot)
	if err != nil {
		return nil, err
	}
	return spot, nil
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
func (p *Client) SpotGetAllOpenOrders(accountID int, symbol string) (spot *SpotGetAllOpenOrdersResponse, err error) {
	params := make(map[string]string)
	params["account-id"] = strconv.Itoa(accountID)
	if symbol != "" {
		params["symbol"] = strings.ToLower(symbol)
	}
	body, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	res, err := p.sendRequest("spot", http.MethodGet, "/v1/order/openOrders", body, &params, true)
	if err != nil {
		return nil, err
	}
	// in Close()
	err = decode(res, &spot)
	if err != nil {
		return nil, err
	}
	return spot, nil
}

type SpotCancelBatchOrdersOpts struct {
	OrderIds []string `json:"order-ids"`
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

func (p *Client) SpotCancelBatchOrders(opts SpotCancelBatchOrdersOpts) (spot *SpotCancelBatchOrdersResponse, err error) {
	body, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	res, err := p.sendRequest("spot", http.MethodPost, "/v1/order/orders/batchcancel", body, nil, true)
	if err != nil {
		return nil, err
	}
	// in Close()
	err = decode(res, &spot)
	if err != nil {
		return nil, err
	}
	return spot, nil
}
