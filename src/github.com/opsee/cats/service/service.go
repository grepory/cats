package service

import (
	"errors"

	"github.com/opsee/cats/checker"
	"github.com/opsee/cats/store"
)

type service struct {
	db store.Store
}

func newService(connect string) (*service, error) {
	svc := new(service)
	db, err := store.NewPostgres(connect)
	if err != nil {
		return nil, err
	}

	svc.db = db

	return svc, nil
}

type GetChecksRequest struct {
	// Checks is an array of CheckIDs for which to retrieve assertions.
	Checks []string `json:"checks"`
}

func (r *GetChecksRequest) Validate() error {
	if len(r.Checks) < 1 {
		return errors.New("Must specify check IDs to retrieve.")
	}

	return nil
}

type PutCheckRequest struct {
	Check *checker.Check `json:"check"`
}

func (r *PutCheckRequest) Validate() error {
	if r.Check == nil {
		return errors.New("No check found in request.")
	}

	return nil
}
