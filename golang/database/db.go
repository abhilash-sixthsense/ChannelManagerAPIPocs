package database

import (
	"fmt"
	"log"

	"channelmanager/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB holds the database connection
var DB *gorm.DB

// Config holds database configuration
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// InitializeDatabase initializes the database connection and runs migrations
func InitializeDatabase(config Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.DBName,
		config.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	DB = db

	// Run migrations
	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Database initialized successfully")
	return db, nil
}

// runMigrations runs all database migrations
func runMigrations(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.PropertyRating{},
		&models.Property{},
		&models.Amenity{},
		&models.Condition{},
		&models.Availability{},
		&models.Pricing{},
		&models.Event{},
	)
}

// PropertyRepository handles property database operations
type PropertyRepository struct {
	db *gorm.DB
}

// NewPropertyRepository creates a new property repository
func NewPropertyRepository(db *gorm.DB) *PropertyRepository {
	return &PropertyRepository{db: db}
}

// GetPropertyByID retrieves a property by ID
func (r *PropertyRepository) GetPropertyByID(id uint) (*models.Property, error) {
	var property models.Property
	if err := r.db.Preload("Amenities").Preload("Conditions").First(&property, id).Error; err != nil {
		return nil, err
	}
	return &property, nil
}

// GetPropertiesByLocation retrieves properties by location with filtering
func (r *PropertyRepository) GetPropertiesByLocation(location string, limit int, offset int) ([]models.Property, int64, error) {
	var properties []models.Property
	var total int64

	query := r.db.Where("location ILIKE ?", "%"+location+"%")
	query.Model(&models.Property{}).Count(&total)

	if err := query.Preload("Amenities").Preload("Conditions").
		Limit(limit).Offset(offset).
		Find(&properties).Error; err != nil {
		return nil, 0, err
	}

	return properties, total, nil
}

// GetPropertiesByCity retrieves properties by city
func (r *PropertyRepository) GetPropertiesByCity(city string, limit int, offset int) ([]models.Property, int64, error) {
	var properties []models.Property
	var total int64

	query := r.db.Where("city ILIKE ?", "%"+city+"%")
	query.Model(&models.Property{}).Count(&total)

	if err := query.Preload("Amenities").Preload("Conditions").
		Limit(limit).Offset(offset).
		Find(&properties).Error; err != nil {
		return nil, 0, err
	}

	return properties, total, nil
}

