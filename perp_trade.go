package huobiapi

import (
	"bytes"
	"errors"
	"net/http"
	"strings"
)

const PerpLimit = "limit"
const PerpPostOnly = "post_only"

type PerpPlaceOrderResponse struct {
	Status string `json:"status"`
	Data   struct {
		OrderID    int64  `json:"order_id"`
		OrderIDStr string `json:"order_id_str"`
	} `json:"data"`
	Ts int64 `json:"ts"`
}
type PerpPlaceOrderOpts struct {
	Symbol           string  `json:"contract_code"`
	ClientOrderId    int64   `json:"client_order_id,omitempty"`
	Price            float64 `json:"price,omitempty"`
	ContractSize     int64   `json:"volume"`
	Side             string  `json:"direction"`
	Offset           string  `json:"offset"`
	LeverRate        int     `json:"lever_rate"`
	OrderType        string  `json:"order_price_type"`
	ReduceOnly       int     `json:"reduce_only,omitempty"`
	TpTriggerPrice   float32 `json:"tp_trigger_price,omitempty"`
	TpOrderPrice     float32 `json:"tp_order_price,omitempty"`
	TpOrderPriceType string  `json:"tp_order_price_type,omitempty"`
	SlTriggerPrice   float32 `json:"sl_trigger_price,omitempty"`
	SlOrderPrice     float32 `json:"sl_order_price,omitempty"`
	SlOrderPriceType string  `json:"sl_order_price_type,omitempty"`
}

