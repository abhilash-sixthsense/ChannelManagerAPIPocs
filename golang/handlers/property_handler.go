package handlers

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"channelmanager/cache"
	"channelmanager/database"
	"channelmanager/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler holds dependencies for HTTP handlers
type Handler struct {
	db               *gorm.DB
	redis            *cache.RedisClient
	propertyRepo     *database.PropertyRepository
	availabilityRepo *database.AvailabilityRepository
	pricingRepo      *database.PricingRepository
	amenityRepo      *database.AmenityRepository
	conditionRepo    *database.ConditionRepository
}

// NewHandler creates a new handler instance
func NewHandler(
	db *gorm.DB,
	redis *cache.RedisClient,
) *Handler {
	return &Handler{
		db:               db,
		redis:            redis,
		propertyRepo:     database.NewPropertyRepository(db),
		availabilityRepo: database.NewAvailabilityRepository(db),
		pricingRepo:      database.NewPricingRepository(db),
		amenityRepo:      database.NewAmenityRepository(db),
		conditionRepo:    database.NewConditionRepository(db),
	}
}

// SearchProperties handles the property search endpoint
func (h *Handler) SearchProperties(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse search filter from request
	filter := models.SearchFilter{}
	if err := c.ShouldBindJSON(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate pagination
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 20
	}

	// Generate cache key
	cacheKey := h.generateSearchCacheKey(filter)
	log.Printf("Cache key: %s", cacheKey)

	// Try to get from cache
	cachedResults, err := h.redis.GetSearchResultsCache(ctx, cacheKey)
	if err != nil {
		log.Printf("Cache retrieval error: %v", err)
	}

	if cachedResults != nil {
		log.Println("Cache HIT for search results")
		c.JSON(http.StatusOK, gin.H{
			"data":      cachedResults.Results,
			"total":     cachedResults.Total,
			"page":      cachedResults.Page,
			"limit":     cachedResults.Limit,
			"cached":    true,
			"cache_age": time.Since(cachedResults.UpdatedAt).Seconds(),
		})
		return
	}

	log.Println("Cache MISS for search results, fetching from database")

	// Fetch from database
	properties, total, err := h.propertyRepo.SearchProperties(filter)
	if err != nil {
		log.Printf("Database search error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search properties"})
		return
	}

	// Convert to search results
	results := h.convertPropertiesToSearchResults(ctx, properties, filter)

	// Cache the results (5 minute TTL for search results)
	cacheResults := &models.SearchResultsCache{
		Results: results,
		Total:   int(total),
		Page:    filter.Page,
		Limit:   filter.Limit,
	}

	if err := h.redis.SetSearchResultsCache(ctx, cacheKey, cacheResults, 5*time.Minute); err != nil {
		log.Printf("Failed to cache search results: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   results,
		"total":  total,
		"page":   filter.Page,
		"limit":  filter.Limit,
		"cached": false,
	})
}

// GetProperty retrieves a single property by ID
func (h *Handler) GetProperty(c *gin.Context) {
	ctx := c.Request.Context()

	propertyID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid property ID"})
		return
	}

	// Try to get from cache
	cachedProperty, err := h.redis.GetPropertyCache(ctx, uint(propertyID))
	if err != nil {
		log.Printf("Cache retrieval error: %v", err)
	}

	if cachedProperty != nil {
		log.Println("Cache HIT for property")
		c.JSON(http.StatusOK, gin.H{
			"data":   cachedProperty,
			"cached": true,
		})
		return
	}

	log.Println("Cache MISS for property, fetching from database")

	// Fetch from database
	property, err := h.propertyRepo.GetPropertyByID(uint(propertyID))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Property not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve property"})
		return
	}

	// Cache the property (1 hour TTL)
	if err := h.redis.SetPropertyCache(ctx, uint(propertyID), property, 1*time.Hour); err != nil {
		log.Printf("Failed to cache property: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   property,
		"cached": false,
	})
}

// GetPropertyAvailability retrieves availability for a property in a date range
func (h *Handler) GetPropertyAvailability(c *gin.Context) {
	ctx := c.Request.Context()

	propertyID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid property ID"})
		return
	}

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if startDate == "" || endDate == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_date and end_date are required"})
		return
	}

	// Fetch from database
	availabilities, err := h.availabilityRepo.GetAvailabilityForDateRange(uint(propertyID), startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve availability"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"property_id":    propertyID,
		"availabilities": availabilities,
	})
}

// GetAmenities retrieves all amenities
func (h *Handler) GetAmenities(c *gin.Context) {
	ctx := c.Request.Context()

	// Try to get from cache
	cachedAmenities, err := h.redis.GetAmenitiesCache(ctx)
	if err != nil {
		log.Printf("Cache retrieval error: %v", err)
	}

	if len(cachedAmenities) > 0 {
		log.Println("Cache HIT for amenities")
		c.JSON(http.StatusOK, gin.H{
			"data":   cachedAmenities,
			"cached": true,
		})
		return
	}

	log.Println("Cache MISS for amenities, fetching from database")

	// Fetch from database
	amenities, err := h.amenityRepo.GetAllAmenities()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve amenities"})
		return
	}

	// Cache amenities (24 hour TTL)
	if err := h.redis.SetAmenitiesCache(ctx, amenities, 24*time.Hour); err != nil {
		log.Printf("Failed to cache amenities: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   amenities,
		"cached": false,
	})
}