// SearchProperties performs a complex search with multiple filters
func (r *PropertyRepository) SearchProperties(filter models.SearchFilter) ([]models.Property, int64, error) {
	query := r.db

	// Location filter
	if filter.Location != "" {
		query = query.Where("location ILIKE ?", "%"+filter.Location+"%")
	}

	// City filter
	if filter.City != "" {
		query = query.Where("city ILIKE ?", "%"+filter.City+"%")
	}

	// Guest count filter
	if filter.NumberOfGuests > 0 {
		query = query.Where("max_guests >= ?", filter.NumberOfGuests)
	}

	// Price range filter
	if filter.MinPrice > 0 || filter.MaxPrice > 0 {
		query = query.Joins("LEFT JOIN pricing ON pricing.property_id = properties.id").
			Where("pricing.total_price BETWEEN ? AND ?", filter.MinPrice, filter.MaxPrice)
	}

	// Rating filter
	if filter.MinRating > 0 {
		query = query.Where("rating >= ?", filter.MinRating)
	}

	// Amenities filter
	if len(filter.AmenityIDs) > 0 {
		query = query.Joins("LEFT JOIN property_amenities ON property_amenities.property_id = properties.id").
			Where("property_amenities.amenity_id IN ?", filter.AmenityIDs).
			Distinct()
	}

	// Conditions filter (pet-friendly, smoking-friendly, etc.)
	if len(filter.ConditionIDs) > 0 {
		query = query.Joins("LEFT JOIN property_conditions ON property_conditions.property_id = properties.id").
			Where("property_conditions.condition_id IN ?", filter.ConditionIDs).
			Distinct()
	}

	// Specific condition filters
	if filter.PetFriendly != nil && *filter.PetFriendly {
		query = query.Joins("LEFT JOIN property_conditions pc ON pc.property_id = properties.id").
			Joins("LEFT JOIN conditions c ON c.id = pc.condition_id").
			Where("c.type = ? AND c.name ILIKE ?", "pets", "%friendly%")
	}

	if filter.SmokingFriendly != nil && *filter.SmokingFriendly {
		query = query.Joins("LEFT JOIN property_conditions pc ON pc.property_id = properties.id").
			Joins("LEFT JOIN conditions c ON c.id = pc.condition_id").
			Where("c.type = ? AND c.name ILIKE ?", "smoking", "%friendly%")
	}

	// Availability filter for date range
	if !filter.CheckinDate.IsZero() && !filter.CheckoutDate.IsZero() {
		query = query.Joins("LEFT JOIN availabilities ON availabilities.property_id = properties.id").
			Where("availabilities.date BETWEEN ? AND ? AND availabilities.available = ?",
				filter.CheckinDate, filter.CheckoutDate, true)
	}

	// Distance filter (if coordinates provided)
	if filter.Latitude != nil && filter.Longitude != nil && filter.RadiusKm > 0 {
		// Using PostgreSQL PostGIS distance calculation
		query = query.Where(
			"earth_distance(ll_to_earth(latitude, longitude), ll_to_earth(?, ?)) / 1000 <= ?",
			*filter.Latitude, *filter.Longitude, filter.RadiusKm,
		)
	}

	// Count total
	var total int64
	if err := query.Model(&models.Property{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Sorting
	sortBy := "rating"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}
	query = query.Order(sortBy + " DESC")

	// Pagination
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	// Execute query
	var properties []models.Property
	if err := query.
		Preload("Amenities").
		Preload("Conditions").
		Limit(limit).
		Offset(offset).
		Find(&properties).Error; err != nil {
		return nil, 0, err
	}

	return properties, total, nil
}

// AvailabilityRepository handles availability database operations
type AvailabilityRepository struct {
	db *gorm.DB
}

// NewAvailabilityRepository creates a new availability repository
func NewAvailabilityRepository(db *gorm.DB) *AvailabilityRepository {
	return &AvailabilityRepository{db: db}
}

// GetAvailabilityForDateRange retrieves availability for a date range
func (r *AvailabilityRepository) GetAvailabilityForDateRange(propertyID uint, startDate, endDate string) ([]models.Availability, error) {
	var availabilities []models.Availability
	if err := r.db.Where("property_id = ? AND date BETWEEN ? AND ?", propertyID, startDate, endDate).
		Find(&availabilities).Error; err != nil {
		return nil, err
	}
	return availabilities, nil
}

// UpdateAvailability updates availability for a property
func (r *AvailabilityRepository) UpdateAvailability(availability *models.Availability) error {
	return r.db.Save(availability).Error
}

// BulkUpdateAvailability updates multiple availabilities
func (r *AvailabilityRepository) BulkUpdateAvailability(availabilities []models.Availability) error {
	return r.db.SaveInBatches(availabilities, 100).Error
}

// PricingRepository handles pricing database operations
type PricingRepository struct {
	db *gorm.DB
}

// NewPricingRepository creates a new pricing repository
func NewPricingRepository(db *gorm.DB) *PricingRepository {
	return &PricingRepository{db: db}
}

// GetPricingForDateRange retrieves pricing for a date range
func (r *PricingRepository) GetPricingForDateRange(propertyID uint, startDate, endDate string) ([]models.Pricing, error) {
	var pricing []models.Pricing
	if err := r.db.Where("property_id = ? AND date BETWEEN ? AND ?", propertyID, startDate, endDate).
		Find(&pricing).Error; err != nil {
		return nil, err
	}
	return pricing, nil
}

// UpdatePricing updates pricing for a property
func (r *PricingRepository) UpdatePricing(pricing *models.Pricing) error {
	return r.db.Save(pricing).Error
}

// AmenityRepository handles amenity database operations
type AmenityRepository struct {
	db *gorm.DB
}

// NewAmenityRepository creates a new amenity repository
func NewAmenityRepository(db *gorm.DB) *AmenityRepository {
	return &AmenityRepository{db: db}
}

// GetAllAmenities retrieves all amenities
func (r *AmenityRepository) GetAllAmenities() ([]models.Amenity, error) {
	var amenities []models.Amenity
	if err := r.db.Find(&amenities).Error; err != nil {
		return nil, err
	}
	return amenities, nil
}

// GetAmenitiesByCategory retrieves amenities by category
func (r *AmenityRepository) GetAmenitiesByCategory(category string) ([]models.Amenity, error) {
	var amenities []models.Amenity
	if err := r.db.Where("category = ?", category).Find(&amenities).Error; err != nil {
		return nil, err
	}
	return amenities, nil
}

// ConditionRepository handles condition database operations
type ConditionRepository struct {
	db *gorm.DB
}

// NewConditionRepository creates a new condition repository
func NewConditionRepository(db *gorm.DB) *ConditionRepository {
	return &ConditionRepository{db: db}
}

// GetAllConditions retrieves all conditions
func (r *ConditionRepository) GetAllConditions() ([]models.Condition, error) {
	var conditions []models.Condition
	if err := r.db.Find(&conditions).Error; err != nil {
		return nil, err
	}
	return conditions, nil
}

// GetConditionsByType retrieves conditions by type
func (r *ConditionRepository) GetConditionsByType(condType string) ([]models.Condition, error) {
	var conditions []models.Condition
	if err := r.db.Where("type = ?", condType).Find(&conditions).Error; err != nil {
		return nil, err
	}
	return conditions, nil
}

// EventRepository handles event database operations
type EventRepository struct {
	db *gorm.DB
}

// NewEventRepository creates a new event repository
func NewEventRepository(db *gorm.DB) *EventRepository {
	return &EventRepository{db: db}
}

// CreateEvent creates a new event
func (r *EventRepository) CreateEvent(event *models.Event) error {
	return r.db.Create(event).Error
}

// GetUnprocessedEvents retrieves unprocessed events
func (r *EventRepository) GetUnprocessedEvents(limit int) ([]models.Event, error) {
	var events []models.Event
	if err := r.db.Where("processed = ?", false).Limit(limit).Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

// MarkEventAsProcessed marks an event as processed
func (r *EventRepository) MarkEventAsProcessed(eventID uint) error {
	return r.db.Model(&models.Event{}).Where("id = ?", eventID).Update("processed", true).Error
}
