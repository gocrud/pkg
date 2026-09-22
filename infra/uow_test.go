package infra

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type transactionPool struct {
	gorm.ConnPool
	begins   int
	beginErr error
	beginCtx context.Context
	tx       *transactionStub
}

func (pool *transactionPool) BeginTx(ctx context.Context, _ *sql.TxOptions) (gorm.ConnPool, error) {
	pool.begins++
	pool.beginCtx = ctx
	return pool.tx, pool.beginErr
}

type transactionStub struct {
	gorm.ConnPool
	commits   int
	rollbacks int
	commitErr error
}

func (tx *transactionStub) Commit() error {
	tx.commits++
	return tx.commitErr
}

func (tx *transactionStub) Rollback() error {
	tx.rollbacks++
	return nil
}

func newTransactionDB(test *testing.T) (*gorm.DB, *transactionPool) {
	test.Helper()
	pool := &transactionPool{tx: &transactionStub{}}
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: pool}), &gorm.Config{
		DisableAutomaticPing: true,
	})
	if err != nil {
		test.Fatal(err)
	}
	return db, pool
}

func TestStoreContext(test *testing.T) {
	db, pool := newTransactionDB(test)
	store := NewStore(db.Where("status = ?", 1))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	query := store.WithContext(ctx).Table("products")
	if query.Statement.ConnPool != pool || query.Statement.Context != ctx {
		test.Fatal("expected base connection and caller context")
	}
	if _, exists := query.Statement.Clauses["WHERE"]; exists {
		test.Fatal("store inherited query conditions")
	}
	for _, plainCtx := range []context.Context{ctx, context.WithValue(ctx, txKey{}, (*gorm.DB)(nil))} {
		plainQuery := store.ForUpdate(plainCtx).Table("products")
		if plainQuery.Error != nil || plainQuery.Statement.ConnPool != pool || plainQuery.Statement.Context != plainCtx {
			test.Fatal("expected ordinary query without a transaction")
		}
		if _, exists := plainQuery.Statement.Clauses["FOR"]; exists {
			test.Fatal("ordinary query must not include a locking clause")
		}
	}
	if db.Error != nil || store.WithContext(ctx).Error != nil || pool.begins != 0 {
		test.Fatal("store mutated the base DB or started a transaction")
	}
}

func TestUnitOfWorkLifecycle(test *testing.T) {
	failure := errors.New("test failure")
	for _, scenario := range []struct {
		name      string
		actionErr error
		beginErr  error
		commitErr error
		panics    bool
		commits   int
		rollbacks int
	}{
		{name: "commit", commits: 1},
		{name: "rollback", actionErr: failure, rollbacks: 1},
		{name: "panic", panics: true, rollbacks: 1},
		{name: "begin failure", beginErr: failure},
		{name: "commit failure", commitErr: failure, commits: 1, rollbacks: 1},
	} {
		test.Run(scenario.name, func(test *testing.T) {
			db, pool := newTransactionDB(test)
			pool.beginErr = scenario.beginErr
			pool.tx.commitErr = scenario.commitErr
			unit := NewUnitOfWork(db)
			ctx := context.Background()
			called := false
			var result error
			var recovered any
			func() {
				defer func() { recovered = recover() }()
				result = unit.Execute(ctx, func(txCtx context.Context) error {
					called = true
					query := NewStore(db).WithContext(txCtx).Table("products")
					if query.Statement.ConnPool != pool.tx || query.Statement.Context != txCtx {
						test.Fatal("expected transaction connection and transaction context")
					}
					if ctx.Value(txKey{}) != nil {
						test.Fatal("original context was modified")
					}
					if scenario.panics {
						panic(failure)
					}
					return scenario.actionErr
				})
			}()
			if scenario.panics && recovered != failure || !scenario.panics && recovered != nil {
				test.Fatalf("unexpected panic: %v", recovered)
			}
			if !scenario.panics {
				wantErr := errors.Join(scenario.actionErr, scenario.beginErr, scenario.commitErr)
				if wantErr == nil && result != nil || wantErr != nil && !errors.Is(result, failure) {
					test.Fatalf("unexpected result: %v", result)
				}
			}
			if called != (scenario.beginErr == nil) || pool.begins != 1 || pool.beginCtx != ctx {
				test.Fatal("unexpected transaction start or callback invocation")
			}
			if pool.tx.commits != scenario.commits || pool.tx.rollbacks != scenario.rollbacks {
				test.Fatalf("commits=%d rollbacks=%d", pool.tx.commits, pool.tx.rollbacks)
			}
		})
	}
}

func TestUnitOfWorkReusesContextTransaction(test *testing.T) {
	db, pool := newTransactionDB(test)
	otherDB, otherPool := newTransactionDB(test)
	failure := errors.New("nested failure")
	err := NewUnitOfWork(db).Execute(context.Background(), func(txCtx context.Context) error {
		childCtx, cancel := context.WithCancel(txCtx)
		defer cancel()
		return NewUnitOfWork(otherDB).Execute(childCtx, func(nestedCtx context.Context) error {
			if nestedCtx != childCtx {
				test.Fatal("nested execution replaced the caller context")
			}
			query := NewStore(otherDB).ForUpdate(nestedCtx)
			if query.Error != nil || query.Statement.ConnPool != pool.tx || query.Statement.Context != childCtx {
				test.Fatal("store did not use the context transaction")
			}
			locking, ok := query.Statement.Clauses["FOR"].Expression.(clause.Locking)
			if !ok || locking.Strength != "UPDATE" {
				test.Fatal("missing FOR UPDATE clause")
			}
			if pool.tx.commits != 0 || pool.tx.rollbacks != 0 {
				test.Fatal("nested execution finalized the transaction")
			}
			return failure
		})
	})
	if !errors.Is(err, failure) || pool.begins != 1 || otherPool.begins != 0 || pool.tx.commits != 0 || pool.tx.rollbacks != 1 {
		test.Fatal("nested error did not roll back the outer transaction exactly once")
	}
}