// GetConditions retrieves all conditions
func (h *Handler) GetConditions(c *gin.Context) {
	ctx := c.Request.Context()

	// Try to get from cache
	cachedConditions, err := h.redis.GetConditionsCache(ctx)
	if err != nil {
		log.Printf("Cache retrieval error: %v", err)
	}

	if len(cachedConditions) > 0 {
		log.Println("Cache HIT for conditions")
		c.JSON(http.StatusOK, gin.H{
			"data":   cachedConditions,
			"cached": true,
		})
		return
	}

	log.Println("Cache MISS for conditions, fetching from database")

	// Fetch from database
	conditions, err := h.conditionRepo.GetAllConditions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve conditions"})
		return
	}

	// Cache conditions (24 hour TTL)
	if err := h.redis.SetConditionsCache(ctx, conditions, 24*time.Hour); err != nil {
		log.Printf("Failed to cache conditions: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   conditions,
		"cached": false,
	})
}

// HealthCheck checks API health
func (h *Handler) HealthCheck(c *gin.Context) {
	ctx := c.Request.Context()

	// Check database
	dbHealth := "down"
	if err := h.db.WithContext(ctx).Raw("SELECT 1").Row().Scan(&dbHealth); err == nil {
		dbHealth = "up"
	}

	// Check Redis
	redisHealth := "down"
	if err := h.redis.HealthCheck(ctx); err == nil {
		redisHealth = "up"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"database":  dbHealth,
		"redis":     redisHealth,
		"timestamp": time.Now(),
	})
}

// HELPER METHODS

// generateSearchCacheKey generates a cache key for search results
func (h *Handler) generateSearchCacheKey(filter models.SearchFilter) string {
	// Create a hash of the search parameters for the cache key
	hash := md5.New()
	hashStr := fmt.Sprintf(
		"%s:%s:%s:%s:%d:%t:%t:%v:%v:%f:%f:%f:%f:%s:%d:%d",
		filter.Location,
		filter.City,
		filter.CheckinDate.String(),
		filter.CheckoutDate.String(),
		filter.NumberOfGuests,
		filter.PetFriendly,
		filter.SmokingFriendly,
		filter.AmenityIDs,
		filter.ConditionIDs,
		filter.MinRating,
		filter.MaxPrice,
		filter.MinPrice,
		filter.RadiusKm,
		filter.SortBy,
		filter.Page,
		filter.Limit,
	)

	hash.Write([]byte(hashStr))
	hashHex := hex.EncodeToString(hash.Sum(nil))

	return fmt.Sprintf("search:%s", hashHex)
}

// convertPropertiesToSearchResults converts Property models to SearchResult models
func (h *Handler) convertPropertiesToSearchResults(ctx context.Context, properties []models.Property, filter models.SearchFilter) []models.SearchResult {
	results := make([]models.SearchResult, 0, len(properties))

	for _, prop := range properties {
		// Get pricing information for the date range
		pricing, err := h.pricingRepo.GetPricingForDateRange(
			prop.ID,
			filter.CheckinDate.Format("2006-01-02"),
			filter.CheckoutDate.Format("2006-01-02"),
		)
		if err != nil {
			log.Printf("Failed to get pricing for property %d: %v", prop.ID, err)
			continue
		}

		// Calculate total price
		totalPrice := 0.0
		avgPrice := 0.0
		if len(pricing) > 0 {
			for _, p := range pricing {
				totalPrice += p.TotalPrice
			}
			avgPrice = totalPrice / float64(len(pricing))
		}

		// Extract amenity and condition names
		amenityNames := make([]string, 0, len(prop.Amenities))
		for _, a := range prop.Amenities {
			amenityNames = append(amenityNames, a.Name)
		}

		conditionNames := make([]string, 0, len(prop.Conditions))
		for _, cond := range prop.Conditions {
			conditionNames = append(conditionNames, cond.Name)
		}

		// Calculate distance if coordinates provided
		var distance *float64
		if filter.Latitude != nil && filter.Longitude != nil {
			dist := h.calculateDistance(*filter.Latitude, *filter.Longitude, prop.Latitude, prop.Longitude)
			distance = &dist
		}

		result := models.SearchResult{
			ID:            prop.ID,
			Name:          prop.Name,
			Description:   prop.Description,
			Location:      prop.Location,
			City:          prop.City,
			State:         prop.State,
			Country:       prop.Country,
			Rating:        prop.Rating,
			ReviewCount:   prop.ReviewCount,
			MaxGuests:     prop.MaxGuests,
			Bedrooms:      prop.Bedrooms,
			Bathrooms:     prop.Bathrooms,
			PricePerNight: avgPrice,
			TotalPrice:    totalPrice,
			Amenities:     amenityNames,
			Conditions:    conditionNames,
			Distance:      distance,
			Available:     true, // Simplified, should check availability in real scenario
		}

		results = append(results, result)
	}

	return results
}

// calculateDistance calculates distance between two coordinates using Haversine formula
func (h *Handler) calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth's radius in km
	dlat := (lat2 - lat1) * 3.14159 / 180
	dlon := (lon2 - lon1) * 3.14159 / 180
	a := (dlat/2)*(dlat/2) + (dlon/2)*(dlon/2)*
		((3.14159/180)*(lat1))*((3.14159/180)*(lat1))
	c := 2 * 3.14159 / 180 * a
	return R * c
}
