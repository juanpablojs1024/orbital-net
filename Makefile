GOPATH := $(shell go env GOPATH)
PROTOC_GEN_GO := $(GOPATH)/bin/protoc-gen-go
PROTOC_GEN_GO_GRPC := $(GOPATH)/bin/protoc-gen-go-grpc

parallel ?= false

setup-cluster:
	minikube addons enable metrics-server
	@echo "✅ Metrics server enabled. HPA will work after a few seconds."

proto:
	# 1. Common
	protoc -I . \
		--plugin=protoc-gen-go=$(PROTOC_GEN_GO) \
		--go_out=. --go_opt=paths=source_relative \
		pkg/proto/common/model.proto

	# 2. Sim-Service
	protoc -I . \
		--plugin=protoc-gen-go=$(PROTOC_GEN_GO) \
		--plugin=protoc-gen-go-grpc=$(PROTOC_GEN_GO_GRPC) \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		pkg/proto/sim/sim-service.proto

	# 3. Pathfinder
	protoc -I . \
		--plugin=protoc-gen-go=$(PROTOC_GEN_GO) \
		--plugin=protoc-gen-go-grpc=$(PROTOC_GEN_GO_GRPC) \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		pkg/proto/pathfinder/pathfinder-service.proto
		
	# 4. Communications
	protoc -I . \
		--plugin=protoc-gen-go=$(PROTOC_GEN_GO) \
		--plugin=protoc-gen-go-grpc=$(PROTOC_GEN_GO_GRPC) \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		pkg/proto/communications/communications-service.proto

	# 5. Storage
	protoc -I . \
		--plugin=protoc-gen-go=$(PROTOC_GEN_GO) \
		--plugin=protoc-gen-go-grpc=$(PROTOC_GEN_GO_GRPC) \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		pkg/proto/storage/storage-service.proto
	
$(PROTOC_GEN_GO):
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

$(PROTOC_GEN_GO_GRPC):
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

build-sim-service:
	eval $$(minikube docker-env) && docker build -t sim-service:latest -f services/sim-service/Dockerfile .
	kubectl apply -f k8s/sim-deployment.yaml
	kubectl rollout restart deployment sim-deployment
	kubectl delete pod sim-client --ignore-not-found=true

build-pathfinder-service:
	eval $$(minikube docker-env) && docker build -t pathfinder-service:latest -f services/pathfinder-service/Dockerfile .
	kubectl apply -f k8s/pathfinder-deployment.yaml
	# Apply HPA for Pathfinder
	kubectl apply -f k8s/hpa-pathfinder.yaml
	kubectl rollout restart deployment pathfinder-deployment

build-communications-service:
	eval $$(minikube docker-env) && docker build -t communications-service:latest -f services/communications-service/Dockerfile .
	kubectl apply -f k8s/communications-deployment.yaml
	# Apply HPA for Communications
	kubectl apply -f k8s/hpa-communications.yaml
	kubectl rollout restart deployment communications-deployment

build-storage-service:
	eval $$(minikube docker-env) && docker build -t storage-service:latest -f services/storage-service/Dockerfile .
	kubectl apply -f k8s/storage-deployment.yaml
	kubectl rollout restart deployment storage-deployment

build-control-center-service:
	eval $$(minikube docker-env) && docker build -t control-center-service:latest -f services/control-center-service/Dockerfile .
	kubectl apply -f k8s/control-center-deployment.yaml
	kubectl rollout restart deployment control-center-deployment

build-all: build-sim-service build-storage-service build-pathfinder-service build-communications-service build-control-center-service

port-forward-communications:
	kubectl port-forward svc/communications-server 50053:50053

port-forward-control:
	kubectl port-forward svc/control-center-server 8080:8080

