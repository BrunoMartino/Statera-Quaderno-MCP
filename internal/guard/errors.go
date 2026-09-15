package guard

import "fmt"

const (
	AuthMissing                  = "AUTH_MISSING"
	FieldForbidden               = "FIELD_FORBIDDEN"
	RouteForbidden               = "ROUTE_FORBIDDEN"
	MethodForbidden              = "METHOD_FORBIDDEN"
	DiscountConfirmationRequired = "DISCOUNT_CONFIRMATION_REQUIRED"
	ObservabilitySecretMissing   = "OBSERVABILITY_SECRET_MISSING"
)

// Error is a typed tool/boot failure. Error() is the stable code (no secrets).
type Error struct {
	Code string
}

func NewError(code string) Error {
	return Error{Code: code}
}

func (e Error) Error() string {
	return e.Code
}

func IsCode(err error, code string) bool {
	if err == nil {
		return false
	}
	var ge Error
	if ok := asError(err, &ge); ok {
		return ge.Code == code
	}
	return err.Error() == code
}

func asError(err error, dest *Error) bool {
	ge, ok := err.(Error)
	if ok {
		*dest = ge
		return true
	}
	return false
}

func fmtStatus(status int, body []byte) error {
	if len(body) > 200 {
		body = body[:200]
	}
	return fmt.Errorf("store status %d", status)
}
