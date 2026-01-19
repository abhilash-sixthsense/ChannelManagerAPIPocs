from typing import Optional

from redis.asyncio.cluster import ClusterNode, RedisCluster

_redis_cluster: Optional[RedisCluster] = None


def remap_docker_ip_to_localhost(address):
    # address is (ip, port)
    # We ignore the internal IP and return localhost,
    # but keep the port (as long as they are mapped 1:1)
    return ("127.0.0.1", address[1])


async def get_redis() -> RedisCluster:
    global _redis_cluster

    if _redis_cluster is None:
        _redis_cluster = RedisCluster(
            startup_nodes=[
                ClusterNode("127.0.0.1", 6379),
                ClusterNode("127.0.0.1", 6380),
                ClusterNode("127.0.0.1", 6381),
            ],
            decode_responses=False,  # keep bytes (needed for bitmap)
            read_from_replicas=False,
            require_full_coverage=False,
            address_remap=remap_docker_ip_to_localhost,
            max_connections=500,  # Allow more simultaneous connections
            socket_keepalive=True,  # Keep TCP connections alive
            socket_connect_timeout=5,  # Fail fast on bad nodes
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
