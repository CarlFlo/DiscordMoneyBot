package database

import (
	"fmt"

	"github.com/CarlFlo/DiscordMoneyBot/src/config"
	"github.com/CarlFlo/malm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

const resetDatabaseOnStart = true

func Connect() {
	if err := connectToDB(); err != nil {
		malm.Fatal("Database initialization error: %s", err)
		return
	}
	malm.Info("Connected to database")
}

func connectToDB() error {

	var err error
	DB, err = gorm.Open(sqlite.Open(config.CONFIG.Database.FileName), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return err
	}

	var modelList = []interface{}{
		&User{},
		&Work{},
		&Daily{},
		&Farm{},
		&FarmPlot{},
		&FarmCrop{},
		&Notify{},
		&Debug{},
	}

	if resetDatabaseOnStart {

		malm.Info("Resetting database...")

		type tmp interface {
			TableName() string
		}

		for _, e := range modelList {
			table := e.(tmp).TableName()
			DB.Exec(fmt.Sprintf("DROP TABLE %s", table))
		}
		defer PopulateDatabase() // Populates the database with the default values
	}

	// Remeber to add new tables to the tableList and not just here!
	return DB.AutoMigrate(modelList...)
}