func (p *Client) PerpPlaceOrder(mode string, opts PerpPlaceOrderOpts) (*PerpPlaceOrderResponse, error) {
	var path string
	switch strings.ToLower(mode) {
	case "cross":
		path = "/linear-swap-api/v1/swap_cross_order"
	case "iso":
		path = "/linear-swap-api/v1/swap_order"
	default:
		return nil, errors.New("invaild mode for place swap order, choose mode between cross or iso")
	}
	body, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	res, err := p.sendRequest("swap", http.MethodPost, path, body, nil, true)
	if err != nil {
		return nil, err
	}
	var result PerpPlaceOrderResponse
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

type PerpCancelOrderOpts struct {
	Symbol        string `json:"contract_code"`
	ClientOrderId string `json:"client_order_id,omitempty"`
	OrderID       string `json:"order_id"`
}

type PerpCancelOrderResponse struct {
	Status string `json:"status"`
	Data   struct {
		Errors []struct {
			OrderID string `json:"order_id"`
			ErrCode int    `json:"err_code"`
			ErrMsg  string `json:"err_msg"`
		} `json:"errors"`
		Successes string `json:"successes"`
	} `json:"data"`
	Ts int64 `json:"ts"`
}

func (p *Client) PerpCancelOrder(mode string, opts PerpCancelOrderOpts) (*PerpCancelOrderResponse, error) {
	var path string
	switch strings.ToLower(mode) {
	case "cross":
		path = "/linear-swap-api/v1/swap_cross_cancel"
	case "iso":
		path = "/linear-swap-api/v1/swap_cancel"
	default:
		return nil, errors.New("invaild mode for cancel swap order, choose mode between cross or iso")
	}
	body, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	res, err := p.sendRequest("swap", http.MethodPost, path, body, nil, true)
	if err != nil {
		return nil, err
	}
	var result PerpCancelOrderResponse
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

type PerpQueryOrderResponse struct {
	Status string `json:"status"`
	Data   []struct {
		Symbol          string      `json:"symbol"`
		ContractCode    string      `json:"contract_code"`
		Volume          int         `json:"volume"`
		Price           float64     `json:"price"`
		OrderPriceType  string      `json:"order_price_type"`
		OrderType       int         `json:"order_type"`
		Direction       string      `json:"direction"`
		Offset          string      `json:"offset"`
		LeverRate       int         `json:"lever_rate"`
		OrderID         int64       `json:"order_id"`
		ClientOrderID   interface{} `json:"client_order_id"`
		CreatedAt       int64       `json:"created_at"`
		TradeVolume     int         `json:"trade_volume"`
		TradeTurnover   float64     `json:"trade_turnover"`
		Fee             float64     `json:"fee"`
		TradeAvgPrice   float64     `json:"trade_avg_price"`
		MarginFrozen    float64     `json:"margin_frozen"`
		Profit          float64     `json:"profit"`
		Status          int         `json:"status"`
		OrderSource     string      `json:"order_source"`
		OrderIDStr      string      `json:"order_id_str"`
		FeeAsset        string      `json:"fee_asset"`
		LiquidationType string      `json:"liquidation_type"`
		CanceledAt      int         `json:"canceled_at"`
		MarginAsset     string      `json:"margin_asset"`
		MarginAccount   string      `json:"margin_account"`
		MarginMode      string      `json:"margin_mode"`
		IsTpsl          int         `json:"is_tpsl"`
		RealProfit      float64     `json:"real_profit"`
	} `json:"data"`
	Ts int64 `json:"ts"`
}

func (p *Client) PerpQueryOrder(mode string, opts PerpCancelOrderOpts) (*PerpQueryOrderResponse, error) {
	var path string
	switch strings.ToLower(mode) {
	case "cross":
		path = "/linear-swap-api/v1/swap_cross_order_info"
	case "iso":
		path = "/linear-swap-api/v1/swap_order_info"
	default:
		return nil, errors.New("invaild mode for query swap order, choose mode between cross or iso")
	}
	body, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	res, err := p.sendRequest("swap", http.MethodPost, path, body, nil, true)
	if err != nil {
		return nil, err
	}
	var result PerpQueryOrderResponse
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

type OnlySymbolOpts struct {
	Symbol string `json:"contract_code"`
}

type CancelAllPerpOrdersResponse struct {
	Status string `json:"status"`
	Data   struct {
		Errors    []interface{} `json:"errors"`
		Successes string        `json:"successes"`
	} `json:"data"`
	Ts int64 `json:"ts"`
}

func (p *Client) CancelAllPerpOrders(mode string, symbol string) (*CancelAllPerpOrdersResponse, error) {
	var path string
	switch strings.ToLower(mode) {
	case "cross":
		path = "/linear-swap-api/v1/swap_cross_cancelall"
	case "iso":
		path = "/linear-swap-api/v1/swap_cancelall"
	default:
		return nil, errors.New("invaild mode for query swap order, choose mode between cross or iso")
	}
	opts := OnlySymbolOpts{
		Symbol: symbol,
	}
	body, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	res, err := p.sendRequest("swap", http.MethodPost, path, body, nil, true)
	if err != nil {
		return nil, err
	}
	var result CancelAllPerpOrdersResponse
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

type GetPerpOpenOrdersResponse struct {
	Status string `json:"status"`
	Data   struct {
		Orders []struct {
			UpdateTime      int64       `json:"update_time"`
			Symbol          string      `json:"symbol"`
			ContractCode    string      `json:"contract_code"`
			Volume          float64     `json:"volume"`
			Price           float64     `json:"price"`
			OrderPriceType  string      `json:"order_price_type"`
			OrderType       int         `json:"order_type"`
			Direction       string      `json:"direction"`
			Offset          string      `json:"offset"`
			LeverRate       int         `json:"lever_rate"`
			OrderID         int64       `json:"order_id"`
			ClientOrderID   interface{} `json:"client_order_id"`
			CreatedAt       int64       `json:"created_at"`
			TradeVolume     float64     `json:"trade_volume"`
			TradeTurnover   float64     `json:"trade_turnover"`
			Fee             float64     `json:"fee"`
			TradeAvgPrice   float64     `json:"trade_avg_price"`
			MarginFrozen    float64     `json:"margin_frozen"`
			Profit          float64     `json:"profit"`
			Status          int         `json:"status"`
			OrderSource     string      `json:"order_source"`
			OrderIDStr      string      `json:"order_id_str"`
			FeeAsset        string      `json:"fee_asset"`
			LiquidationType string      `json:"liquidation_type"`
			CanceledAt      int64       `json:"canceled_at"`
			MarginAsset     string      `json:"margin_asset"`
			MarginAccount   string      `json:"margin_account"`
			MarginMode      string      `json:"margin_mode"`
			IsTpsl          int         `json:"is_tpsl"`
			RealProfit      float64     `json:"real_profit"`
			TradePartition  string      `json:"trade_partition"`
		} `json:"orders"`
		TotalPage   int `json:"total_page"`
		CurrentPage int `json:"current_page"`
		TotalSize   int `json:"total_size"`
	} `json:"data"`
	Ts int64 `json:"ts"`
}

func (p *Client) GetPerpOpenOrders(mode string, symbol string) (*GetPerpOpenOrdersResponse, error) {
	var path string
	switch strings.ToLower(mode) {
	case "cross":
		path = "/linear-swap-api/v1/swap_cross_openorders"
	case "iso":
		path = "/linear-swap-api/v1/swap_openorders"
	default:
		return nil, errors.New("invaild mode for query swap order, choose mode between cross or iso")
	}
	opts := OnlySymbolOpts{
		Symbol: symbol,
	}
	body, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	res, err := p.sendRequest("swap", http.MethodPost, path, body, nil, true)
	if err != nil {
		return nil, err
	}
	var swaps GetPerpOpenOrdersResponse
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	err = json.Unmarshal(buf.Bytes(), &swaps)
	if err != nil {
		return nil, err
	}
	return &swaps, nil
}

type PerpPlaceBatchOrdersOpts struct {
	Symbol           string  `json:"contract_code"`
	ClientOrderId    int64   `json:"client_order_id,omitempty"`
	Price            float64 `json:"price,omitempty"`
	ContractSize     int64   `json:"volume"`
	Side             string  `json:"direction"`
	Offset           string  `json:"offset"`
	LeverRate        int     `json:"lever_rate"`
	OrderType        string  `json:"order_price_type"`
	ReduceOnly       int     `json:"reduce_only,omitempty"`
	TpTriggerPrice   float32 `json:"tp_trigger_price,omitempty"`
	TpOrderPrice     float32 `json:"tp_order_price,omitempty"`
	TpOrderPriceType string  `json:"tp_order_price_type,omitempty"`
	SlTriggerPrice   float32 `json:"sl_trigger_price,omitempty"`
	SlOrderPrice     float32 `json:"sl_order_price,omitempty"`
	SlOrderPriceType string  `json:"sl_order_price_type,omitempty"`
}

type PerpPlaceBatchOrdersResponse struct {
	Status string `json:"status"`
	Data   struct {
		Errors []struct {
			Index   int    `json:"index"`
			ErrCode int    `json:"err_code"`
			ErrMsg  string `json:"err_msg"`
		} `json:"errors"`
		Success []struct {
			OrderID       int64  `json:"order_id,omitempty"`
			ClientOrderID int64  `json:"client_order_id,omitempty"`
			Index         int    `json:"index"`
			OrderIDStr    string `json:"order_id_str"`
		} `json:"success"`
	} `json:"data"`
	Ts int64 `json:"ts"`
}

// not available yet
// max 10 orders
func (p *Client) PerpPlaceBatchOrders(mode string, opts []PerpPlaceBatchOrdersOpts) (*PerpPlaceBatchOrdersResponse, error) {
	var path string
	switch strings.ToLower(mode) {
	case "cross":
		path = "/linear-swap-api/v1/swap_cross_batchorder"
	case "iso":
		path = "/linear-swap-api/v1/swap_batchorder"
	default:
		return nil, errors.New("invaild mode for query swap order, choose mode between cross or iso")
	}
	body, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	res, err := p.sendRequest("swap", http.MethodPost, path, body, nil, true)
	if err != nil {
		return nil, err
	}
	var result PerpPlaceBatchOrdersResponse
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
