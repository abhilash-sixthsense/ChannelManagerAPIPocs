package utils

import (
	"context"
	"log"
	"time"

	"channelmanager/models"

	"gorm.io/gorm"
)

// SeedDatabase seeds initial data for development/testing
func SeedDatabase(db *gorm.DB) error {
	ctx := context.Background()

	// Check if data already exists
	var count int64
	if err := db.Model(&models.Amenity{}).Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		log.Println("Database already seeded, skipping seed")
		return nil
	}

	log.Println("Starting database seed")

	// Create amenities
	amenities := []models.Amenity{
		{Name: "Air Conditioning", Category: "comfort", Icon: "ac"},
		{Name: "WiFi", Category: "entertainment", Icon: "wifi"},
		{Name: "Swimming Pool", Category: "entertainment", Icon: "pool"},
		{Name: "Kitchen", Category: "kitchen", Icon: "kitchen"},
		{Name: "Washing Machine", Category: "comfort", Icon: "washing_machine"},
		{Name: "Dishwasher", Category: "kitchen", Icon: "dishwasher"},
		{Name: "TV", Category: "entertainment", Icon: "tv"},
		{Name: "Heating", Category: "comfort", Icon: "heating"},
		{Name: "Gym", Category: "entertainment", Icon: "gym"},
		{Name: "Parking", Category: "comfort", Icon: "parking"},
	}

	if err := db.CreateInBatches(amenities, 5).Error; err != nil {
		return err
	}
	log.Println("Created amenities")

	// Create conditions
	conditions := []models.Condition{
		{Name: "Pet Friendly", Type: "pets"},
		{Name: "No Pets", Type: "pets"},
		{Name: "Smoking Friendly", Type: "smoking"},
		{Name: "No Smoking", Type: "smoking"},
		{Name: "Wheelchair Accessible", Type: "accessibility"},
		{Name: "Family Friendly", Type: "family"},
		{Name: "Quiet Hours", Type: "rules"},
		{Name: "No Events", Type: "rules"},
	}

	if err := db.CreateInBatches(conditions, 5).Error; err != nil {
		return err
	}
	log.Println("Created conditions")

	// Create sample properties
	properties := []models.Amenity{}
	if err := db.Find(&properties).Error; err != nil {
		return err
	}

	prop1 := models.Property{
		ChannelID:   "ch_001",
		Name:        "Luxury Beach Villa",
		Description: "Beautiful beachfront villa with stunning ocean views",
		Location:    "Malibu, CA",
		City:        "Malibu",
		State:       "CA",
		Country:     "USA",
		Latitude:    34.0195,
		Longitude:   -118.6819,
		MaxGuests:   8,
		Bedrooms:    4,
		Bathrooms:   3,
		Rating:      4.8,
		ReviewCount: 125,
	}

	if err := db.Create(&prop1).Error; err != nil {
		return err
	}
	log.Printf("Created property: %s", prop1.Name)

	prop2 := models.Property{
		ChannelID:   "ch_002",
		Name:        "Downtown Apartment",
		Description: "Modern apartment in the heart of the city",
		Location:    "New York, NY",
		City:        "New York",
		State:       "NY",
		Country:     "USA",
		Latitude:    40.7128,
		Longitude:   -74.0060,
		MaxGuests:   4,
		Bedrooms:    2,
		Bathrooms:   2,
		Rating:      4.5,
		ReviewCount: 89,
	}

	if err := db.Create(&prop2).Error; err != nil {
		return err
	}
	log.Printf("Created property: %s", prop2.Name)

	// Create availability for next 90 days
	now := time.Now()
	for i := 0; i < 90; i++ {
		date := now.AddDate(0, 0, i)
		availability := models.Availability{
			PropertyID: prop1.ID,
			Date:       date,
			Available:  true,
			MinStay:    2,
			MaxGuests:  8,
		}
		if err := db.Create(&availability).Error; err != nil {
			return err
		}

		availability2 := models.Availability{
			PropertyID: prop2.ID,
			Date:       date,
			Available:  true,
			MinStay:    1,
			MaxGuests:  4,
		}
		if err := db.Create(&availability2).Error; err != nil {
			return err
		}
	}
	log.Println("Created availability records")

	// Create pricing for next 90 days
	for i := 0; i < 90; i++ {
		date := now.AddDate(0, 0, i)

		// Pricing for property 1
		basePrice := 500.0
		if date.Weekday() == 0 || date.Weekday() == 6 { // Weekend
			basePrice = 700.0
		}

		pricing1 := models.Pricing{
			PropertyID: prop1.ID,
			Date:       date,
			BasePrice:  basePrice,
			Taxes:      basePrice * 0.1,
			Fees:       basePrice * 0.05,
			Discount:   0,
		}
		if err := db.Create(&pricing1).Error; err != nil {
			return err
		}

		// Pricing for property 2
		basePrice2 := 200.0
		if date.Weekday() == 0 || date.Weekday() == 6 { // Weekend
			basePrice2 = 280.0
		}

		pricing2 := models.Pricing{
			PropertyID: prop2.ID,
			Date:       date,
			BasePrice:  basePrice2,
			Taxes:      basePrice2 * 0.1,
			Fees:       basePrice2 * 0.05,
			Discount:   0,
		}
		if err := db.Create(&pricing2).Error; err != nil {
			return err
		}
	}
	log.Println("Created pricing records")

	// Associate amenities with properties
	amenityList, err := getAmenities(db)
	if err != nil {
		return err
	}

	// Assign first 5 amenities to property 1
	if err := db.Model(&prop1).Association("Amenities").Append(amenityList[:5]); err != nil {
		return err
	}

	// Assign last 5 amenities to property 2
	if err := db.Model(&prop2).Association("Amenities").Append(amenityList[5:]); err != nil {
		return err
	}
	log.Println("Associated amenities with properties")

	// Associate conditions with properties
	conditionList, err := getConditions(db)
	if err != nil {
		return err
	}

	// Assign pet-friendly and no-smoking to property 1
	petFriendly := conditionList[0]
	noSmoking := conditionList[3]
	if err := db.Model(&prop1).Association("Conditions").Append([]models.Condition{petFriendly, noSmoking}); err != nil {
		return err
	}

	// Assign no-pets and smoking-friendly to property 2
	noPets := conditionList[1]
	smokingFriendly := conditionList[2]
	if err := db.Model(&prop2).Association("Conditions").Append([]models.Condition{noPets, smokingFriendly}); err != nil {
		return err
	}
	log.Println("Associated conditions with properties")

	log.Println("Database seed completed successfully")
	return nil
}

func getAmenities(db *gorm.DB) ([]models.Amenity, error) {
	var amenities []models.Amenity
	if err := db.Find(&amenities).Error; err != nil {
		return nil, err
	}
	return amenities, nil
}

func getConditions(db *gorm.DB) ([]models.Condition, error) {
	var conditions []models.Condition
	if err := db.Find(&conditions).Error; err != nil {
		return nil, err
	}
	return conditions, nil
}
