1️⃣ Steps to properly create a local Redis cluster

Let’s assume you have 3 nodes running on ports 6379, 6380, 6381.

### Install redis-cli (version ≥ 5) on your host machine.

```
sudo apt install redis-tools 
```

### Connect the nodes and create the cluster:

```

docker exec hotel_search-redis-node-1-1 redis-cli --cluster create \
  hotel_search-redis-node-1-1:6379 \
  hotel_search-redis-node-2-1:6380 \
  hotel_search-redis-node-3-1:6381 \
  --cluster-replicas 0 \
  --cluster-yes

  ```


```
--cluster-replicas 0 → no replicas (for local testing).
```

You can add replicas later for HA.

It will ask you to confirm → type yes.

Verify cluster is healthy:

redis-cli -c -p 6379 cluster info
redis-cli -c -p 6379 cluster nodes

