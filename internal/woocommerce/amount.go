package woocommerce

import (
	"encoding/json"
	"strings"
)

// couponAmount accepts JSON string or number (WC uses string amounts).
type couponAmount string

func (a *couponAmount) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*a = couponAmount(s)
		return nil
	}
	*a = couponAmount(strings.TrimSpace(string(b)))
	return nil
}

func (a couponAmount) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(a))
}
