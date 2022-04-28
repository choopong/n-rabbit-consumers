#!/bin/bash
set -e

# rabbitmq-plugins enable rabbitmq_sharding
# rabbitmqctl set_policy some.exchange.shard "^some.exchange$" '{"shards-per-node": 2}'
# rabbitmqadmin declare exchange name=some.exchange type=x-modulus-hash
# rabbitmq-plugins enable rabbitmq_consistent_hash_exchange
rabbitmqadmin declare exchange name=some.exchange type=topic
rabbitmqadmin declare queue name=some.queue durable=true
rabbitmqadmin declare binding source="some.exchange" destination_type="queue" destination="some.queue" routing_key="*"

rabbitmqadmin declare exchange name=worker.exchange.1 type=topic
rabbitmqadmin declare queue name=worker.queue.1 durable=true
rabbitmqadmin declare binding source="worker.exchange.1" destination_type="queue" destination="worker.queue.1" routing_key="*"

rabbitmqadmin declare exchange name=worker.exchange.2 type=topic
rabbitmqadmin declare queue name=worker.queue.2 durable=true
rabbitmqadmin declare binding source="worker.exchange.2" destination_type="queue" destination="worker.queue.2" routing_key="*"