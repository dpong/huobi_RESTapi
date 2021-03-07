package huobiapi

import (
	"net/http"
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

func (p *Client) Swaps() (swaps *SwapsResponse) {
	res, err := p.sendRequest("swap", http.MethodGet, "/linear-swap-api/v1/swap_contract_info", nil, false)
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
	res, err := p.sendRequest("swap", http.MethodGet, "/linear-swap-api/v1/swap_open_interest", nil, false)
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
	res, err := p.sendRequest("swap", http.MethodGet, "/linear-swap-api/v1/swap_batch_funding_rate", nil, false)
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

type SwapCrossMarginAccountInfoResponse struct {
	Status string `json:"status"`
	Data   []struct {
		MarginMode        string                          `json:"margin_mode"`
		MarginAccount     string                          `json:"margin_account"`
		MarginAsset       string                          `json:"margin_asset"`
		MarginBalance     float64                         `json:"margin_balance"`
		MarginStatic      float64                         `json:"margin_static"`
		MarginPosition    float64                         `json:"margin_position"`
		MarginFrozen      float64                         `json:"margin_frozen"`
		ProfitReal        float64                         `json:"profit_real"`
		ProfitUnreal      float64                         `json:"profit_unreal"`
		WithdrawAvailable float64                         `json:"withdraw_available"`
		RiskRate          float64                         `json:"risk_rate"`
		ContractDetail    []SwapCrossMarginContractDetail `json:"contract_detail"`
	} `json:"data"`
	Ts int64 `json:"ts"`
}

type SwapCrossMarginContractDetail struct {
	Symbol           string  `json:"symbol"`
	ContractCode     string  `json:"contract_code"`
	MarginPosition   float64 `json:"margin_position"`
	MarginFrozen     float64 `json:"margin_frozen"`
	MarginAvailable  float64 `json:"margin_available"`
	ProfitUnreal     float64 `json:"profit_unreal"`
	LiquidationPrice float64 `json:"liquidation_price"`
	LeverRate        int     `json:"lever_rate"`
	AdjustFactor     float64 `json:"adjust_factor"`
}

func (p *Client) SwapAccountInfo() (swaps *SwapCrossMarginAccountInfoResponse) {
	res, err := p.sendRequest("swap", http.MethodPost, "/linear-swap-api/v1/swap_cross_account_info", nil, true)
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
	res, err := p.sendRequest("swap", http.MethodPost, "/linear-swap-api/v1/swap_cross_available_level_rate", nil, true)
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
