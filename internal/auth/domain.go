// Design Pattern: STATE
package auth

import (
	errors "GuiFernandess7/gothenticate/internal/infra/errors"
)

const maxLoginRetries = 5

type AccountStatus string

const (
	StatusLabelActive  AccountStatus = "active"
	StatusLabelLocked  AccountStatus = "locked"
	StatusLabelBlocked AccountStatus = "blocked"
)

type AccountStateInterface interface {
	Label() AccountStatus
	CanAttemptLogin(account *AccountState) error
	RecordFailedLogin(account *AccountState) error
	RecordSuccessfulLogin(account *AccountState) error
	Block(account *AccountState) error
	Unblock(account *AccountState) error
}

type AccountState struct {
	UserID  int64
	status  AccountStateInterface
	Retries int
}

func NewAccountState(userID int64, label AccountStatus, retries int) *AccountState {
	return &AccountState{
		UserID:  userID,
		status:  stateFromLabel(label),
		Retries: retries,
	}
}

func stateFromLabel(label AccountStatus) AccountStateInterface {
	switch label {
	case StatusLabelLocked:
		return &StatusLocked{}
	case StatusLabelBlocked:
		return &StatusBlocked{}
	default:
		return &StatusActive{}
	}
}

func (a *AccountState) Label() AccountStatus {
	return a.status.Label()
}

func (a *AccountState) CanAttemptLogin() error {
	return a.status.CanAttemptLogin(a)
}

func (a *AccountState) RecordFailedLogin() error {
	return a.status.RecordFailedLogin(a)
}

func (a *AccountState) RecordSuccessfulLogin() error {
	return a.status.RecordSuccessfulLogin(a)
}

func (a *AccountState) Block() error {
	return a.status.Block(a)
}

func (a *AccountState) Unblock() error {
	return a.status.Unblock(a)
}

// ========== ACTIVE ==========

type StatusActive struct{}

func (s *StatusActive) Label() AccountStatus { return StatusLabelActive }

func (s *StatusActive) CanAttemptLogin(account *AccountState) error {
	return nil
}

func (s *StatusActive) RecordFailedLogin(account *AccountState) error {
	account.Retries++
	if account.Retries >= maxLoginRetries {
		account.status = &StatusLocked{}
	}
	return nil
}

func (s *StatusActive) RecordSuccessfulLogin(account *AccountState) error {
	account.Retries = 0
	return nil
}

func (s *StatusActive) Block(account *AccountState) error {
	account.status = &StatusBlocked{}
	return nil
}

func (s *StatusActive) Unblock(account *AccountState) error {
	return errors.ErrNotFound
}

// ========== LOCKED (too many failed login attempts) ==========

type StatusLocked struct{}

func (s *StatusLocked) Label() AccountStatus { return StatusLabelLocked }

func (s *StatusLocked) CanAttemptLogin(account *AccountState) error {
	return errors.ErrAccountLocked
}

func (s *StatusLocked) RecordFailedLogin(account *AccountState) error {
	return errors.ErrAccountLocked
}

func (s *StatusLocked) RecordSuccessfulLogin(account *AccountState) error {
	return errors.ErrAccountLocked
}

func (s *StatusLocked) Block(account *AccountState) error {
	account.status = &StatusBlocked{}
	return nil
}

func (s *StatusLocked) Unblock(account *AccountState) error {
	account.Retries = 0
	account.status = &StatusActive{}
	return nil
}

// ========== BLOCKED (administrative) ==========

type StatusBlocked struct{}

func (s *StatusBlocked) Label() AccountStatus { return StatusLabelBlocked }

func (s *StatusBlocked) CanAttemptLogin(account *AccountState) error {
	return errors.ErrBlocked
}

func (s *StatusBlocked) RecordFailedLogin(account *AccountState) error {
	return errors.ErrBlocked
}

func (s *StatusBlocked) RecordSuccessfulLogin(account *AccountState) error {
	return errors.ErrBlocked
}

func (s *StatusBlocked) Block(account *AccountState) error {
	return errors.ErrBlocked
}

func (s *StatusBlocked) Unblock(account *AccountState) error {
	account.Retries = 0
	account.status = &StatusActive{}
	return nil
}
