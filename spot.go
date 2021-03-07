package huobiapi

import (
	"net/http"
)

type GetAllAccountsResponse struct {
	Data []struct {
		ID      int    `json:"id"`
		Type    string `json:"type"`
		Subtype string `json:"subtype"`
		State   string `json:"state"`
	} `json:"data"`
}

func (p *Client) GetAllAccounts() (swaps *GetAllAccountsResponse, err error) {
	res, err := p.sendRequest("spot", http.MethodGet, "/v1/account/accounts", nil, true)
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
