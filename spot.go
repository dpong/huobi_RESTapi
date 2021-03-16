package huobiapi

import (
	"bytes"
	"net/http"
	"strconv"
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
	res, err := p.sendRequest("spot", http.MethodGet, "/v1/account/accounts", nil, nil, true)
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

type GetAccountDataResponse struct {
	Status string `json:"status"`
	Data   struct {
		ID    int            `json:"id"`
		Type  string         `json:"type"`
		State string         `json:"state"`
		List  []*SpotListData `json:"list"`
	} `json:"data"`
}

type SpotListData struct {
	Currency string `json:"currency"`
	Type     string `json:"type"`
	Balance  string `json:"balance"`
}

func (p *Client) GetAccountData(id int) (swaps *GetAccountDataResponse, err error) {
	var buffer bytes.Buffer
	buffer.WriteString("/v1/account/accounts/")
	buffer.WriteString(strconv.Itoa(id))
	buffer.WriteString("/balance")
	res, err := p.sendRequest("spot", http.MethodGet, buffer.String(), nil, nil, true)
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
