package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"l0/internal/config"
	"l0/internal/domain"
	"l0/internal/storage/redis"
	"l0/pkg/e"
	"log"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	pool        *pgxpool.Pool
	redisClient *redis.Redis
	logger      *slog.Logger
}

func NewPostgres(ctx context.Context, cfg *config.Config, logger *slog.Logger, redisClient *redis.Redis) (*Postgres, error) {
	connectionString := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Postgres.Host,
		cfg.Postgres.Port,
		cfg.Postgres.User,
		cfg.Postgres.Password,
		cfg.Postgres.Database,
		cfg.Postgres.SSLMode,
	)
	config, err := pgxpool.ParseConfig(connectionString)
	if err != nil {
		return nil, e.Wrap("storage.pg.NewPostgres.ParseConfig", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, e.Wrap("storage.pg.NewPostgres.NewWithConfig", err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		return nil, e.Wrap("storage.pg.NewPostgres.Ping", err)
	}

	return &Postgres{
		pool:        pool,
		redisClient: redisClient,
		logger:      logger,
	}, nil
}

func (p *Postgres) GetByID(ctx context.Context, id int) (domain.Order, error) {
	var o domain.Order
	var payment_id_fk int64
	key := fmt.Sprintf("order: %v", id)
	_, err := p.redisClient.Get(ctx, key, &o)
	if err != nil {
		log.Println("Failed get order from cache")
	} else {
		log.Printf("Order from redis cache")
		return o, nil
	}
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return domain.Order{}, e.Wrap("storage.pg.GetByUID.Begin", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, sql.ErrTxDone) {
			p.logger.Error("failed to rollback transaction", slog.String("error", err.Error()))
		}
	}()

	err = tx.QueryRow(ctx, `SELECT OrderUID, Entry, InternalSignature, payment_id_fk, Locale, CustomerID, 
	TrackNumber, DeliveryService, Shardkey, SmID FROM orders WHERE id = $1`, id).Scan(&o.OrderUID, &o.Entry,
		&o.InternalSignature, &payment_id_fk, &o.Locale, &o.CustomerID, &o.TrackNumber, &o.DeliveryService, &o.Shardkey,
		&o.SmID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Order{}, e.ErrNotFound
		}
		return o, e.Wrap("storage.pg.GetByUID.Order", err)
	}

	err = tx.QueryRow(ctx, `SELECT name, phone, zip, city, address, region, email FROM delivery 
	WHERE id = $1`, id).Scan(&o.Delivery.Name, &o.Delivery.Phone, &o.Delivery.Zip, &o.Delivery.City, &o.Delivery.Address, &o.Delivery.Region, &o.Delivery.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Order{}, e.ErrNotFound
		}
		return o, e.Wrap("storage.pg.GetByUID.Delivery", err)
	}

	err = tx.QueryRow(ctx, `SELECT Transaction, Currency, Provider, Amount, PaymentDt, Bank, DeliveryCost,
	GoodsTotal FROM payment WHERE id = $1`, payment_id_fk).Scan(&o.Payment.Transaction, &o.Payment.Currency, &o.Payment.Provider,
		&o.Payment.Amount, &o.Payment.PaymentDt, &o.Payment.Bank, &o.Payment.DeliveryCost, &o.Payment.GoodsTotal)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Order{}, e.ErrNotFound
		}
		return o, e.Wrap("storage.pg.GetByUID.Payment", err)
	}

	rowsItems, err := tx.Query(ctx, "SELECT item_id_fk FROM order_items WHERE order_id_fk = $1", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Order{}, e.ErrNotFound
		}
		return o, e.Wrap("storage.pg.GetByUID.ItemsID", err)
	}
	defer rowsItems.Close()

	var itemIDs []int64
	for rowsItems.Next() {
		var itemID int64
		if err := rowsItems.Scan(&itemID); err != nil {
			return o, e.Wrap("storage.pg.GetByUID.RowsItems.Next()", err)
		}

		itemIDs = append(itemIDs, itemID)
	}

	if err := rowsItems.Err(); err != nil {
		return domain.Order{}, e.Wrap("storage.pg.GetByUID.Rows.Err()", err)
	}

	for _, itemID := range itemIDs {
		var item domain.Items
		err = tx.QueryRow(ctx, `SELECT ChrtID, Price, Rid, Name, Sale, Size, TotalPrice, NmID, Brand 
		FROM items WHERE id = $1`, itemID).Scan(&item.ChrtID, &item.Price, &item.Rid, &item.Name, &item.Sale, &item.Size,
			&item.TotalPrice, &item.NmID, &item.Brand)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.Order{}, e.ErrNotFound
			}
			return o, e.Wrap("storage.pg.GetByUID.Items", err)
		}
		o.Items = append(o.Items, item)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Order{}, e.Wrap("storage.pg.GetByUID.Commit", err)
	}
	err = p.redisClient.Set(ctx, key, o, time.Minute*5)
	if err != nil {
		log.Printf("Failed to save in redis cache: %v", err)
	} else {
		log.Println("Order saved in redis cache")
	}
	return o, nil

}
func (p *Postgres) Create(ctx context.Context, o domain.Order) (int, error) {
	var lastInsertId int
	var itemsIds []int = []int{}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return 0, e.Wrap("storage.pg.CreateOrder.Begin", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, sql.ErrTxDone) {
			p.logger.Error("failed to rollback transaction", slog.String("error", err.Error()))
		}
	}()

	for _, item := range o.Items {
		err := tx.QueryRow(ctx, `INSERT INTO items (ChrtID, Price, Rid, Name, Sale, Size, TotalPrice, NmID, Brand)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`, item.ChrtID, item.Price, item.Rid, item.Name, item.Sale, item.Size,
			item.TotalPrice, item.NmID, item.Brand).Scan(&lastInsertId)
		if err != nil {
			return 0, e.Wrap("storage.pg.CreateOrder1", err)
		}
		itemsIds = append(itemsIds, lastInsertId)
	}

	err = tx.QueryRow(ctx, `INSERT INTO payment (Transaction, Currency, Provider, Amount, PaymentDt, Bank, DeliveryCost,
		 GoodsTotal) values ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`, o.Payment.Transaction, o.Payment.Currency, o.Payment.Provider,
		o.Payment.Amount, o.Payment.PaymentDt, o.Payment.Bank, o.Payment.DeliveryCost, o.Payment.GoodsTotal).Scan(&lastInsertId)
	if err != nil {
		return 0, e.Wrap("storage.pg.CreateOrder", err)
	}
	paymentIdFk := lastInsertId

	_, err = tx.Exec(ctx, `INSERT INTO delivery (name, phone, zip, city, address, region, email) VALUES($1, $2, $3, $4, $5, $6, $7)`,
		o.Delivery.Name, o.Delivery.Phone, o.Delivery.Zip, o.Delivery.City, o.Delivery.Address, o.Delivery.Region, o.Delivery.Email)
	if err != nil {
		return 0, e.Wrap("storage.pg.CreateOrder", err)
	}

	err = tx.QueryRow(ctx, `INSERT INTO orders (OrderUID, Entry, InternalSignature, payment_id_fk, Locale, 
		CustomerID, TrackNumber, DeliveryService, Shardkey, SmID) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`,
		o.OrderUID, o.Entry, o.InternalSignature, paymentIdFk, o.Locale, o.CustomerID, o.TrackNumber, o.DeliveryService,
		o.Shardkey, o.SmID).Scan(&lastInsertId)
	if err != nil {
		return 0, e.Wrap("storage.pg.CreateOrder", err)
	}
	orderIdFk := lastInsertId

	for _, itemId := range itemsIds {
		_, err := tx.Exec(ctx, `INSERT INTO order_items (order_id_fk, item_id_fk) values ($1, $2)`,
			orderIdFk, itemId)
		if err != nil {
			return 0, e.Wrap("storage.pg.CreateOrder", err)
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return 0, e.Wrap("storage.pg.CreateOrder.Commit", err)
	}
	log.Println("Order successfull added in db")

	return orderIdFk, nil

}

func (p *Postgres) CloseConnection() {
	p.pool.Close()
	stat := p.pool.Stat()
	if stat.AcquiredConns() > 0 {
		p.logger.Warn("postgres connections not fully closed after Close()", slog.Any("acquired connections", stat.AcquiredConns()))

	}
}
