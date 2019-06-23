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
	docker run -d --network="my-blog" -e BOT_PROXY=${BOT_PROXY} -e BOT_CHAT_ID=${BOT_CHAT_ID} -e BOT_TOKEN=${BOT_TOKEN} -e BOT_PORT=:${BOT_PORT} -e BOT_AMQP_HOST=${BOT_AMQP_HOST} -e BOT_AMQP_USER=${BOT_AMQP_USER} -e BOT_AMQP_PASSWORD=${BOT_AMQP_PASSWORD} -p ${BOT_PORT}:${BOT_PORT} --name my-blog-chat my-blog-chat
run-dev:
	BOT_PROXY=${BOT_PROXY} BOT_CHAT_ID=${BOT_CHAT_ID} BOT_TOKEN=${BOT_TOKEN} BOT_PORT=:${BOT_PORT} BOT_AMQP_HOST=${BOT_AMQP_HOST} BOT_AMQP_USER=${BOT_AMQP_USER} BOT_AMQP_PASSWORD=${BOT_AMQP_PASSWORD} ./my-blog-chat
