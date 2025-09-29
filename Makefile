simulator-service:
	go run simulator/main.go;

communications-service:
	go run communications/main.go;

pathfinder-service:
	go run pathfinder/main.go;

controlcenter-service:
	go run controlcenter/main.go;

database-service:
	go run database/main.go;

run-consul-dev-server:
	docker run -d -p 8500:8500 -p 8600:8600/udp --name=dev-consul consul:1.15.4 agent -server -ui -node=server-1 -bootstrap-expect=1 -client=0.0.0.0

run-redis:
	docker run -d --name redis -p 6379:6379 redis
