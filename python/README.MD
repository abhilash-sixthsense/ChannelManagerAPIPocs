1️⃣ Steps to properly create a local Redis cluster

Let’s assume you have 3 nodes running on ports 6379, 6380, 6381.

Install redis-cli (version ≥ 5) on your host machine.

Connect the nodes and create the cluster:

redis-cli --cluster create \
  127.0.0.1:6379 \
  127.0.0.1:6380 \
  127.0.0.1:6381 \
  --cluster-replicas 0


--cluster-replicas 0 → no replicas (for local testing).

You can add replicas later for HA.

It will ask you to confirm → type yes.

Verify cluster is healthy:

redis-cli -c -p 6379 cluster info
redis-cli -c -p 6379 cluster nodes


You should see all 3 nodes and assigned slots.

Now CLUSTER SLOTS will return proper slot mapping.

2️⃣ Notes for Docker Compose

The Redis nodes in your docker-compose.yml don’t automatically know each other. For local dev:

Make sure you use host networking or Docker service names.

Or manually run redis-cli --cluster create with container IPs.

Example (with container names):

docker network inspect bridge
# get container IPs
docker exec -it redis-node-1 redis-cli --cluster create \
  172.18.0.2:6379 172.18.0.3:6379 172.18.0.4:6379 --cluster-replicas 0

3️⃣ Quick local testing alternative

If you just want async bitmaps + geo for local testing, you don’t need cluster:

Start a single Redis node on port 6379

Update your redis_shards.py to point all cities to the same node.

REDIS_SHARDS: Dict[str, str] = {
    "NY": "redis://localhost:6379/0",
    "LA": "redis://localhost:6379/0",
    "CHI": "redis://localhost:6379/0",
    "DEFAULT": "redis://localhost:6379/0"
}


Once you’re ready for HA / billions of queries, initialize the cluster properly.

```sudo apt install redis-tools``` for redis-cli