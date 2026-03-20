package converter

import (
	"fmt"
	"strings"
)

func JiuyanToTushare(code string) string {
	if len(code) < 3 {
		return code
	}
	prefix := strings.ToLower(code[:2])
	number := code[2:]
	return number + "." + strings.ToUpper(prefix)
}

func TushareToJiuyan(code string) string {
	parts := strings.Split(code, ".")
	if len(parts) != 2 {
		return code
	}
	return strings.ToLower(parts[1]) + parts[0]
}

func DetectMarket(symbol string) (string, error) {
	if len(symbol) < 1 {
		return "", fmt.Errorf("invalid symbol")
	}
	switch symbol[0] {
	case '6':
		return "SH", nil
	case '0', '3':
		return "SZ", nil
	case '8', '4':
		return "BJ", nil
	default:
		return "", fmt.Errorf("unknown market for symbol: %s", symbol)
	}
}

func SymbolToTsCode(symbol string) (string, error) {
	market, err := DetectMarket(symbol)
	if err != nil {
		return "", err
	}
	return symbol + "." + market, nil
}
