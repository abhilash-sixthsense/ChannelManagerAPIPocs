package handlers

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"channelmanager/cache"
	"channelmanager/database"
	"channelmanager/models"

	"gorm.io/gorm"
)

// EventListener handles database change events for cache invalidation
type EventListener struct {
	db        *gorm.DB
	redis     *cache.RedisClient
	eventRepo *database.EventRepository
	ticker    *time.Ticker
	done      chan bool
}

// NewEventListener creates a new event listener
func NewEventListener(db *gorm.DB, redis *cache.RedisClient) *EventListener {
	return &EventListener{
		db:        db,
		redis:     redis,
		eventRepo: database.NewEventRepository(db),
		ticker:    time.NewTicker(5 * time.Second), // Check for events every 5 seconds
		done:      make(chan bool),
	}
}

// Start begins listening for database change events
func (el *EventListener) Start() {
	go func() {
		log.Println("Event listener started")
		for {
			select {
			case <-el.ticker.C:
				el.processUnprocessedEvents()
			case <-el.done:
				log.Println("Event listener stopped")
				return
			}
		}
	}()
}

// Stop stops the event listener
func (el *EventListener) Stop() {
	el.ticker.Stop()
	el.done <- true
}

// processUnprocessedEvents processes unprocessed events and invalidates cache
func (el *EventListener) processUnprocessedEvents() {
	ctx := context.Background()

	// Get unprocessed events
	events, err := el.eventRepo.GetUnprocessedEvents(100)
	if err != nil {
		log.Printf("Failed to get unprocessed events: %v", err)
		return
	}

	if len(events) == 0 {
		return
	}

	log.Printf("Processing %d unprocessed events", len(events))

	for _, event := range events {
		el.handleEvent(ctx, event)

		// Mark event as processed
		if err := el.eventRepo.MarkEventAsProcessed(event.ID); err != nil {
			log.Printf("Failed to mark event %d as processed: %v", event.ID, err)
		}
	}
}

// handleEvent handles a single event and invalidates relevant cache
func (el *EventListener) handleEvent(ctx context.Context, event models.Event) {
	log.Printf("Processing event: Type=%s, Table=%s, RecordID=%d", event.EventType, event.TableName, event.RecordID)

	switch event.TableName {
	case "properties":
		el.handlePropertyEvent(ctx, event)
	case "availabilities":
		el.handleAvailabilityEvent(ctx, event)
	case "pricing":
		el.handlePricingEvent(ctx, event)
	case "amenities":
		el.handleAmenityEvent(ctx, event)
	case "conditions":
		el.handleConditionEvent(ctx, event)
	case "property_amenities", "property_conditions":
		el.handlePropertyRelationEvent(ctx, event)
	default:
		log.Printf("Unknown event table: %s", event.TableName)
	}
}

// handlePropertyEvent handles property-related events
func (el *EventListener) handlePropertyEvent(ctx context.Context, event models.Event) {
	propertyID := event.RecordID

	// Invalidate property cache
	if err := el.redis.InvalidatePropertyCache(ctx, propertyID); err != nil {
		log.Printf("Failed to invalidate property cache: %v", err)
	}

	// Invalidate search cache (broad invalidation)
	if err := el.redis.InvalidateSearchCache(ctx, "", ""); err != nil {
		log.Printf("Failed to invalidate search cache: %v", err)
	}

	// Invalidate availability cache
	if err := el.redis.InvalidateAvailabilityCache(ctx, propertyID); err != nil {
		log.Printf("Failed to invalidate availability cache: %v", err)
	}

	log.Printf("Invalidated caches for property %d", propertyID)
}

// handleAvailabilityEvent handles availability-related events
func (el *EventListener) handleAvailabilityEvent(ctx context.Context, event models.Event) {
	var availability models.Availability
	if err := json.Unmarshal(event.Data, &availability); err != nil {
		log.Printf("Failed to unmarshal availability data: %v", err)
		return
	}

	propertyID := availability.PropertyID

	// Invalidate availability cache
	if err := el.redis.InvalidateAvailabilityCache(ctx, propertyID); err != nil {
		log.Printf("Failed to invalidate availability cache: %v", err)
	}

	// Invalidate search cache (availability affects search results)
	if err := el.redis.InvalidateSearchCache(ctx, "", ""); err != nil {
		log.Printf("Failed to invalidate search cache: %v", err)
	}

	log.Printf("Invalidated availability cache for property %d", propertyID)
}

// handlePricingEvent handles pricing-related events
func (el *EventListener) handlePricingEvent(ctx context.Context, event models.Event) {
	var pricing models.Pricing
	if err := json.Unmarshal(event.Data, &pricing); err != nil {
		log.Printf("Failed to unmarshal pricing data: %v", err)
		return
	}

	propertyID := pricing.PropertyID

	// Invalidate search cache (pricing affects search results)
	if err := el.redis.InvalidateSearchCache(ctx, "", ""); err != nil {
		log.Printf("Failed to invalidate search cache: %v", err)
	}

	// Invalidate property cache
	if err := el.redis.InvalidatePropertyCache(ctx, propertyID); err != nil {
		log.Printf("Failed to invalidate property cache: %v", err)
	}

	log.Printf("Invalidated pricing-related cache for property %d", propertyID)
}

// handleAmenityEvent handles amenity-related events
func (el *EventListener) handleAmenityEvent(ctx context.Context, event models.Event) {
	// Invalidate amenities cache
	if err := el.redis.InvalidateAmenitiesCache(ctx); err != nil {
		log.Printf("Failed to invalidate amenities cache: %v", err)
	}

	// Invalidate search cache (amenities affect search results)
	if err := el.redis.InvalidateSearchCache(ctx, "", ""); err != nil {
		log.Printf("Failed to invalidate search cache: %v", err)
	}

	log.Printf("Invalidated amenity-related cache")
}

// handleConditionEvent handles condition-related events
func (el *EventListener) handleConditionEvent(ctx context.Context, event models.Event) {
	// Invalidate conditions cache
	if err := el.redis.InvalidateConditionsCache(ctx); err != nil {
		log.Printf("Failed to invalidate conditions cache: %v", err)
	}

	// Invalidate search cache (conditions affect search results)
	if err := el.redis.InvalidateSearchCache(ctx, "", ""); err != nil {
		log.Printf("Failed to invalidate search cache: %v", err)
	}

	log.Printf("Invalidated condition-related cache")
}

// handlePropertyRelationEvent handles property relationship changes (amenities, conditions)
func (el *EventListener) handlePropertyRelationEvent(ctx context.Context, event models.Event) {
	// Invalidate search cache (relationships affect search results)
	if err := el.redis.InvalidateSearchCache(ctx, "", ""); err != nil {
		log.Printf("Failed to invalidate search cache: %v", err)
	}

	// Invalidate property cache
	if err := el.redis.InvalidatePropertyCache(ctx, event.RecordID); err != nil {
		log.Printf("Failed to invalidate property cache: %v", err)
	}

	log.Printf("Invalidated cache for property relationship change")
}
