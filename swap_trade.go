package huobiapi

import (
	"net/http"
)

type SwapPlaceOrderResponse struct {
	Status string `json:"status"`
	Data   struct {
		OrderID    int64  `json:"order_id"`
		OrderIDStr string `json:"order_id_str"`
	} `json:"data"`
	Ts int64 `json:"ts"`
}
type SwapPlaceOrderOpts struct {
	Symbol           string  `json:"contract_code"`
	ClientOrderId    int64   `json:"client_order_id,omitempty"`
	Price            float32 `json:"price,omitempty"`
	ContractSize     int64   `json:"volume"`
	Side             string  `json:"direction"`
	Offset           string  `json:"offset"`
	LeverRate        int     `json:"lever_rate"`
	OrderType        string  `json:"order_price_type"`
	TpTriggerPrice   float32 `json:"tp_trigger_price,omitempty"`
	TpOrderPrice     float32 `json:"tp_order_price,omitempty"`
	TpOrderPriceType string  `json:"tp_order_price_type,omitempty"`
	SlTriggerPrice   float32 `json:"sl_trigger_price,omitempty"`
	SlOrderPrice     float32 `json:"sl_order_price,omitempty"`
	SlOrderPriceType string  `json:"sl_order_price_type,omitempty"`
}

func (p *Client) SwapPlaceOrder(opts SwapPlaceOrderOpts) (swaps *SwapPlaceOrderResponse, err error) {
	body, err := json.Marshal(opts)
	if err != nil {
		p.Logger.Println(err)
		return nil, err
	}
	res, err := p.sendRequest("swap", http.MethodPost, "/linear-swap-api/v1/swap_cross_order", body, nil, true)
	if err != nil {
		p.Logger.Println(err)
		return nil, err
	}
	// in Close()
	err = decode(res, &swaps)
	if err != nil {
		p.Logger.Println(err)
		return nil, err
	}
	return swaps, nil
}
