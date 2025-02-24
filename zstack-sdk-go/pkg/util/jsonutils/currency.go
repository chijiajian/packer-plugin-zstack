// Copyright (c) HashiCorp, Inc.

package jsonutils

import (
	"fmt"
	"strings"

	"zstack.io/zstack-sdk-go/pkg/util/regutils"
)

func normalizeUSCurrency(currency string) string {
	return strings.Replace(currency, ",", "", -1)
}

func normalizeEUCurrency(currency string) string {
	commaPos := strings.IndexByte(currency, ',')
	if commaPos >= 0 {
		return fmt.Sprintf("%s.%s", strings.Replace(currency[:commaPos], ".", "", -1), currency[commaPos+1:])
	} else {
		return strings.Replace(currency, ".", "", -1)
	}
}

func normalizeCurrencyString(currency string) string {
	if regutils.MatchUSCurrency(currency) {
		return normalizeUSCurrency(currency)
	}
	if regutils.MatchEUCurrency(currency) {
		return normalizeEUCurrency(currency)
	}
	return currency
}
