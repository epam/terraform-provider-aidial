## ADDED Requirements

### Requirement: Reach a Core running outside the devcontainer
The devcontainer SHALL map `host.docker.internal` to the host so tests can reach a Core started on the host, and SHALL NOT include a Docker client or daemon or run privileged.

#### Scenario: Host Core reachable
- **WHEN** a Core is running on the host and the devcontainer has been rebuilt
- **THEN** `curl http://host.docker.internal:8080/health` from the devcontainer gets an HTTP answer
