### === PATHS & TOOLS ======================================================

GOPATH := $(shell go env GOPATH)
PROTOC_GEN_GO := $(GOPATH)/bin/protoc-gen-go
PROTOC_GEN_GO_GRPC := $(GOPATH)/bin/protoc-gen-go-grpc


### === CLUSTER SETUP ======================================================

setup-cluster:
	minikube addons enable metrics-server
	@echo "✅ Metrics server enabled."


### === PROTO GENERATION ===================================================

proto: $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC)
	@echo "🔧 Generating Protobuf code..."

	# Common
	protoc -I . \
		--plugin=protoc-gen-go=$(PROTOC_GEN_GO) \
		--go_out=. --go_opt=paths=source_relative \
		pkg/proto/common/model.proto

	# Sim-Service
	protoc -I . \
		--plugin=protoc-gen-go=$(PROTOC_GEN_GO) \
		--plugin=protoc-gen-go-grpc=$(PROTOC_GEN_GO_GRPC) \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		pkg/proto/sim/sim-service.proto

	# Pathfinder
	protoc -I . \
		--plugin=protoc-gen-go=$(PROTOC_GEN_GO) \
		--plugin=protoc-gen-go-grpc=$(PROTOC_GEN_GO_GRPC) \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		pkg/proto/pathfinder/pathfinder-service.proto

	# Communications
	protoc -I . \
		--plugin=protoc-gen-go=$(PROTOC_GEN_GO) \
		--plugin=protoc-gen-go-grpc=$(PROTOC_GEN_GO_GRPC) \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		pkg/proto/communications/communications-service.proto

	# Storage
	protoc -I . \
		--plugin=protoc-gen-go=$(PROTOC_GEN_GO) \
		--plugin=protoc-gen-go-grpc=$(PROTOC_GEN_GO_GRPC) \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		pkg/proto/storage/storage-service.proto

	@echo "✅ Protobuf generation complete."

$(PROTOC_GEN_GO):
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

$(PROTOC_GEN_GO_GRPC):
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest


### === BUILD & DEPLOY SERVICES ==========================================

define build_service
	eval $$(minikube docker-env) && docker build -t $(1):latest -f services/$(1)-service/Dockerfile .
	kubectl apply -f k8s/$(1)-deployment.yaml
	kubectl rollout restart deployment $(1)-deployment
endef

build-sim:
	$(call build_service,sim)

build-pathfinder:
	$(call build_service,pathfinder)
	kubectl apply -f k8s/hpa-pathfinder.yaml

build-communications:
	$(call build_service,communications)
	kubectl apply -f k8s/hpa-communications.yaml

build-storage:
	$(call build_service,storage)

build-control-center:
	$(call build_service,control-center)


### === BUILD ALL =========================================================

build-all: build-sim build-storage build-pathfinder build-communications build-control-center
	@echo "🚀 All services built and deployed."


### === ACCESS ============================================================

tunnel:
	minikube tunnel

get-ip:
	kubectl get svc control-center

### === MONITOR ===========================================================

monitor-scaling:
	kubectl get hpa -w

monitor-communications:
	kubectl logs -l app=orbital-communications -f --max-log-requests=50

### === LOAD TEST ==========================================================

test:
	@echo "🔥 Running load test from ORIGIN=$(origin) to DESTINATION=$(dest)..."

	# Cleanup stale tunnels
	-fuser -k 50053/tcp >/dev/null 2>&1 || true
	-pkill -f "port-forward svc/communications-server 5005[3]" || true
	@sleep 2

	# Start tunnel
	@kubectl port-forward --address 0.0.0.0 svc/communications-server 50053:50053 > /dev/null 2>&1 & echo $$! > .pf_pid
	@sleep 5

	# Run K6
	-docker run --rm -i \
		--network host \
		-v $$(pwd):/app \
		-w /app \
		-e TARGET=localhost:50053 \
		-e ORIGIN=$(origin) \
		-e DESTINATION=$(dest) \
		grafana/k6 run k6-load-test.js

	# Stop tunnel
	-kill $$(cat .pf_pid) 2>/dev/null || true
	-rm .pf_pid 2>/dev/null || true

	@echo "✅ Load test complete."


### === CLEAN =============================================================

clean:
	@echo "🧹 Cleaning Kubernetes resources..."

	kubectl delete -f k8s/sim-deployment.yaml --ignore-not-found=true
	kubectl delete -f k8s/pathfinder-deployment.yaml --ignore-not-found=true
	kubectl delete -f k8s/communications-deployment.yaml --ignore-not-found=true
	kubectl delete -f k8s/storage-deployment.yaml --ignore-not-found=true
	kubectl delete -f k8s/control-center-deployment.yaml --ignore-not-found=true

	# HPAs
	kubectl delete -f k8s/hpa-communications.yaml --ignore-not-found=true
	kubectl delete -f k8s/hpa-pathfinder.yaml --ignore-not-found=true

	# Clients
	kubectl delete pod -l app=orbital-sim-client --ignore-not-found=true
	kubectl delete pod -l app=orbital-pathfinder-client --ignore-not-found=true
	kubectl delete pod -l app=orbital-communications-client --ignore-not-found=true
	kubectl delete pod -l app=orbital-storage-client --ignore-not-found=true
	kubectl delete pod -l app=orbital-control-client --ignore-not-found=true

	@echo "✨ Clean complete."

reset: clean build-all
