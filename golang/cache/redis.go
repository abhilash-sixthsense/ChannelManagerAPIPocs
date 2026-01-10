package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"channelmanager/models"

	"github.com/redis/go-redis/v9"
)

// RedisClient holds the Redis client instance
type RedisClient struct {
	client *redis.Client
}

// Config holds Redis configuration
type Config struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// NewRedisClient creates a new Redis client
func NewRedisClient(config Config) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password: config.Password,
		DB:       config.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Println("Redis connected successfully")
	return &RedisClient{client: client}, nil
}

// Close closes the Redis connection
func (rc *RedisClient) Close() error {
	return rc.client.Close()
}

// GetClient returns the underlying Redis client
func (rc *RedisClient) GetClient() *redis.Client {
	return rc.client
}

// AVAILABILITY CACHE OPERATIONS

// GetAvailabilityCache retrieves availability from cache
func (rc *RedisClient) GetAvailabilityCache(ctx context.Context, propertyID uint, date string) (*models.PropertyAvailabilityCache, error) {
	key := fmt.Sprintf("availability:%d:%s", propertyID, date)
	val, err := rc.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, err
	}

	var availability models.PropertyAvailabilityCache
	if err := json.Unmarshal([]byte(val), &availability); err != nil {
		return nil, err
	}

	return &availability, nil
}

// SetAvailabilityCache sets availability in cache with TTL
func (rc *RedisClient) SetAvailabilityCache(ctx context.Context, propertyID uint, date string, availability *models.PropertyAvailabilityCache, ttl time.Duration) error {
	key := fmt.Sprintf("availability:%d:%s", propertyID, date)
	data, err := json.Marshal(availability)
	if err != nil {
		return err
	}

	return rc.client.Set(ctx, key, data, ttl).Err()
}

// InvalidateAvailabilityCache invalidates availability cache for a property
func (rc *RedisClient) InvalidateAvailabilityCache(ctx context.Context, propertyID uint) error {
	pattern := fmt.Sprintf("availability:%d:*", propertyID)
	return rc.deleteByPattern(ctx, pattern)
}

// InvalidateAvailabilityDateRange invalidates availability cache for a date range
func (rc *RedisClient) InvalidateAvailabilityDateRange(ctx context.Context, propertyID uint, startDate, endDate string) error {
	pattern := fmt.Sprintf("availability:%d:*", propertyID)
	iter := rc.client.Scan(ctx, 0, pattern, 0).Iterator()

	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if len(keys) > 0 {
		return rc.client.Del(ctx, keys...).Err()
	}
	return nil
}

// SEARCH RESULTS CACHE OPERATIONS

// GetSearchResultsCache retrieves cached search results
func (rc *RedisClient) GetSearchResultsCache(ctx context.Context, cacheKey string) (*models.SearchResultsCache, error) {
	val, err := rc.client.Get(ctx, cacheKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, err
	}

	var results models.SearchResultsCache
	if err := json.Unmarshal([]byte(val), &results); err != nil {
		return nil, err
	}

	// Check if cache has expired
	if results.ExpiresAt.Before(time.Now()) {
		// Cache expired, delete it
		rc.client.Del(ctx, cacheKey)
		return nil, nil
	}

	return &results, nil
}

// SetSearchResultsCache sets search results in cache with TTL
func (rc *RedisClient) SetSearchResultsCache(ctx context.Context, cacheKey string, results *models.SearchResultsCache, ttl time.Duration) error {
	results.UpdatedAt = time.Now()
	results.ExpiresAt = time.Now().Add(ttl)

	data, err := json.Marshal(results)
	if err != nil {
		return err
	}

	return rc.client.Set(ctx, cacheKey, data, ttl).Err()
}

// InvalidateSearchCache invalidates search cache by pattern
func (rc *RedisClient) InvalidateSearchCache(ctx context.Context, location string, city string) error {
	patterns := []string{
		fmt.Sprintf("search:location:%s:*", location),
		fmt.Sprintf("search:city:%s:*", city),
		"search:*",
	}

	for _, pattern := range patterns {
		if err := rc.deleteByPattern(ctx, pattern); err != nil {
			return err
		}
	}

	return nil
}

// PROPERTY CACHE OPERATIONS

