package http

import (
	"encoding/json"

	"wasfaty.api/pkg/env"
	"wasfaty.api/pkg/middleware/headers"

	"wasfaty.api/pkg/http/fiber"

	gofiber "github.com/gofiber/fiber/v2"
)

func NewServer(httpCfg env.HTTPServer, srvCfg env.Service, traceCfg *env.Trace, uc UseCase) *fiber.Server {
	s := fiber.NewServer(&fiber.ServerConfig{
		Service: srvCfg,
		Server:  httpCfg,
		Trace:   traceCfg,
		// use standard json marshaling because native fiber marshaller
		// catches panic with complex fhir structs
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,
	}).WithDefaultKit()

	h := newHandler(uc)

	s.Fiber().Use(headers.ValidateJSONContentType(gofiber.MethodGet, gofiber.MethodDelete))

	s.Fiber().Post("/Patient/$create-request", h.createPatient)
	s.Fiber().Post("/Patient/$confirm-request", h.confirmCreatePatient)
	s.Fiber().Post("/Patient/:id/$update", h.updatePatient)
	s.Fiber().Post("/Patient/:id/$update-email", h.updatePatientEmail)
	s.Fiber().Post("/Patient/:id/$update-identity", h.updatePatientIdentity)
	s.Fiber().Post("/Patient/:id/$confirm-identity", h.confirmUpdatePatientIdentity)

	return s
}
