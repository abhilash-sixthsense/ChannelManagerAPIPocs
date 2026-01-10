package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/lib/pq"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Property represents a property/room listing in the system
type Property struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	ChannelID   string         `gorm:"index:idx_channel_property" json:"channel_id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Location    string         `gorm:"index:idx_location" json:"location"`
	City        string         `gorm:"index:idx_city" json:"city"`
	State       string         `json:"state"`
	Country     string         `json:"country"`
	Latitude    float64        `json:"latitude"`
	Longitude   float64        `json:"longitude"`
	MaxGuests   int            `json:"max_guests"`
	Bedrooms    int            `json:"bedrooms"`
	Bathrooms   int            `json:"bathrooms"`
	Rating      float32        `json:"rating"`
	ReviewCount int            `json:"review_count"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Amenities      []Amenity      `gorm:"many2many:property_amenities" json:"amenities"`
	Conditions     []Condition    `gorm:"many2many:property_conditions" json:"conditions"`
	Availabilities []Availability `gorm:"foreignKey:PropertyID" json:"availabilities,omitempty"`
	Pricing        []Pricing      `gorm:"foreignKey:PropertyID" json:"pricing,omitempty"`
}

// TableName specifies the table name
func (Property) TableName() string {
	return "properties"
}

// Amenity represents amenities like AC, WiFi, Pool, etc.
type Amenity struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"uniqueIndex;type:varchar(100)" json:"name"`
	Category  string         `json:"category"` // e.g., "comfort", "entertainment", "kitchen"
	Icon      string         `json:"icon"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Properties []Property `gorm:"many2many:property_amenities" json:"-"`
}

// TableName specifies the table name
func (Amenity) TableName() string {
	return "amenities"
}

// Condition represents conditions like pet-friendly, smoking-friendly, wheelchair accessible, etc.
type Condition struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"uniqueIndex;type:varchar(100)" json:"name"`
	Type      string         `json:"type"` // e.g., "pets", "smoking", "accessibility"
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Properties []Property `gorm:"many2many:property_conditions" json:"-"`
}

// TableName specifies the table name
func (Condition) TableName() string {
	return "conditions"
}

// PropertyRating represents star rating of property (2-star, 3-star, 5-star, etc.)
type PropertyRating struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"uniqueIndex;type:varchar(50)" json:"name"` // e.g., "2-star", "3-star"
	Stars     int            `json:"stars"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name
func (PropertyRating) TableName() string {
	return "property_ratings"
}

// Availability represents room availability for specific dates
type Availability struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	PropertyID uint           `gorm:"index:idx_property_date" json:"property_id"`
	Date       time.Time      `gorm:"index:idx_property_date;type:date" json:"date"`
	Available  bool           `gorm:"index" json:"available"`
	MinStay    int            `json:"min_stay"`
	MaxGuests  int            `json:"max_guests"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationship
	Property *Property `gorm:"foreignKey:PropertyID" json:"-"`
}

// TableName specifies the table name
func (Availability) TableName() string {
	return "availabilities"
}

// Pricing represents pricing for specific dates
type Pricing struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	PropertyID uint           `gorm:"index:idx_property_pricing_date" json:"property_id"`
	Date       time.Time      `gorm:"index:idx_property_pricing_date;type:date" json:"date"`
	BasePrice  float64        `json:"base_price"`
	Taxes      float64        `json:"taxes"`
	Fees       float64        `json:"fees"`
	Discount   float64        `json:"discount"`
	TotalPrice float64        `gorm:"generatedColumn:STORED" json:"total_price"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationship
	Property *Property `gorm:"foreignKey:PropertyID" json:"-"`
}

// TableName specifies the table name
func (Pricing) TableName() string {
	return "pricing"
}

// SearchFilter represents the search criteria for property search
type SearchFilter struct {
	Location        string        `json:"location"`
	City            string        `json:"city"`
	CheckinDate     time.Time     `json:"checkin_date"`
	CheckoutDate    time.Time     `json:"checkout_date"`
	NumberOfGuests  int           `json:"number_of_guests"`
	PetFriendly     *bool         `json:"pet_friendly"`
	SmokingFriendly *bool         `json:"smoking_friendly"`
	AmenityIDs      pq.Int64Array `json:"amenity_ids"`
	ConditionIDs    pq.Int64Array `json:"condition_ids"`
	MinRating       float32       `json:"min_rating"`
	MaxPrice        float64       `json:"max_price"`
	MinPrice        float64       `json:"min_price"`
	Latitude        *float64      `json:"latitude"`
	Longitude       *float64      `json:"longitude"`
	RadiusKm        float64       `json:"radius_km"`
	SortBy          string        `json:"sort_by"` // price, rating, distance
	Page            int           `json:"page"`
	Limit           int           `json:"limit"`
}

// Scan implements the sql.Scanner interface
func (s *SearchFilter) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return gorm.ErrInvalidData
	}
	return json.Unmarshal(bytes, &s)
}

// Value implements the driver.Valuer interface
func (s SearchFilter) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// SearchResult represents a property in search results
type SearchResult struct {
	ID            uint     `json:"id"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Location      string   `json:"location"`
	City          string   `json:"city"`
	State         string   `json:"state"`
	Country       string   `json:"country"`
	Rating        float32  `json:"rating"`
	ReviewCount   int      `json:"review_count"`
	MaxGuests     int      `json:"max_guests"`
	Bedrooms      int      `json:"bedrooms"`
	Bathrooms     int      `json:"bathrooms"`
	PricePerNight float64  `json:"price_per_night"`
	TotalPrice    float64  `json:"total_price"`
	Amenities     []string `json:"amenities"`
	Conditions    []string `json:"conditions"`
	Distance      *float64 `json:"distance,omitempty"`
	Available     bool     `json:"available"`
}

// PropertyAvailabilityCache represents cached availability data in Redis
type PropertyAvailabilityCache struct {
	PropertyID uint      `json:"property_id"`
	Available  bool      `json:"available"`
	MinStay    int       `json:"min_stay"`
	MaxGuests  int       `json:"max_guests"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// SearchResultsCache represents cached search results in Redis
type SearchResultsCache struct {
	Results   []SearchResult `json:"results"`
	Total     int            `json:"total"`
	Page      int            `json:"page"`
	Limit     int            `json:"limit"`
	UpdatedAt time.Time      `json:"updated_at"`
	ExpiresAt time.Time      `json:"expires_at"`
}

// Event represents database change events for cache invalidation
type Event struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	EventType string         `json:"event_type"` // CREATE, UPDATE, DELETE
	TableName string         `json:"table_name"`
	RecordID  uint           `json:"record_id"`
	Data      datatypes.JSON `json:"data"`
	CreatedAt time.Time      `json:"created_at"`
	Processed bool           `gorm:"index" json:"processed"`
}

// TableName specifies the table name
func (Event) TableName() string {
	return "events"
}
