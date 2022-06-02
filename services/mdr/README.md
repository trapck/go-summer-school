## Environment

| VAR_NAME                          | Default Value | Description                                                        |
|-----------------------------------|---------------|--------------------------------------------------------------------|
| SERVICE_NAME                      | required      | Service name                                                       |
| SERVICE_LOG_LEVEL                 | info          | Service log level (trace, debug, info, warn, error)                |
| HTTP_SERVER_HOST                  |               | HTTP server host                                                   |
| HTTP_SERVER_PORT                  | required      | HTTP server port                                                   |
| HTTP_SERVER_READ_TIMEOUT_SEC      | 60            | HTTP server read timeout                                           |
| HTTP_SERVER_WRITE_TIMEOUT_SEC     | 30            | HTTP server write timeout                                          |
| HTTP_SERVER_IDLE_TIMEOUT_SEC      | 0             | HTTP server idle timeout                                           |
| HTTP_SERVER_CLOSE_ON_SHUTDOWN     | true          | Adds a `Connection: close` header when the server is shutting down |
| HTTP_SERVER_LOG_RESPONSE          | false         | Log HTTP responses                                                 |
| TRACE_ENVIRONMENT                 | development   | Environment name for tracing                                       |
| TRACE_USE_AGENT                   | false         | Use Jaeger Agent for tracing                                       |
| TRACE_URL                         | default:""    | URL for tracing                                                    |
| TRACE_AGENT_HOST                  | localhost     | Jaeger Agent Host for tracing                                      |
| TRACE_AGENT_PORT                  | 6831          | Jaeger Agent Port for tracing                                      |
| TRACE_AGENT_RECONNECTION_INTERVAL | 30            | Jaeger Agent Reconnection Interval seconds for tracing             |
| FHIR_SERVER_API_URL               | required      | FHIR API host                                                      |
| FHIR_SERVER_SEARCH_URL            | required      | FHIR Search API host                                               |
| FHIR_SERVER_API_CONSUMER          | required      | FHIR API consumer                                                  |
| FHIR_SERVER_API_REQUEST_TIMEOUT   | 30s           | FHIR API request timeout                                           |
| OTP_SERVICE_HOST                  | required      | OTP service host                                                   |
| OTP_SERVICE_REQUEST_TIMEOUT       | 30s           | OTP service request timeout                                        |