// GetPropertyCache retrieves cached property details
func (rc *RedisClient) GetPropertyCache(ctx context.Context, propertyID uint) (*models.Property, error) {
	key := fmt.Sprintf("property:%d", propertyID)
	val, err := rc.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, err
	}

	var property models.Property
	if err := json.Unmarshal([]byte(val), &property); err != nil {
		return nil, err
	}

	return &property, nil
}

// SetPropertyCache sets property details in cache
func (rc *RedisClient) SetPropertyCache(ctx context.Context, propertyID uint, property *models.Property, ttl time.Duration) error {
	key := fmt.Sprintf("property:%d", propertyID)
	data, err := json.Marshal(property)
	if err != nil {
		return err
	}

	return rc.client.Set(ctx, key, data, ttl).Err()
}

// InvalidatePropertyCache invalidates property cache
func (rc *RedisClient) InvalidatePropertyCache(ctx context.Context, propertyID uint) error {
	key := fmt.Sprintf("property:%d", propertyID)
	return rc.client.Del(ctx, key).Err()
}

// AMENITIES & CONDITIONS CACHE OPERATIONS

// GetAmenitiesCache retrieves all amenities from cache
func (rc *RedisClient) GetAmenitiesCache(ctx context.Context) ([]models.Amenity, error) {
	key := "amenities:all"
	val, err := rc.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, err
	}

	var amenities []models.Amenity
	if err := json.Unmarshal([]byte(val), &amenities); err != nil {
		return nil, err
	}

	return amenities, nil
}

// SetAmenitiesCache sets all amenities in cache
func (rc *RedisClient) SetAmenitiesCache(ctx context.Context, amenities []models.Amenity, ttl time.Duration) error {
	key := "amenities:all"
	data, err := json.Marshal(amenities)
	if err != nil {
		return err
	}

	return rc.client.Set(ctx, key, data, ttl).Err()
}

// InvalidateAmenitiesCache invalidates amenities cache
func (rc *RedisClient) InvalidateAmenitiesCache(ctx context.Context) error {
	keys := []string{"amenities:all", "amenities:*"}
	for _, key := range keys {
		if err := rc.deleteByPattern(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

// GetConditionsCache retrieves all conditions from cache
func (rc *RedisClient) GetConditionsCache(ctx context.Context) ([]models.Condition, error) {
	key := "conditions:all"
	val, err := rc.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, err
	}

	var conditions []models.Condition
	if err := json.Unmarshal([]byte(val), &conditions); err != nil {
		return nil, err
	}

	return conditions, nil
}

// SetConditionsCache sets all conditions in cache
func (rc *RedisClient) SetConditionsCache(ctx context.Context, conditions []models.Condition, ttl time.Duration) error {
	key := "conditions:all"
	data, err := json.Marshal(conditions)
	if err != nil {
		return err
	}

	return rc.client.Set(ctx, key, data, ttl).Err()
}

// InvalidateConditionsCache invalidates conditions cache
func (rc *RedisClient) InvalidateConditionsCache(ctx context.Context) error {
	keys := []string{"conditions:all", "conditions:*"}
	for _, key := range keys {
		if err := rc.deleteByPattern(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

// UTILITY METHODS

// deleteByPattern deletes all keys matching a pattern
func (rc *RedisClient) deleteByPattern(ctx context.Context, pattern string) error {
	iter := rc.client.Scan(ctx, 0, pattern, 0).Iterator()

	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return err
	}

	if len(keys) > 0 {
		return rc.client.Del(ctx, keys...).Err()
	}

	return nil
}

// Flush flushes the entire Redis database
func (rc *RedisClient) Flush(ctx context.Context) error {
	return rc.client.FlushDB(ctx).Err()
}

// HealthCheck checks Redis connection health
func (rc *RedisClient) HealthCheck(ctx context.Context) error {
	return rc.client.Ping(ctx).Err()
}

// GetCacheStats returns cache statistics
func (rc *RedisClient) GetCacheStats(ctx context.Context) (map[string]string, error) {
	return rc.client.Info(ctx, "stats").Val(), nil
}

// SetWithExpiry sets a value with expiry time
func (rc *RedisClient) SetWithExpiry(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return rc.client.Set(ctx, key, data, ttl).Err()
}

// GetWithExpiry gets a value from cache
func (rc *RedisClient) GetWithExpiry(ctx context.Context, key string, result interface{}) error {
	val, err := rc.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil // Cache miss
		}
		return err
	}

	return json.Unmarshal([]byte(val), result)
}
