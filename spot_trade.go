package huobiapi

import (
	"bytes"
	"net/http"
)

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
		p.Logger.Println(err)
		return nil, err
	}
	res, err := p.sendRequest("spot", http.MethodPost, "/v1/order/orders/place", body, nil, true)
	if err != nil {
		p.Logger.Println(err)
		return nil, err
	}
	// in Close()
	err = decode(res, &spot)
	if err != nil {
		p.Logger.Println(err)
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
		p.Logger.Println(err)
		return nil, err
	}
	// in Close()
	err = decode(res, &spot)
	if err != nil {
		p.Logger.Println(err)
		return nil, err
	}
	return spot, nil
}
