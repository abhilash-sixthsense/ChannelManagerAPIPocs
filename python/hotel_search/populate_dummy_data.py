import asyncio, random
from faker import Faker
from db import async_session, engine
from .models import Base, Property, Availability
from .events import index_property_to_redis

faker = Faker()

async def create_dummy_properties(n=20000):
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)

    async with async_session() as session:
        for i in range(1, n+1):
            city = random.choice(["NY", "LA", "CHI"])
            p = Property(
                name=faker.company(),
                city=city,
                latitude=faker.latitude(),
                longitude=faker.longitude(),
                max_guests=random.randint(1,6),
                has_pool=random.choice([True,False]),
                has_bar=random.choice([True,False]),
                smoking_allowed=random.choice([True,False])
            )
            session.add(p)
            await session.commit()
            await index_property_to_redis(p)

asyncio.run(create_dummy_properties())
