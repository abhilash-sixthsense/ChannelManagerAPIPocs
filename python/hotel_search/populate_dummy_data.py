import asyncio
import random

from db import async_session, engine
from events import index_property_to_redis
from faker import Faker
from models import Availability, Base, Property
from timer_utils import print_timer, timer_end, timer_start

faker = Faker()
BATCH_SIZE = 3000
REDIS_CONCURRENCY = 5


def safe_latitude():
    """Return a float latitude valid for Redis GEO (-85.05112878 → 85.05112878)."""
    return random.uniform(-85.05112878, 85.05112878)


def safe_longitude():
    """Return a float longitude valid for Redis GEO (-180 → 180)."""
    return random.uniform(-180, 180)


async def create_dummy_properties(n=20000):
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)

    semaphore = asyncio.Semaphore(REDIS_CONCURRENCY)

    async with async_session() as session:
        batch = []
        total = 0
        for i in range(1, n + 1):
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
            batch.append(p)

            # When batch is full → flush
            if len(batch) >= BATCH_SIZE:
                await _flush_batch(session, batch, semaphore)
                total += len(batch)
                batch.clear()

                if total % 1000 == 0:
                    print(f"Added {total} properties")
            if i % 1000 == 1:
                print(f"Added {i} properties")


async def _flush_batch(session, batch, semaphore):
    print(f"\n--- Batch Size: {len(batch)} ---")

    # --- DB TIMER ---
    timer_start("Database Insert")
    session.add_all(batch)
    await session.commit()
    for p in batch:
        await session.refresh(p)
    print_timer("Database Insert")

    # --- REDIS TIMER ---
    timer_start("Redis Indexing")

    async def safe_index(p):
        async with semaphore:
            await index_property_to_redis(p)

    await asyncio.gather(*(safe_index(p) for p in batch))

    print_timer("Redis Indexing")


asyncio.run(create_dummy_properties())
