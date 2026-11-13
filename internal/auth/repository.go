// auth/repository.go

package auth

import (
	"context"
	"database/sql"
)

type UserRepositoryInterface interface {
    FindByEmail(ctx context.Context, email string) (*User, error)
    Create(ctx context.Context, user *User) error
    UpdateAccountState(ctx context.Context, userID int64, status AccountStatus, retries int) error
}

type UserRepository struct {
    DB *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
    return &UserRepository{DB: db}
}

func (r *UserRepository) FindByEmail(
    ctx context.Context,
    email string,
) (*User, error) {

    user := &User{}

    err := r.DB.QueryRowContext(
        ctx,
        `SELECT id, name, email, password, status, retries
         FROM users WHERE email = ?`,
        email,
    ).Scan(
        &user.ID,
        &user.Name,
        &user.Email,
        &user.Password,
        &user.Status,
        &user.Retries,
    )

    if err != nil {
        return nil, err
    }

    return user, nil
}

func (r *UserRepository) Create(
    ctx context.Context,
    user *User,
) error {

    result, err := r.DB.ExecContext(
        ctx,
        `INSERT INTO users (name, email, password)
         VALUES (?, ?, ?)`,
        user.Name,
        user.Email,
        user.Password,
    )

    if err != nil {
        return err
    }

    user.ID, err = result.LastInsertId()
    return err
}

func (r *UserRepository) UpdateAccountState(
    ctx context.Context,
    userID int64,
    status AccountStatus,
    retries int,
) error {

    _, err := r.DB.ExecContext(
        ctx,
        `UPDATE users SET status = ?, retries = ? WHERE id = ?`,
        string(status),
        retries,
        userID,
    )

    return err
}