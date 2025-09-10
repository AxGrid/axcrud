package axcrud

import (
	"context"
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
	"gorm.io/driver/sqlite" // Sqlite driver based on CGO
	"gorm.io/gorm"
)

var ctx context.Context

type TestUser struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"index"`
	Email     string `gorm:"uniqueIndex"`
	Role      string `gorm:"index"`
	Age       int    `gorm:"index"`
	UserID    uint   `gorm:"index"` // tenant scope пример
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func TestGormRepo_Create(t *testing.T) {
	db := ctx.Value("db").(*gorm.DB)
	cfg := RepoConfig{
		AllowedFilterOps: map[string]FieldSet{},
	}
	repo := NewGormRepo[TestUser, uint](db, cfg)
	user := TestUser{
		Name:  "John Doe",
		Email: "jd@example.com",
		Role:  "admin",
		Age:   30,
	}
	if err := repo.Create(ctx, &user); err != nil {
		t.Fatal(err)
	}
	if user.ID == 0 {
		t.Fatal("expected user ID to be set")
	}
	t.Logf("Created user: %+v", user)
	//Delete all
	count, err := repo.DeleteMany(ctx, []uint{user.ID})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, int64(1), count)
}

func TestGormRepo_ContextAnsScope(t *testing.T) {
	cfg := RepoConfig{
		AllowedFilterOps: map[string]FieldSet{},
		Scopes: []func(*gorm.DB) *gorm.DB{
			func(db *gorm.DB) *gorm.DB {
				userId := db.Statement.Context.Value("user_id").(uint)

				return db.Where("user_id = ?", userId)
			}, // ACL/tenant
		},
	}
	db := ctx.Value("db").(*gorm.DB)
	inCtx := context.WithValue(ctx, "user_id", uint(1))
	repo := NewGormRepo[TestUser, uint](db, cfg)
	// Create user with UserID=1
	user1 := TestUser{
		Name:   "User One",
		Email:  "uo@example.com",
		Role:   "user",
		Age:    25,
		UserID: 1,
	}
	if err := repo.Create(inCtx, &user1); err != nil {
		t.Fatal(err)
	}
	// Create user with UserID=2
	user2 := TestUser{
		Name:   "User Two",
		Email:  "ut@example.com",
		Role:   "user",
		Age:    28,
		UserID: 2,
	}
	if err := repo.Create(inCtx, &user2); err != nil {
		t.Fatal(err)
	}

	// Try to get user1 with context user_id=1
	got, err := repo.GetOne(inCtx, user1.ID)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, got.Email, user1.Email)
	// Try to get user2 with context user_id=1 - should fail
	_, err = repo.GetOne(inCtx, user2.ID)
	if err == nil {
		t.Fatal("expected error when accessing user with different UserID")
	}

	// Try delete user2
	_ = repo.Delete(inCtx, user2.ID)

	var usersCount int64
	if err = db.Model(&TestUser{}).Count(&usersCount).Error; err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, int64(2), usersCount) // both users should exist

	//// Cleanup
	//_, _ = repo.DeleteMany(ctx, []uint{user1.ID, user2.ID})
	if err = db.Delete(&user1).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Delete(&user2).Error; err != nil {
		t.Fatal(err)
	}

}

func TestMain(m *testing.M) {
	db, err := setupTestDB()
	ctx = context.WithValue(context.Background(), "db", db)
	if err != nil {
		panic(err)
	}
	m.Run()
}

func setupTestDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err = db.AutoMigrate(&TestUser{}); err != nil {
		return nil, err
	}
	return db, nil
}
