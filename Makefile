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
	docker run -d --network="my-blog" -e BOT_PROXY=${BOT_PROXY} -e BOT_CHAT_ID=${BOT_CHAT_ID} -e BOT_TOKEN=${BOT_TOKEN} -e BOT_PORT=:${BOT_PORT} -p ${BOT_PORT}:${BOT_PORT} --name my-blog-chat my-blog-chat
