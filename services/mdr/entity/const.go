package entity

import (
	"wasfaty.api/pkg/fhir/model"
)

const (
	NationalIDAgeFrom                               = 15
	MaxPatientsWithSamePhone                        = 10
	TaskBusinessStatusPatientCreated                = "Patient Created"
	TaskBusinessStatusOTPCodeSent                   = "OTP code sent"
	TaskBusinessStatusPatientIdentityUpdated        = "Confirm Updating Identity & save Parameters"
	TaskBusinessStatusConfirmPatientIdentityUpdated = "Patient Identity Updated"
)

func CreatePatientIdentPriority() []string {
	return []string{
		model.IdentNationalID,
		model.IdentPermanentResidentCardNumber,
		model.IdentBorderNumber,
		model.IdentDisplacedPerson,
		model.IdentGulfCooperationCouncilNumber,
		model.IdentJurisdictionalHealthNumber,
		model.IdentVisa,
		model.IdentPassport,
		model.IdentCitizenshipCard,
	}
}

func UpdatePatientURLList() []string {
	return []string{
		model.StructureDefinitionReligion,
		model.StructureDefinitionImportance,
		model.StructureDefinitionOccupation,
	}
}

func UpdatePatientParametersList() []string {
	return []string{
		"meritalStatus",
		"communication",
		"contact",
		"religion",
		"importance",
		"occupation",
		"citizenship",
		"nationality",
	}
}

func UpdateEmailPatientParametersList() []string {
	return []string{
		"telecom",
	}
}

func CreatePatientForbiddenParams() []string {
	return []string{
		"deceasedBoolean",
		"deceasedDateTime",
		"multipleBirth",
		"photo",
		"generalPractitioner",
		"managingOrganization",
		"link",
	}
}

func IdentifierCodeForSANationality() []string {
	return []string{
		model.IdentNationalID,
		model.IdentDisplacedPerson,
		model.IdentCitizenshipCard,
		model.IdentJurisdictionalHealthNumber,
	}
}

func IdentifierCodeForOtherNationality() []string {
	return []string{
		model.IdentPermanentResidentCardNumber,
		model.IdentBorderNumber,
		model.IdentDisplacedPerson,
		model.IdentJurisdictionalHealthNumber,
		model.IdentGulfCooperationCouncilNumber,
		model.IdentVisa,
		model.IdentPassport,
	}
}
