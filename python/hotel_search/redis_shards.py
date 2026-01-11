import aioredis
from typing import Dict

REDIS_SHARDS: Dict[str, str] = {
    "NY": "redis://localhost:6379/0",
    "LA": "redis://localhost:6380/0",
    "CHI": "redis://localhost:6381/0",
    "DEFAULT": "redis://localhost:6379/0"
}

_redis_clients = {}

async def get_redis_for_city(city: str) -> aioredis.Redis:
    prefix = city[:3].upper()
    url = REDIS_SHARDS.get(prefix, REDIS_SHARDS["DEFAULT"])
    if url not in _redis_clients:
        _redis_clients[url] = await aioredis.from_url(url)
    return _redis_clients[url]

async def bitmap_from_key(redis_client, key: str):
    data = await redis_client.get(key)
    if not data:
        return None
    from bitarray import bitarray
    ba = bitarray()
    ba.frombytes(data)
    return ba
