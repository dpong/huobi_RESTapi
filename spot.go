package huobiapi

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"
)

type SpotsResponse struct {
	Status string `json:"status"`
	Data   []struct {
		BaseCurrency                    string  `json:"base-currency"`
		QuoteCurrency                   string  `json:"quote-currency"`
		PricePrecision                  int     `json:"price-precision"`
		AmountPrecision                 int     `json:"amount-precision"`
		SymbolPartition                 string  `json:"symbol-partition"`
		Symbol                          string  `json:"symbol"`
		State                           string  `json:"state"`
		ValuePrecision                  int     `json:"value-precision"`
		MinOrderAmt                     float64 `json:"min-order-amt"`
		MaxOrderAmt                     float64 `json:"max-order-amt"`
		MinOrderValue                   float64 `json:"min-order-value"`
		LimitOrderMinOrderAmt           float64 `json:"limit-order-min-order-amt"`
		LimitOrderMaxOrderAmt           float64 `json:"limit-order-max-order-amt"`
		LimitOrderMaxBuyAmt             float64 `json:"limit-order-max-buy-amt"`
		LimitOrderMaxSellAmt            float64 `json:"limit-order-max-sell-amt"`
		BuyLimitMustLessThan            float64 `json:"buy-limit-must-less-than"`
		SellLimitMustGreaterThan        float64 `json:"sell-limit-must-greater-than"`
		SellMarketMinOrderAmt           float64 `json:"sell-market-min-order-amt"`
		SellMarketMaxOrderAmt           float64 `json:"sell-market-max-order-amt"`
		BuyMarketMaxOrderValue          float64 `json:"buy-market-max-order-value"`
		MarketSellOrderRateMustLessThan float64 `json:"market-sell-order-rate-must-less-than"`
		MarketBuyOrderRateMustLessThan  float64 `json:"market-buy-order-rate-must-less-than"`
		APITrading                      string  `json:"api-trading"`
		Tags                            string  `json:"tags"`
	} `json:"data"`
}

func (p *Client) Spots() (*SpotsResponse, error) {
	res, err := p.sendRequest("spot", http.MethodGet, "/v1/common/symbols", nil, nil, false)
	if err != nil {
		return nil, err
	}
	var result SpotsResponse
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

type GetAllAccountsResponse struct {
	Data []struct {
		ID      int    `json:"id"`
		Type    string `json:"type"`
		Subtype string `json:"subtype"`
		State   string `json:"state"`
	} `json:"data"`
}

func (p *Client) GetAllAccounts() (*GetAllAccountsResponse, error) {
	res, err := p.sendRequest("spot", http.MethodGet, "/v1/account/accounts", nil, nil, true)
	if err != nil {
		return nil, err
	}
	var result GetAllAccountsResponse
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
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

func (p *Client) GetAccountData(id int) (*GetAccountDataResponse, error) {
	var buffer bytes.Buffer
	buffer.WriteString("/v1/account/accounts/")
	buffer.WriteString(strconv.Itoa(id))
	buffer.WriteString("/balance")
	res, err := p.sendRequest("spot", http.MethodGet, buffer.String(), nil, nil, true)
	if err != nil {
		return nil, err
	}
	var result GetAccountDataResponse
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

type GetLastTradeResponse struct {
	Ch     string  `json:"ch"`
	Status string  `json:"status"`
	Ts     float64 `json:"ts"`
	Tick   struct {
		ID   float64 `json:"id"`
		Ts   float64 `json:"ts"`
		Data []struct {
			ID        float64 `json:"id"`
			Ts        float64 `json:"ts"`
			TradeID   float64 `json:"trade-id"`
			Amount    float64 `json:"amount"`
			Price     float64 `json:"price"`
			Direction string  `json:"direction"`
		} `json:"data"`
	} `json:"tick"`
}

// symbol ex: btcusdt
func (p *Client) GetLastTrade(symbol string) (*GetLastTradeResponse, error) {
	url := "/market/trade"
	params := make(map[string]string)
	params["symbol"] = strings.ToLower(symbol)
	res, err := p.sendRequest("spot", http.MethodGet, url, nil, &params, false)
	if err != nil {
		return nil, err
	}
	var result GetLastTradeResponse
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
