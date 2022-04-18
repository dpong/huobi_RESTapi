package huobiapi

import (
	"bytes"
	"fmt"
	"net/http"
)

type FundingResponse struct {
	Status string `json:"status"`
	Data   struct {
		TotalPage   int `json:"total_page"`
		CurrentPage int `json:"current_page"`
		TotalSize   int `json:"total_size"`
		Data        []struct {
			Rate            string `json:"funding_rate"`
			RealizedRate    string `json:"realized_rate"`
			Time            string `json:"funding_time"`
			Future          string `json:"contract_code"`
			Symbol          string `json:"symbol"`
			FeeAsset        string `json:"fee_asset"`
			AvgPremiumIndex string `json:"avg_premium_index"`
		} `json:"data"`
	} `json:"data"`
	Ts int64 `json:"ts"`
}

func (p *Client) Fundings(symbol string, pages int) (*FundingResponse, error) {
	params := make(map[string]string)
	params["page_size"] = fmt.Sprintf("%v", pages)
	params["contract_code"] = symbol
	body, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	res, err := p.sendRequest("swap", http.MethodGet, "/linear-swap-api/v1/swap_historical_funding_rate", body, &params, false)
	if err != nil {
		return nil, err
	}
	var result FundingResponse
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
