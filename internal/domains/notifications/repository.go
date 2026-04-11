package notifications

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"notificationapi.com/internal/domains/notifications/dtos"
)

type Repository struct {
	DB *pgxpool.Pool
}

func (r *Repository) GetUser(ctx context.Context, application string, user string) (dtos.Users, error) {
	query := `
		SELECT application, username
		FROM applications_users
		WHERE application = $1
			and username = $2
	`

	var app dtos.Users

	err := r.DB.QueryRow(ctx, query, application, user).
		Scan(&app.Application, &app.User)

	return app, err
}

func (r *Repository) DoesEndpointExist(ctx context.Context, endpoint string) (bool, error) {
	var exists bool

	err := r.DB.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM applications_subscriptions WHERE endpoint=$1)",
		endpoint,
	).Scan(&exists)

	return exists, err
}

func (r *Repository) Subscribe(ctx context.Context, application string, subscription dtos.RequestSubscriptionType) error {
	_, err := r.DB.Exec(ctx, `
		insert into applications_subscriptions (application, endpoint, p256dh, auth, tag, username)
		values ($1, $2, $3, $4, $5, $6)
	`, application, subscription.Endpoint, subscription.Keys.P256dh, subscription.Keys.Auth, subscription.Tag, subscription.User)
	return err
}

func (r *Repository) GetSubscriptionsByUser(ctx context.Context, application string, user string) (pgx.Rows, error) {
	rows, err := r.DB.Query(ctx,
		`SELECT application, endpoint, p256dh, auth, tag, username from applications_subscriptions where application = $1 and username = $2`,
		application, user,
	)
	return rows, err
}

func (r *Repository) GetSubscriptionsByUserAndTag(ctx context.Context, application string, user string, tag string) (pgx.Rows, error) {
	rows, err := r.DB.Query(ctx,
		`SELECT application, endpoint, p256dh, auth, tag, username from applications_subscriptions where application = $1 and username = $2 and tag = $3`,
		application, user, tag,
	)
	return rows, err
}

func (r *Repository) DeleteSubscription(ctx context.Context, application string, endpoint string) error {
	_, err := r.DB.Exec(ctx, `
		delete from applications_subscriptions where application = $1 and endpoint = $2
	`, application, endpoint)
	return err
}
