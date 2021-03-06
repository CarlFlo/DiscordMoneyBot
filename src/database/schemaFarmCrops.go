package database

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/CarlFlo/malm"
)

type FarmCrop struct {
	Model
	Name           string
	Emoji          string
	DurationToGrow time.Duration
	HarvestReward  int
}

func (FarmCrop) TableName() string {
	return "farmCrops"
}

func (fc *FarmCrop) GetAllCrops() []FarmCrop {
	var crops []FarmCrop
	DB.Find(&crops)
	return crops
}

func (fc *FarmCrop) GetCropByName(name string) bool {
	ok := DB.Raw("SELECT * FROM farmCrops WHERE name LIKE ?", name).First(&fc)
	return ok.RowsAffected > 0
}

// Outputs the duration in a pretty format
// Example: 10 days; 1 day; 16 hours; 1 hour; 20 mins
// Does not handle days with hours, or hours with minutes
// Does not handle seconds
func (fc *FarmCrop) GetDuration() string {

	var err error

	// TODO: subtract the plantedAt duration from the durationToGrow

	duration := fmt.Sprintf("%v", fc.DurationToGrow)

	split := strings.Split(duration, "h")

	if len(split) == 2 {
		// We have hours. Ignore the minutes
		hours, err := strconv.Atoi(split[0])
		if err != nil {
			malm.Error("%s", err)
			return "error"
		}
		if hours == 24 {
			return fmt.Sprintf("%d day", hours/24)
		}
		if hours > 24 {

			return fmt.Sprintf("%d days", hours/24)
		}
		if hours == 1 {
			return fmt.Sprintf("%d hour", hours)
		}

		return fmt.Sprintf("%d hours", hours)
	}

	// We dont have hours. Check if we have minutes (which we must have)

	split = strings.Split(duration, "m")

	// convert to int
	minutes, err := strconv.Atoi(split[0])
	if err != nil {
		malm.Error("%s", err)
		return "error"
	}

	return fmt.Sprintf("%d mins", minutes)
}
