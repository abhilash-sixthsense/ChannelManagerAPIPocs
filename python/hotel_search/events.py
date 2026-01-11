import asyncio
from db import async_session
from models import Property, Availability
from redis_shards import get_redis_for_city
from bitarray import bitarray

async def index_property_to_redis(property: Property):
    redis_client = await get_redis_for_city(property.city)

    # City bitmap
    key = f"bitmap:city:{property.city}"
    ba = await bitmap_from_key(redis_client, key) or bitarray(property.id)
    while len(ba) < property.id:
        ba.append(0)
    ba[property.id-1] = 1
    await redis_client.set(key, ba.tobytes())

    # Guest count
    key = f"bitmap:max_guests:{property.max_guests}"
    ba = await bitmap_from_key(redis_client, key) or bitarray(property.id)
    while len(ba) < property.id:
        ba.append(0)
    ba[property.id-1] = 1
    await redis_client.set(key, ba.tobytes())

    # Amenities
    amenity_fields = [c for c in property.__dict__.keys() if property.__dict__[c] is True]
    for a in amenity_fields:
        key = f"bitmap:amenity:{a}"
        ba = await bitmap_from_key(redis_client, key) or bitarray(property.id)
        while len(ba) < property.id:
            ba.append(0)
        ba[property.id-1] = 1
        await redis_client.set(key, ba.tobytes())

    # GEO
    await redis_client.geoadd(f"geo:{property.city}", property.longitude, property.latitude, property.id)