run-test:
	@echo "🔧 Ensuring Environment is ready..."
	minikube addons enable metrics-server
	
	@echo "🧹 Cleaning up any zombie tunnels on port 50053..."
	# Intentamos matar cualquier cosa en el puerto 50053 (más preciso que pkill por nombre)
	-fuser -k 50053/tcp >/dev/null 2>&1 || true
	# Fallback por si fuser no está instalado: pkill con regex que evita matarse a sí mismo
	-pkill -f "port-forward svc/communications-server 5005[3]" || true
	@sleep 2
	
	@echo "🚀 Starting Background Tunnel..."
	# CORRECCIÓN 1: --address 0.0.0.0 permite conexiones desde Docker
	@kubectl port-forward --address 0.0.0.0 svc/communications-server 50053:50053 > /dev/null 2>&1 & echo $$! > .pf_pid
	@echo "⏳ Waiting 5s for tunnel to stabilize..."
	@sleep 5
	
	@echo "🔥 Launching K6 Attack from $(origin) to $(dest)..."
	# CORRECCIÓN 2: Usamos localhost directo con network host, mucho más estable en Linux
	# PASAMOS ORIGIN Y DEST COMO VARIABLES DE ENTORNO (-e)
	-docker run --rm -i \
		--network host \
		-v $$(pwd):/app \
		-w /app \
		-e TARGET=localhost:50053 \
		-e ORIGIN=$(origin) \
		-e DESTINATION=$(dest) \
		grafana/k6 run k6-load-test.js
	
	@echo "🧹 Cleaning up tunnel..."
	-kill $$(cat .pf_pid) 2>/dev/null || true
	-rm .pf_pid 2>/dev/null || true
	@echo "✅ Test sequence complete."

monitor:
	kubectl get hpa -w

clean:
	@echo "🧹 Cleaning K8s resources..."
	# Deployments & Services
	kubectl delete -f k8s/sim-deployment.yaml --ignore-not-found=true
	kubectl delete -f k8s/pathfinder-deployment.yaml --ignore-not-found=true
	kubectl delete -f k8s/communications-deployment.yaml --ignore-not-found=true
	kubectl delete -f k8s/storage-deployment.yaml --ignore-not-found=true
	kubectl delete -f k8s/control-center-deployment.yaml --ignore-not-found=true
	# HPAs
	kubectl delete -f k8s/hpa-communications.yaml --ignore-not-found=true
	kubectl delete -f k8s/hpa-pathfinder.yaml --ignore-not-found=true
	# Pods
	kubectl delete pod sim-client --ignore-not-found=true
	kubectl delete pod pathfinder-client --ignore-not-found=true
	kubectl delete pod communications-client --ignore-not-found=true
	kubectl delete pod storage-client --ignore-not-found=true
	kubectl delete pod control-center-client --ignore-not-found=true
	@echo "✨ Clean complete."

reset: clean build-all

tunnel:
	minikube tunnel

get-external-ip:
	kubectl get svc control-center

deploy-port: clean build-all
	@echo "🔍 Asegurando que Minikube esté iniciado..."
	@minikube status >/dev/null 2>&1 || minikube start

	@echo "🚀 Desplegando Orbital Net..."

	@echo "⏳ Esperando a que el Control Center arranque completamente..."
	@kubectl wait --for=condition=available deployment/control-center-deployment --timeout=120s > /dev/null

	@echo "🔌 Iniciando Port-Forward (localhost:8080 -> k8s:8080)..."

	@( \
		kubectl port-forward --address 0.0.0.0 svc/control-center 8080:8080 > /dev/null 2>&1 & \
		PF_PID=$$!; \
		echo "⏳ Estableciendo conexión..."; \
		sleep 3; \
		if ! ps -p $$PF_PID > /dev/null; then \
			echo ""; echo "❌ Error: El port-forward falló inmediatamente."; \
			echo "🔍 Tip: Verifica si el puerto 8080 ya está ocupado."; \
			exit 1; \
		fi; \
		echo "✅ ¡Sistema listo!"; \
		echo "🌍 URL: http://localhost:8080"; \
		xdg-open "http://localhost:8080" 2>/dev/null || \
		open "http://localhost:8080" 2>/dev/null || \
		echo "⚠️ Abre el navegador manualmente."; \
		echo "🔴 El sistema está activo. MANTÉN ESTA TERMINAL ABIERTA."; \
		echo "🔴 Presiona Ctrl+C para detener."; \
		wait $$PF_PID; \
	)

	@true