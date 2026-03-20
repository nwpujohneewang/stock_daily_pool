package limiter

import "stock/internal/model"

func CalcLimitUpPrice(preClose float64, limitUpRatio float64) float64 {
	if preClose <= 0 {
		return 0
	}
	return float64(int(preClose*(1+limitUpRatio)*100)) / 100
}

func DetectBoard(symbol string) model.BoardCode {
	switch {
	case len(symbol) >= 3 && symbol[:3] == "688":
		return model.BoardSTAR
	case len(symbol) >= 3 && (symbol[:3] == "300" || symbol[:3] == "301"):
		return model.BoardGEM
	case len(symbol) >= 1 && (symbol[0] == '8' || symbol[0] == '4'):
		return model.BoardBSE
	default:
		return model.BoardMain
	}
}

func DetectLimitUp(input model.DetectInput, rule *model.BoardRule) model.DetectOutput {
	output := model.DetectOutput{
		Skipped: false,
	}

	if input.IsST {
		output.Skipped = true
		output.SkipReason = "ST stock"
		return output
	}

	if input.PreClose <= 0 {
		output.Skipped = true
		output.SkipReason = "invalid pre_close"
		return output
	}

	limitUpPrice := CalcLimitUpPrice(input.PreClose, rule.LimitUpRatio)
	output.LimitUpPrice = limitUpPrice

	output.IsLimitUp = input.CurrentPrice >= limitUpPrice
	output.IsAbove5Pct = input.ChangePct >= 5.0 && !output.IsLimitUp

	currentState := input.PrevState

	if output.IsLimitUp {
		if currentState == model.LimitStateNone {
			output.IsFirstLimitUp = true
			output.CurrentState = model.LimitStateLimitUp
			output.FirstLimitTime = &input.QuoteTime
		} else if currentState == model.LimitStateLimitUp {
			output.CurrentState = model.LimitStateLimitUp
		} else if currentState == model.LimitStateOpened {
			output.CurrentState = model.LimitStateReSealed
		} else {
			output.CurrentState = model.LimitStateLimitUp
		}
	} else {
		if currentState == model.LimitStateLimitUp || currentState == model.LimitStateReSealed {
			output.CurrentState = model.LimitStateOpened
		} else {
			output.CurrentState = model.LimitStateNone
		}
	}

	return output
}
