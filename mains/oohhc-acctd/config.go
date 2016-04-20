package main

import (
	"os"
	"strconv"
)

type config struct {
	path               string
	superUserKey       string
	port               int
	insecureSkipVerify bool
	skipMutualTLS      bool
	//Group Store client values
	oortGroupSyndicate string
}

func resolveConfig(c *config) *config {
	cfg := &config{}
	if c != nil {
		*cfg = *c
	}
	if env := os.Getenv("OOHHC_ACCT_PATH"); env != "" {
		cfg.path = env
	}
	if cfg.path == "" {
		cfg.path = "/var/lib/oohhc-acct"
	}
	if env := os.Getenv("OOHHC_ACCT_SUPERUSER_KEY"); env != "" {
		cfg.superUserKey = env
	}
	if cfg.superUserKey == "" {
		cfg.superUserKey = "123456789abcdef"
	}

	if env := os.Getenv("OOHHC_ACCT_PORT"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.port = val
		}
	}
	if cfg.port == 0 {
		cfg.port = 8449
	}
	if env := os.Getenv("OOHHC_ACCT_INSECURE_SKIP_VERIFY"); env == "true" {
		cfg.insecureSkipVerify = true
	}
	if env := os.Getenv("OOHHC_ACCT_SKIP_MUTUAL_TLS"); env == "true" {
		cfg.skipMutualTLS = true
	}
	// cfg.oortGroupSyndicate == "" means default SRV resolution.
	if env := os.Getenv("OOHHC_ACCT_OORT_GROUP_SYNDICATE"); env != "" {
		cfg.oortGroupSyndicate = env
	}
	return cfg
}
