package models

import "time"

// Gender represents the patient's gender enum ('M' or 'F').
type Gender string

const (
	GenderMale   Gender = "M"
	GenderFemale Gender = "F"
)

// Patient represents the patient entity in the database.
type Patient struct {
	ID           string    `json:"id"`
	HospitalID   string    `json:"hospital_id"`
	PatientHN    string    `json:"patient_hn"`
	NationalID   string    `json:"national_id"`
	PassportID   string    `json:"passport_id"`
	FirstNameTH  string    `json:"first_name_th"`
	MiddleNameTH string    `json:"middle_name_th"`
	LastNameTH   string    `json:"last_name_th"`
	FirstNameEN  string    `json:"first_name_en"`
	MiddleNameEN string    `json:"middle_name_en"`
	LastNameEN   string    `json:"last_name_en"`
	DateOfBirth  time.Time `json:"date_of_birth"`
	PhoneNumber  string    `json:"phone_number"`
	Email        string    `json:"email"`
	Gender       Gender    `json:"gender"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// PatientWithHospital represents a patient record with hospital details joined.
type PatientWithHospital struct {
	Patient
	HospitalHN   string `json:"hospital_hn"`
	HospitalName string `json:"hospital_name"`
}
