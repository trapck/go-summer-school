package cmd

import (
	"wasfaty.api/pkg/env"
)

type config struct {
	env.Service
	env.HTTPServer
	env.Trace
	env.FHIR
	env.OTPClient
}
