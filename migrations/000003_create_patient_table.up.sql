DO $$ BEGIN
    CREATE TYPE gender_type AS ENUM ('M', 'F');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE IF NOT EXISTS patient (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hospital_id UUID NOT NULL,
    patient_hn VARCHAR(50) NOT NULL,
    national_id VARCHAR(20),
    passport_id VARCHAR(50),
    first_name_th VARCHAR(100),
    middle_name_th VARCHAR(100),
    last_name_th VARCHAR(100),
    first_name_en VARCHAR(100),
    middle_name_en VARCHAR(100),
    last_name_en VARCHAR(100),
    date_of_birth DATE NOT NULL,
    phone_number VARCHAR(20),
    email VARCHAR(255),
    gender gender_type NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Foreign Keys
    CONSTRAINT fk_patient_hospital FOREIGN KEY (hospital_id) REFERENCES hospital(id) ON DELETE CASCADE,

    -- Multi-tenant Unique Constraints (scoped to hospital_id)
    CONSTRAINT uq_patient_hospital_hn UNIQUE (hospital_id, patient_hn),
    CONSTRAINT uq_patient_hospital_national_id UNIQUE (hospital_id, national_id),
    CONSTRAINT uq_patient_hospital_passport_id UNIQUE (hospital_id, passport_id),

    -- Validation (At least one identification must be provided)
    CONSTRAINT chk_patient_identity CHECK (national_id IS NOT NULL OR passport_id IS NOT NULL)
);

-- Search Indexes for /patient/search API optimization (covering first name, last name, and combinations)
CREATE INDEX IF NOT EXISTS idx_patient_hospital_name_en ON patient(hospital_id, first_name_en, last_name_en);
CREATE INDEX IF NOT EXISTS idx_patient_hospital_last_name_en ON patient(hospital_id, last_name_en);
CREATE INDEX IF NOT EXISTS idx_patient_hospital_name_th ON patient(hospital_id, first_name_th, last_name_th);
CREATE INDEX IF NOT EXISTS idx_patient_hospital_last_name_th ON patient(hospital_id, last_name_th);

-- Auto-update updated_at timestamp trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_patient_updated_at ON patient;
CREATE TRIGGER trg_patient_updated_at
BEFORE UPDATE ON patient
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();
