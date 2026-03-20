package migration

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/gorm"
)

type Runner struct {
	db             *gorm.DB
	migrationsPath string
}

func NewRunner(db *gorm.DB, migrationsPath string) *Runner {
	return &Runner{db: db, migrationsPath: migrationsPath}
}

func (r *Runner) Run(ctx context.Context) error {
	files, err := filepath.Glob(filepath.Join(r.migrationsPath, "*.sql"))
	if err != nil {
		return fmt.Errorf("glob migrations: %w", err)
	}

	if len(files) == 0 {
		log.Println("no migration files found")
		return nil
	}

	for _, file := range files {
		if err := r.runFile(ctx, file); err != nil {
			return fmt.Errorf("run migration %s: %w", file, err)
		}
		log.Printf("applied: %s", filepath.Base(file))
	}

	return nil
}

func (r *Runner) runFile(ctx context.Context, file string) error {
	content, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	statements := strings.Split(string(content), ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}
		if err := r.db.WithContext(ctx).Exec(stmt).Error; err != nil {
			return err
		}
	}

	return nil
}
