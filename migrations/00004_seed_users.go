package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	goose.AddMigration(Up00004, Down00004)
}

func Up00004(tx *sql.Tx) error {
	users := []struct {
		username string
		name     string
		password string
	}{
		{"agungdh", "Agung Dwi", "ihikihik"},
		{"surimbim", "Surimbim", "dududuw"},
		{"rakawik", "Rakawik", "dodo"},
	}

	for _, u := range users {
		hash, err := bcrypt.GenerateFromPassword([]byte(u.password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		_, err = tx.Exec("INSERT INTO users (username, password, name) VALUES (?, ?, ?)", u.username, string(hash), u.name)
		if err != nil {
			return err
		}
	}

	return nil
}

func Down00004(tx *sql.Tx) error {
	_, err := tx.Exec("DELETE FROM users WHERE username IN (?, ?, ?)", "agungdh", "surimbim", "rakawik")
	return err
}
