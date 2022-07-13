package entity

import (
	"time"

	fhirModel "wasfaty.api/pkg/fhir/model"
)

type SearchTaskParams struct {
	Telecom    *SearchTaskByTelecomParams
	Identifier *SearchTaskByIdentifierParams
}

type SearchTaskByTelecomParams struct {
	Value     string
	Use       string
	System    string
	BirthDate string
	// new feature 1
	// new feature 2
	// new bug bix 22
	// new feature 3
	// new feature 4
	// new feature 5
}

type SearchTaskByIdentifierParams struct {
	Value string
	Type  string
}

type SearchPatientParams struct {
	Phone      *SearchPatientByPhoneParams
	Identifier *SearchPatientByIdentifierParams
}

type SearchPatientByPhoneParams struct {
	Phone     string
	BirthDate string
}

type SearchPatientByIdentifierParams struct {
	Value string
	Type  string
}

type UpdatePatientIdentityParameters struct {
	ConfirmationMethod string
	Patient            *fhirModel.Patient
}

type ExtDocRegistrySearchResult struct {
	IsValid bool
}

type ConfirmRequestParameters struct {
	OTPCode string
	TaskID  fhirModel.ID
}

type OTP struct {
	Code             string    `json:"code,omitempty"`
	ExpiresAt        time.Time `json:"expiresAt,omitempty"`
	AttemptsCount    int       `json:"attemptsCount"`
	MaxAttemptsCount int       `json:"maxAttemptsCount"`
	Value            string    `json:"value,omitempty"`
}

type ValidateOTPParams struct {
	Code      string `json:"code,omitempty"`
	Value     string `json:"value,omitempty"`
	ProcessID string `json:"processID,omitempty"`
}
