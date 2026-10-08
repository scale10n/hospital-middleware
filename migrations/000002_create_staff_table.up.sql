CREATE TABLE IF NOT EXISTS staff (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hospital_id UUID NOT NULL,
    username VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_staff_hospital FOREIGN KEY (hospital_id) REFERENCES hospital(id) ON DELETE CASCADE,
    CONSTRAINT uq_staff_hospital_username UNIQUE (hospital_id, username)
);

CREATE INDEX IF NOT EXISTS idx_staff_hospital_id ON staff(hospital_id);
