package huobiapi

import (
	"bytes"
	"net/http"
	"strconv"
)

type SpotsResponse struct {
	Status string `json:"status"`
	Data   []struct {
		BaseCurrency             string  `json:"base-currency"`
		QuoteCurrency            string  `json:"quote-currency"`
		PricePrecision           int     `json:"price-precision"`
		AmountPrecision          int     `json:"amount-precision"`
		SymbolPartition          string  `json:"symbol-partition,omitempty"`
		Symbol                   string  `json:"symbol"`
		State                    string  `json:"state"`
		ValuePrecision           int     `json:"value-precision"`
		MinOrderAmt              float64 `json:"min-order-amt"`
		MaxOrderAmt              float64 `json:"max-order-amt,omitempty"`
		MinOrderValue            float64 `json:"min-order-value"`
		LimitOrderMinOrderAmt    float64 `json:"limit-order-min-order-amt,omitempty"`
		LimitOrderMaxOrderAmt    float64 `json:"limit-order-max-order-amt,omitempty"`
		SellMarketMinOrderAmt    float64 `json:"sell-market-min-order-amt,omitempty"`
		SellMarketMaxOrderAmt    float64 `json:"sell-market-max-order-amt,omitempty"`
		BuyMarketMaxOrderValue   float64 `json:"buy-market-max-order-value,omitempty"`
		LeverageRatio            float64 `json:"leverage-ratio,omitempty"`
		SuperMarginLeverageRatio float64 `json:"super-margin-leverage-ratio,omitempty"`
		FundingLeverageRatio     float64 `json:"funding-leverage-ratio,omitempty"`
		APITrading               string  `json:"api-trading"`
		Tags                     string  `json:"tags,omitempty"`
	} `json:"data"`
}

func (p *Client) Spots() (spots *SpotsResponse, err error) {
	res, err := p.sendRequest("spot", http.MethodGet, "/v1/common/symbols", nil, nil, false)
	if err != nil {
		return nil, err
	}
	// in Close()
	err = decode(res, &spots)
	if err != nil {
		return nil, err
	}
	return spots, nil
}

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
		return nil, err
	}
	// in Close()
	err = decode(res, &swaps)
	if err != nil {
		return nil, err
	}
	return swaps, nil
}

type GetAccountDataResponse struct {
	Status string `json:"status"`
	Data   struct {
		ID    int             `json:"id"`
		Type  string          `json:"type"`
		State string          `json:"state"`
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
		return nil, err
	}
	// in Close()
	err = decode(res, &swaps)
	if err != nil {
		return nil, err
	}
	return swaps, nil
}
