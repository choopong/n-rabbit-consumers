run:
	docker-compose down && docker-compose up --build --force-recreate

open-rabbitmq:
	open http://localhost:15672/

init-rabbitmq:
	docker exec -i n-rabbit-consumers_rabbitmq_1 sh < ./script/rabbitmq.sh
