package main

import (
	"os"
	"strconv"
)

type config struct {
	path               string
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
	if env := os.Getenv("OOHHC_FILESYS_PATH"); env != "" {
		cfg.path = env
	}
	if cfg.path == "" {
		cfg.path = "/var/lib/oohhc-filesys"
	}
	if env := os.Getenv("OOHHC_FILESYS_PORT"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.port = val
		}
	}
	if cfg.port == 0 {
		cfg.port = 8448
	}
	if env := os.Getenv("OOHHC_FILESYS_INSECURE_SKIP_VERIFY"); env == "true" {
		cfg.insecureSkipVerify = true
	}
	if env := os.Getenv("OOHHC_FILESYS_SKIP_MUTUAL_TLS"); env == "true" {
		cfg.skipMutualTLS = true
	}
	// cfg.oortGroupSyndicate == "" means default SRV resolution.
	if env := os.Getenv("OOHHC_FILESYS_OORT_GROUP_SYNDICATE"); env != "" {
		cfg.oortGroupSyndicate = env
	}
	return cfg
}
