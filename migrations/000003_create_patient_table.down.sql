DROP TRIGGER IF EXISTS trg_patient_updated_at ON patient;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP TABLE IF EXISTS patient;
DROP TYPE IF EXISTS gender_type;
