import asyncio
import random

from db import async_session, engine
from events import index_property_to_redis
from faker import Faker
from models import Availability, Base, Property

faker = Faker()


def safe_latitude():
    """Return a float latitude valid for Redis GEO (-85.05112878 → 85.05112878)."""
    return random.uniform(-85.05112878, 85.05112878)


def safe_longitude():
    """Return a float longitude valid for Redis GEO (-180 → 180)."""
    return random.uniform(-180, 180)


async def create_dummy_properties(n=20000):
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)

    async with async_session() as session:
        for i in range(1, n + 1):
            city = random.choice(["NY", "LA", "CHI"])
            p = Property(
                name=faker.company(),
                city=faker.city(),
                latitude=safe_latitude(),
                longitude=safe_longitude(),
                max_guests=random.randint(1, 6),
                has_pool=random.choice([True, False]),
                has_bar=random.choice([True, False]),
                smoking_allowed=random.choice([True, False]),
            )
            session.add(p)
            await session.commit()
            await index_property_to_redis(p)
            if i % 1000 == 1:
                print(f"Added {i} properties")


asyncio.run(create_dummy_properties())
