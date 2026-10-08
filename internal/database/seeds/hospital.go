package seeds

import (
	"context"
	"database/sql"
	"fmt"
	"log"
)

// HospitalSeed represents the hospital mockup data structure.
type HospitalSeed struct {
	HN   string
	Name string
}

// DefaultHospitals contains realistic mockup data for table hospital.
var DefaultHospitals = []HospitalSeed{
	{HN: "HOSP001", Name: "โรงพยาบาลศิริราช (Siriraj Hospital)"},
	{HN: "HOSP002", Name: "โรงพยาบาลจุฬาลงกรณ์ สภากาชาดไทย (King Chulalongkorn Memorial Hospital)"},
	{HN: "HOSP003", Name: "โรงพยาบาลรามาธิบดี (Ramathibodi Hospital)"},
	{HN: "HOSP004", Name: "โรงพยาบาลกรุงเทพ (Bangkok Hospital)"},
	{HN: "HOSP005", Name: "โรงพยาบาลบำรุงราษฎร์ (Bumrungrad International Hospital)"},
	{HN: "HOSP006", Name: "โรงพยาบาลสมิติเวช สุขุมวิท (Samitivej Sukhumvit Hospital)"},
	{HN: "HOSP007", Name: "โรงพยาบาลพระมงกุฎเกล้า (Phramongkutklao Hospital)"},
	{HN: "HOSP008", Name: "โรงพยาบาลราชวิถี (Rajavithi Hospital)"},
	{HN: "HOSP009", Name: "โรงพยาบาลธรรมศาสตร์เฉลิมพระเกียรติ (Thammasat University Hospital)"},
	{HN: "HOSP010", Name: "โรงพยาบาลมหาราชนครเชียงใหม่ (Maharaj Nakorn Chiang Mai Hospital)"},
}

// SeedHospitals inserts or updates hospital mockup records in a transaction.
// Uses ON CONFLICT (hn) DO UPDATE to ensure idempotency.
func SeedHospitals(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO hospital (hn, name)
		VALUES ($1, $2)
		ON CONFLICT (hn) DO UPDATE
		SET name = EXCLUDED.name, updated_at = CURRENT_TIMESTAMP
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, h := range DefaultHospitals {
		if _, err := stmt.ExecContext(ctx, h.HN, h.Name); err != nil {
			return fmt.Errorf("failed to seed hospital hn=%s: %w", h.HN, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit seed transaction: %w", err)
	}

	log.Printf("[INFO] Successfully seeded %d hospital records into 'hospital' table", len(DefaultHospitals))
	return nil
}
