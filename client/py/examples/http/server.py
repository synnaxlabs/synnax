#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import math
import time

from flask import Flask, Response, jsonify, request


def create_app() -> Flask:
    """Create the Flask app with multiple health check endpoints."""
    app = Flask(__name__)
    start_time = time.time()

    @app.route("/health", methods=["GET"])
    def health() -> tuple[Response, int]:
        """Simple health endpoint. Returns {"status": "ok"}."""
        return jsonify({"status": "ok"}), 200

    @app.route("/health/detailed", methods=["GET"])
    def health_detailed() -> tuple[Response, int]:
        """Detailed health with uptime and metrics."""
        elapsed = time.time() - start_time
        return (
            jsonify(
                {
                    "status": "ok",
                    "uptime_seconds": round(elapsed, 2),
                    "version": "1.0.0",
                    "checks": {
                        "memory": "ok",
                        "disk": "ok",
                        "cpu": "ok",
                    },
                }
            ),
            200,
        )

    @app.route("/health/degraded", methods=["GET"])
    def health_degraded() -> tuple[Response, int]:
        """Simulates a degraded service. Returns 200 but status is "degraded"."""
        elapsed = time.time() - start_time
        return (
            jsonify(
                {
                    "status": "degraded",
                    "uptime_seconds": round(elapsed, 2),
                    "message": "High memory usage detected",
                }
            ),
            200,
        )

    @app.route("/health/failing", methods=["GET"])
    def health_failing() -> tuple[Response, int]:
        """Simulates a failing health check. Returns 503."""
        return (
            jsonify(
                {
                    "status": "error",
                    "message": "Service unavailable",
                }
            ),
            503,
        )

    @app.route("/health/flapping", methods=["GET"])
    def health_flapping() -> tuple[Response, int]:
        """Alternates between healthy and unhealthy every 10 seconds."""
        elapsed = time.time() - start_time
        is_healthy = int(elapsed / 10) % 2 == 0
        if is_healthy:
            return jsonify({"status": "ok"}), 200
        return jsonify({"status": "error"}), 503

    @app.route("/api/v1/status", methods=["GET"])
    def api_status() -> tuple[Response, int]:
        """Alternative status endpoint at a different path."""
        elapsed = time.time() - start_time
        return (
            jsonify(
                {
                    "service": "http-mock",
                    "healthy": True,
                    "uptime": round(elapsed, 2),
                }
            ),
            200,
        )

    @app.route("/api/v1/ping", methods=["GET", "POST"])
    def api_ping() -> tuple[Response, int]:
        """Simple ping/pong endpoint supporting both GET and POST."""
        return jsonify({"response": "pong"}), 200

    @app.route("/api/v1/metrics", methods=["GET"])
    def api_metrics() -> tuple[Response, int]:
        """Simulated metrics endpoint with sine-wave sensor data."""
        elapsed = time.time() - start_time
        return (
            jsonify(
                {
                    "status": "ok",
                    "timestamp": time.time(),
                    "sensors": {
                        f"sensor_{i}": round(math.sin(elapsed + i * 0.5) * 100, 2)
                        for i in range(5)
                    },
                }
            ),
            200,
        )

    @app.route("/api/v1/echo", methods=["POST"])
    def api_echo() -> tuple[Response, int]:
        """Echoes back the request JSON body. Useful for testing POST."""
        body = request.get_json(silent=True) or {}
        return (
            jsonify(
                {
                    "status": "ok",
                    "echo": body,
                }
            ),
            200,
        )

    @app.route("/auth/bearer", methods=["GET"])
    def auth_bearer() -> tuple[Response, int]:
        """Requires Bearer token authentication."""
        auth = request.headers.get("Authorization", "")
        if not auth.startswith("Bearer ") or len(auth) <= 7:
            return jsonify({"status": "error", "message": "Unauthorized"}), 401
        return jsonify({"status": "ok", "auth": "bearer"}), 200

    @app.route("/auth/api-key", methods=["GET"])
    def auth_api_key() -> tuple[Response, int]:
        """Requires API key in X-API-Key header."""
        api_key = request.headers.get("X-API-Key", "")
        if not api_key:
            return jsonify({"status": "error", "message": "Unauthorized"}), 401
        return jsonify({"status": "ok", "auth": "api_key"}), 200

    @app.route("/auth/basic", methods=["GET"])
    def auth_basic() -> tuple[Response, int]:
        """Requires Basic authentication."""
        auth = request.authorization
        if auth is None:
            return jsonify({"status": "error", "message": "Unauthorized"}), 401
        return (
            jsonify(
                {
                    "status": "ok",
                    "auth": "basic",
                    "user": auth.username,
                }
            ),
            200,
        )

    return app


def run_server(
    host: str = "127.0.0.1",
    port: int = 8081,
) -> None:
    """Run the HTTP mock server directly."""
    app = create_app()
    print(f"Starting HTTP mock server on {host}:{port}")
    print()
    print("Available endpoints:")
    print(f"  GET  http://{host}:{port}/health              - Simple health check")
    print(
        f"  GET  http://{host}:{port}/health/detailed     - Detailed health + metrics"
    )
    print(f"  GET  http://{host}:{port}/health/degraded     - Degraded status (200)")
    print(f"  GET  http://{host}:{port}/health/failing      - Failing status (503)")
    print(f"  GET  http://{host}:{port}/health/flapping     - Alternates every 10s")
    print(f"  GET  http://{host}:{port}/api/v1/status       - Alternative status")
    print(f"  GET  http://{host}:{port}/api/v1/ping         - Ping/pong")
    print(f"  POST http://{host}:{port}/api/v1/ping         - Ping/pong (POST)")
    print(f"  GET  http://{host}:{port}/api/v1/metrics      - Sensor metrics")
    print(f"  POST http://{host}:{port}/api/v1/echo         - Echo request body")
    print(f"  GET  http://{host}:{port}/auth/bearer         - Bearer auth required")
    print(f"  GET  http://{host}:{port}/auth/api-key        - API key auth required")
    print(f"  GET  http://{host}:{port}/auth/basic          - Basic auth required")
    print()
    app.run(host=host, port=port, debug=False)


if __name__ == "__main__":
    run_server()
