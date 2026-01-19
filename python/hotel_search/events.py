import asyncio

from bitarray import bitarray
from db import async_session
from models import Availability, Property
from redis_shards import bitmap_from_key, get_redis


async def index_property_to_redis(property: Property):
    redis_client = await get_redis()
    # The bit offset is the ID - 1
    offset = property.id - 1

    async with redis_client.pipeline(transaction=False) as pipe:
        # Atomic bit updates - very fast
        pipe.setbit(f"bitmap:city:{property.city}", offset, 1)
        pipe.setbit(f"bitmap:max_guests:{property.max_guests}", offset, 1)

        # Handle amenities
        for a in ["has_pool", "has_bar", "smoking_allowed"]:
            if getattr(property, a):
                pipe.setbit(f"bitmap:amenity:{a}", offset, 1)

        # GEO is the only "heavy" part
        pipe.geoadd(f"geo:{property.city}", [property.longitude, property.latitude, str(property.id)])

        await pipe.execute()

    # async def index_property_to_redis(property: Property):
    redis_client = await get_redis()

    # City bitmap
    key = f"bitmap:city:{property.city}"
    ba = await bitmap_from_key(key) or bitarray(property.id)
    while len(ba) < property.id:
        ba.append(0)
    ba[property.id - 1] = 1
    await redis_client.set(key, ba.tobytes())

    # Guest count
    key = f"bitmap:max_guests:{property.max_guests}"
    ba = await bitmap_from_key(key) or bitarray(property.id)
    while len(ba) < property.id:
        ba.append(0)
    ba[property.id - 1] = 1
    await redis_client.set(key, ba.tobytes())

    # Amenities
    amenity_fields = [c for c in property.__dict__.keys() if property.__dict__[c] is True]
    for a in amenity_fields:
        key = f"bitmap:amenity:{a}"
        ba = await bitmap_from_key(key) or bitarray(property.id)
        while len(ba) < property.id:
            ba.append(0)
        ba[property.id - 1] = 1
        await redis_client.set(key, ba.tobytes())

    # GEO
    await redis_client.geoadd(f"geo:{property.city}", [float(property.longitude), float(property.latitude), str(property.id)])
