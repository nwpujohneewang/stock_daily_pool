package limiter

import (
	"stock/model/dal_model"
)

func CalcLimitUpPrice(preClose float64, limitUpRatio float64) float64 {
	if preClose <= 0 {
		return 0
	}
	return float64(int(preClose*(1+limitUpRatio)*100)) / 100
}

func DetectBoard(symbol string) dal_model.BoardCode {
	switch {
	case len(symbol) >= 3 && symbol[:3] == "688":
		return dal_model.BoardSTAR
	case len(symbol) >= 3 && (symbol[:3] == "300" || symbol[:3] == "301"):
		return dal_model.BoardGEM
	case len(symbol) >= 1 && (symbol[0] == '8' || symbol[0] == '4' || symbol[0] == '9'):
		return dal_model.BoardBSE
	default:
		return dal_model.BoardMain
	}
}

func DetectLimitUp(input dal_model.DetectInput, rule *dal_model.BoardRule) dal_model.DetectOutput {
	output := dal_model.DetectOutput{
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
		if currentState == dal_model.LimitStateNone {
			output.IsFirstLimitUp = true
			output.CurrentState = dal_model.LimitStateLimitUp
			output.FirstLimitTime = &input.QuoteTime
		} else if currentState == dal_model.LimitStateLimitUp {
			output.CurrentState = dal_model.LimitStateLimitUp
		} else if currentState == dal_model.LimitStateOpened {
			output.CurrentState = dal_model.LimitStateReSealed
		} else {
			output.CurrentState = dal_model.LimitStateLimitUp
		}
	} else {
		if currentState == dal_model.LimitStateLimitUp || currentState == dal_model.LimitStateReSealed {
			output.CurrentState = dal_model.LimitStateOpened
		} else {
			output.CurrentState = dal_model.LimitStateNone
		}
	}

	return output
}
