package huobiapi

import (
	"net/http"
	"strconv"
	"strings"
)

type WithdrawResponse struct {
	Data int `json:"data"`
}

// "trc20usdt" as chain parameter
func (p *Client) Withdraw(currency, chain, address string, amount, fee float64) (spot *WithdrawResponse, err error) {
	params := make(map[string]string)
	params["address"] = address
	params["currency"] = strings.ToLower(currency)
	params["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	params["fee"] = strconv.FormatFloat(fee, 'f', -1, 64)
	params["chain"] = chain
	body, err := json.Marshal(params)
	if err != nil {
		p.Logger.Println(err)
		return nil, err
	}
	res, err := p.sendRequest("spot", http.MethodPost, "/v1/dw/withdraw/api/create", body, nil, true)
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
