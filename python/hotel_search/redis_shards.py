from typing import Optional

from redis.asyncio.cluster import RedisCluster,ClusterNode

_redis_cluster: Optional[RedisCluster] = None


async def get_redis() -> RedisCluster:
    global _redis_cluster

    if _redis_cluster is None:
        _redis_cluster = RedisCluster(
            startup_nodes=[
                ClusterNode("localhost", 6379),
                ClusterNode("localhost", 6380),
                ClusterNode("localhost", 6381),
            ],
            decode_responses=False,  # keep bytes (needed for bitmap)
            read_from_replicas=False,
        )

    return _redis_cluster


async def bitmap_from_key(key: str):
    redis_client = await get_redis()

    data = await redis_client.get(key)
    if not data:
        return None

    from bitarray import bitarray

    ba = bitarray()
    ba.frombytes(data)
    return ba
