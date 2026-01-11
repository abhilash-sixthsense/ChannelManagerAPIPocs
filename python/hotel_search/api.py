from fastapi import FastAPI, Query
from datetime import date, timedelta
from typing import List, Optional
from bitarray import bitarray
import asyncio
from redis_shards import get_redis_for_city, bitmap_from_key

app = FastAPI()

criteria_fields = [
    "has_pool","has_bar","smoking_allowed","has_wifi","has_gym","has_spa","has_parking",
    "has_aircon","has_kitchen","has_balcony","pet_friendly","near_beach","near_airport",
    "family_friendly","romantic","business_ready","breakfast_included","free_cancellation",
    "early_checkin","late_checkout","wheelchair_accessible","tv","minibar","room_service","non_smoking_rooms"
]

@get("/search")
async def search_properties(
    city: str,
    min_guests: Optional[int] = None,
    amenities: Optional[List[str]] = Query(None),
    checkin: Optional[date] = None,
    checkout: Optional[date] = None,
    lat: Optional[float] = None,
    lon: Optional[float] = None,
    radius_miles: Optional[float] = 10
):
    redis_client = await get_redis_for_city(city)
    bitmaps = []

    # City bitmap
    city_ba = await bitmap_from_key(redis_client, f"bitmap:city:{city}")
    if city_ba:
        bitmaps.append(city_ba)

    # Guest count
    if min_guests:
        guest_bitmaps = []
        for g in range(min_guests, 7):
            ba_g = await bitmap_from_key(redis_client, f"bitmap:max_guests:{g}")
            if ba_g:
                guest_bitmaps.append(ba_g)
        if guest_bitmaps:
            ba_union = guest_bitmaps[0].copy()
            for b in guest_bitmaps[1:]:
                ba_union |= b
            bitmaps.append(ba_union)

    # Amenities
    if amenities:
        amenity_bitmaps = await asyncio.gather(*[
            bitmap_from_key(redis_client, f"bitmap:amenity:{a}")
            for a in amenities if a in criteria_fields
        ])
        for ba in amenity_bitmaps:
            if ba:
                bitmaps.append(ba)

    # Availability (optional, dummy for now)
    # Add similar logic for dates if needed

    # Geo
    if lat is not None and lon is not None:
        geo_ids = await redis_client.georadius(f"geo:{city}", lon, lat, radius_miles, unit="mi")
        if geo_ids:
            max_pid = max([int(pid) for pid in geo_ids])
            geo_ba = bitarray(max_pid)
            geo_ba.setall(0)
            for pid in geo_ids:
                geo_ba[int(pid)-1] = 1
            bitmaps.append(geo_ba)

    if not bitmaps:
        return {"properties": []}

    # Intersect all bitmaps
    result = bitmaps[0].copy()
    for b in bitmaps[1:]:
        result &= b

    return {"properties": [i+1 for i, bit in enumerate(result) if bit]}
