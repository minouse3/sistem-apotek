package penerimaan

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestDistributorWriteErrorRecognizesOnlyPBFNameConstraint(t *testing.T) {
	nameConflict := &pgconn.PgError{Code: "23505", ConstraintName: "pbf_nama_pbf_key"}
	if !errors.Is(distributorWriteError(nameConflict), ErrNameTaken) {
		t.Fatal("duplicate PBF name was not mapped to ErrNameTaken")
	}

	otherConflict := &pgconn.PgError{Code: "23505", ConstraintName: "another_unique_key"}
	if errors.Is(distributorWriteError(otherConflict), ErrNameTaken) {
		t.Fatal("unrelated unique constraint was mapped to ErrNameTaken")
	}
}
