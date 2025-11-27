# 🛰️ Orbital Net

### **Distributed Satellite Communication Simulator**

**Orbital Net** is a Kubernetes-native distributed system designed to simulate, visualize, and test communication protocols in a Low-Earth-Orbit (LEO) satellite network.

The system is built using a **microservices architecture** with **gRPC-only communication**, implementing dynamic routing, data persistence, automatic horizontal scaling, and real-time orbital physics.

---

## 🚀 System Architecture

Orbital Net is composed of **5 microservices**, deployed as independent Kubernetes workloads:

### **Microservices Overview**

| Service                    | K8s Type     | Description                                                                                                                                     |
| -------------------------- | ------------ | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| **sim-service**            | ClusterIP    | Physics Engine. Computes Keplerian orbits and visibility matrices in real time (20 Hz). Emits a continuous gRPC stream.                         |
| **pathfinder-service**     | ClusterIP    | Routing Brain. Builds a dynamic graph and computes optimal routes using a **Widest Path algorithm**, prioritizing higher-generation satellites. |
| **communications-service** | ClusterIP    | Message Orchestrator. Implements Store-and-Forward, manages queues, handles overload, and retries delivery. Includes HorizontalPodAutoscaler.   |
| **storage-service**        | ClusterIP    | Persistence Layer. Stores message logs using an append-only on-disk structure.                                                                  |
| **control-center**         | LoadBalancer | API Gateway & Web UI. Visualizes the network, allows spawning satellites, and sending commands.                                                 |

---

## 📋 Prerequisites

Required tools:

* **Go 1.24+**
* **Docker**
* **Minikube**
* **Kubectl**
* **K6** (for load testing)
* **Make**

---

# ⚡ Quick Start Deployment

You can deploy the entire system with just a few commands.

---
### **1. Clean Previous Deployments**

```bash
make clean
```

---

### **2. Generate gRPC Code (Protobuf)**

Runs `protoc` for all proto files in:

* `pkg/proto/common`
* `pkg/proto/sim`
* `pkg/proto/pathfinder`
* `pkg/proto/communications`
* `pkg/proto/storage`

```bash
make proto
```

---
## **3. Prepare the Kubernetes Cluster**

Enable the Kubernetes metrics server (required for HPA):

```bash
make setup-cluster
```

---

## **4. Deploy Everything 🪄**

This builds Docker images and applies all Kubernetes manifests:

```bash
make build-all
```

---

### **5. Expose the UI (Minikube Tunnel)**

Run in a separate terminal:

```bash
make tunnel
```

Retrieve the external IP:

```bash
make get-ip
```

Open in the browser:

```
http://<EXTERNAL-IP>:8080
```

---


# 🧪 Load Testing & Auto-Scaling

Orbital Net includes a full K6 load-testing setup to stress the communication network and trigger autoscaling.

---

## **Run a Load Test**

Specify **origin** and **destination** node IDs:

```bash
make test origin=<origin_id> dest=<dest_id>
```

Example:

```bash
make test origin=gn_4c3e3 dest=on_8bbd4
```

This automatically:

* Kills old port-forward sessions
* Opens a new tunnel to `communications-service`
* Runs K6 inside Docker
* Closes the port-forward when done

---

## **Monitor Auto-Scaling**

In another terminal:

```bash
make monitor-scaling
```

You’ll see pods scaling dynamically (e.g., 1 → 10 replicas).

---

# 🛑 Stopping the System

Stop the Minikube tunnel:

```
Ctrl + C
```

Clean all deployments:

```bash
make clean
```

Or reset (clean + deploy everything):

```bash
make reset
```
