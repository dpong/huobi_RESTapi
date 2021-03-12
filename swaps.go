package huobiapi

import (
	"net/http"
	"strconv"
	"strings"
)

type SwapsResponse struct {
	Status string `json:"status"`
	Data   []struct {
		Symbol         string  `json:"symbol"`
		ContractCode   string  `json:"contract_code"`
		ContractSize   float64 `json:"contract_size"`
		PriceTick      float64 `json:"price_tick"`
		SettlementDate string  `json:"settlement_date"`
		CreateDate     string  `json:"create_date"`
		ContractStatus int     `json:"contract_status"`
	} `json:"data"`
	Ts int64 `json:"ts"`
}

func (p *Client) Swaps(mode string) (swaps *SwapsResponse, err error) {
	params := make(map[string]string)
	if mode != "" {
		params["support_margin_mode"] = strings.ToLower(mode)
	}
	body, err := json.Marshal(params)
	if err != nil {
		p.Logger.Println(err)
		return nil, err
	}
	res, err := p.sendRequest("swap", http.MethodGet, "/linear-swap-api/v1/swap_contract_info", body, &params, false)
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

type SwapOpenInterestResponse struct {
	Status string `json:"status"`
	Data   []struct {
		Volume        float64 `json:"volume"`
		Amount        float64 `json:"amount"`
		Symbol        string  `json:"symbol"`
		Value         float64 `json:"value"`
		ContractCode  string  `json:"contract_code"`
		TradeAmount   float64 `json:"trade_amount"`
		TradeVolume   float64 `json:"trade_volume"`
		TradeTurnover float64 `json:"trade_turnover"`
	} `json:"data"`
	Ts int64 `json:"ts"`
}

func (p *Client) SwapOpenInterests() (swaps *SwapOpenInterestResponse) {
	res, err := p.sendRequest("swap", http.MethodGet, "/linear-swap-api/v1/swap_open_interest", nil, nil, false)
	if err != nil {
		p.Logger.Println(err)
		return nil
	}
	// in Close()
	err = decode(res, &swaps)
	if err != nil {
		p.Logger.Println(err)
		return nil
	}
	return swaps
}

type SwapNextFundingResponse struct {
	Status string `json:"status"`
	Data   []struct {
		EstimatedRate   string `json:"estimated_rate"`
		FundingRate     string `json:"funding_rate"`
		ContractCode    string `json:"contract_code"`
		Symbol          string `json:"symbol"`
		FeeAsset        string `json:"fee_asset"`
		FundingTime     string `json:"funding_time"`
		NextFundingTime string `json:"next_funding_time"`
	} `json:"data"`
	Ts int64 `json:"ts"`
}

func (p *Client) SwapNextFundings() (swaps *SwapNextFundingResponse) {
	res, err := p.sendRequest("swap", http.MethodGet, "/linear-swap-api/v1/swap_batch_funding_rate", nil, nil, false)
	if err != nil {
		p.Logger.Println(err)
		return nil
	}
	// in Close()
	err = decode(res, &swaps)
	if err != nil {
		p.Logger.Println(err)
		return nil
	}
	return swaps
}


type SwapCrossMarginLeverages struct {
	Status string `json:"status"`
	Data   []struct {
		ContractCode       string `json:"contract_code"`
		AvailableLevelRate string `json:"available_level_rate"`
		MarginMode         string `json:"margin_mode"`
	} `json:"data"`
	Ts int64 `json:"ts"`
}

func (p *Client) SwapLeverages() (swaps *SwapCrossMarginLeverages, err error) {
	res, err := p.sendRequest("swap", http.MethodPost, "/linear-swap-api/v1/swap_cross_available_level_rate", nil, nil, true)
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

type SwapAssetTransferResponse struct {
	Code    int    `json:"code"`
	Data    int    `json:"data"`
	Message string `json:"message"`
	Success bool   `json:"success"`
}

func (p *Client) SwapAssetTransfer(from, to, currency, marginAccount string, amount float64) (swaps *SwapAssetTransferResponse, err error) {
	params := make(map[string]string)
	params["from"] = strings.ToLower(from) //spot、linear-swap
	params["to"] = strings.ToLower(to)
	params["currency"] = currency
	params["margin-account"] = marginAccount //e.g. BTC-USDT，ETH-USDT, USDT(cross margin)
	params["amount"] = strconv.FormatFloat(amount, 'f', 8, 64)
	body, err := json.Marshal(params)
	if err != nil {
		p.Logger.Println(err)
		return nil, err
	}
	res, err := p.sendRequest("spot", http.MethodPost, "/v2/account/transfer", body, &params, true)
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
