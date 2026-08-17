package config

import "net/http"

// Config is the immutable product of ConfigFactory for one MCP process (one store).
type Config struct {
	StoreID         string
	Environment     string
	BaseURL         string
	UserLogin       string
	AllowedStatuses []string
	Auth            Auth
}

// Auth is the HTTP identity created by the factory (Application Password or WC consumer).
type Auth interface {
	Apply(req *http.Request)
}

type applicationPasswordAuth struct {
	user     string
	password string
}

func (a applicationPasswordAuth) Apply(req *http.Request) {
	req.SetBasicAuth(a.user, a.password)
}

type wooCommerceConsumerAuth struct {
	key    string
	secret string
}

func (a wooCommerceConsumerAuth) Apply(req *http.Request) {
	req.SetBasicAuth(a.key, a.secret)
}
