include .env
export $(shell sed 's/=.*//' .env)

build:
	docker build -t my-blog-chat .
kill:
	docker kill my-blog-chat
rm:
	docker rm my-blog-chat
rmi:
	docker rmi my-blog-chat
run:
	docker run -d --network="my-blog" -e BOT_LOG_PASSWORD=${BOT_LOG_PASSWORD} -e BOT_LOG_EMAIL=${BOT_LOG_EMAIL} -e BOT_PROXY=${BOT_PROXY} -e BOT_CHAT_ID=${BOT_CHAT_ID} -e BOT_TOKEN=${BOT_TOKEN} -e BOT_PORT=${BOT_PORT} -e BOT_AMQP_HOST=${BOT_AMQP_HOST} -e BOT_AMQP_USER=${BOT_AMQP_USER} -e BOT_AMQP_PASSWORD=${BOT_AMQP_PASSWORD} -e BOT_AMQP_QUEUE=${BOT_AMQP_QUEUE} -p ${BOT_PORT}:${BOT_PORT} --name my-blog-chat my-blog-chat
run-dev:
	./my-blog-chat
