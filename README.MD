# Orbital Net — Satellite Communications Simulator

**Orbital Net** is a simulation project for satellite-based communication networks. Its goal is to emulate how ground nodes communicate through orbital nodes, replicating the behavior of a dynamic satellite network.

## Main features
- **Dynamic topology**: simulates satellite orbits and planetary rotation.
- **Network protocols**: models node communication and basic protocol behavior.
- **Pathfinding**: routing algorithms that adapt to changing topology.
- **Port speed & bandwidth**: simulate bandwidth constraints and link capacity.
- **Hybrid nodes**: supports ground nodes and orbital relays.

## Project structure
```
satellite-coms/orbital-net/
├── simulator/        # orbit, position and motion simulation
├── communications/   # node communication, link models, protocols
├── pathfinder/       # routing and search algorithms
├── controlcenter/    # monitoring, metrics and visualization hooks
├── database/         # persistence and test datasets
├── examples/         # sample scenarios and configs
└── README.md
```

## Quick start
1. Clone the repository:
```
git clone https://github.com/<user>/satellite-coms.git
cd satellite-coms/orbital-net
```
2. Create a virtual environment (optional) and install dependencies:
```
# for Python components
python -m venv venv
source venv/bin/activate    # macOS / Linux
venv\Scripts\activate     # Windows PowerShell

pip install -r requirements.txt
```
3. Run services or scripts independently:
```
# example, if modules are Python-based
python simulator/main.py
python communications/main.py
python pathfinder/main.py
python controlcenter/main.py
```
Alternatively use `Makefile` / `docker-compose` if available.

## Simulation flow (high level)
1. Simulator generates orbital node positions and visibility windows between ground and orbital nodes.
2. Ground nodes request connections; pathfinder computes candidate routes across satellites.
3. Communications module enforces latency, bandwidth and link constraints.
4. Controlcenter collects metrics, logs and optionally visualizes the network.

## Configuration & inputs
- Satellite orbit parameters (altitude, inclination, period).
- Ground node coordinates and traffic demand.
- Link bandwidth / latency models and port speeds.
- Pathfinding strategy and protocol timeouts.

## Recommended technologies
- **Python / Go** for simulation and services.
- **Redis** for ephemeral state and fast caching.
- **PostgreSQL** for persistent results and scenarios.
- **Docker / Kubernetes** for containerized deployment.
- **Prometheus / Grafana** for monitoring.

## Performance & scaling notes
- Simulating many satellites and long time horizons can be CPU and memory intensive.
- Consider running heavy simulations in batches or with distributed workers.
- Use approximate/heuristic pathfinding (A*, greedy) for larger topologies.

## Tests & examples
Include small reproducible scenarios in `examples/`:
- Single-satellite relay test.
- Low-earth constellation with 4 satellites.
- Failure and recovery scenarios.

## Roadmap / future work
- Real-time visualization dashboard.
- More realistic RF link models (SNR, fading).
- Integrated packet-level simulation for throughput studies.
- Fault injection and resiliency experiments.

## Contributing
Please open issues, fork the repo and submit pull requests. Provide test cases for new features and keep changes modular.

## License
This project is available under the MIT license. See `LICENSE` for details.
