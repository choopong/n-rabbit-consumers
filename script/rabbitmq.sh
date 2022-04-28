#!/bin/bash
set -e

rabbitmqadmin declare exchange name=some.exchange type=topic
rabbitmqadmin declare queue name=some.queue durable=true
rabbitmqadmin declare binding source="some.exchange" destination_type="queue" destination="some.queue" routing_key="*"
