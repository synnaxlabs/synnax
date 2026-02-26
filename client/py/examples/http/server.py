#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import argparse
import datetime
import ipaddress
import math
import os
import tempfile
import time

from flask import Flask, Response, jsonify, request


AUTH_CREDENTIALS = {
    "username": "admin",
    "password": "password",
    "token": "test-token",
    "api_key": "test-api-key",
}


def _check_basic_auth() -> tuple[Response, int] | None:
    auth = request.authorization
    if (
        auth is None
        or auth.username != AUTH_CREDENTIALS["username"]
        or auth.password != AUTH_CREDENTIALS["password"]
    ):
        return jsonify({"status": "error", "message": "Unauthorized"}), 401
    return None


def _check_bearer_auth() -> tuple[Response, int] | None:
    header = request.headers.get("Authorization", "")
    if not header.startswith("Bearer ") or header[7:] != AUTH_CREDENTIALS["token"]:
        return jsonify({"status": "error", "message": "Unauthorized"}), 401
    return None


def _check_api_key_auth() -> tuple[Response, int] | None:
    key = request.headers.get("X-API-Key", "")
    if key != AUTH_CREDENTIALS["api_key"]:
        return jsonify({"status": "error", "message": "Unauthorized"}), 401
    return None


AUTH_CHECKERS = {
    "basic": _check_basic_auth,
    "bearer": _check_bearer_auth,
    "api-key": _check_api_key_auth,
}


def create_app(auth_type: str = "none") -> Flask:
    """Create the Flask app with multiple health check endpoints."""
    app = Flask(__name__)
    start_time = time.time()

    if auth_type != "none":
        checker = AUTH_CHECKERS[auth_type]

        @app.before_request
        def enforce_auth() -> tuple[Response, int] | None:
            return checker()

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

    @app.route("/api/v1/data", methods=["GET"])
    def api_data() -> tuple[Response, int]:
        """Returns mixed data types. Useful for testing dataType selection.

        Supports query parameters:
          ?scale=<float>  - multiplier for numeric values (default 1.0)
          ?include=<csv>  - comma-separated fields to include
        """
        elapsed = time.time() - start_time
        scale = float(request.args.get("scale", 1.0))
        include = request.args.get("include", "").split(",")
        include_all = include == [""]
        data: dict = {
            "timestamp": time.time(),
        }
        if include_all or "temperature" in include:
            data["temperature"] = round(math.sin(elapsed * 0.1) * 25 * scale + 20, 2)
        if include_all or "pressure" in include:
            data["pressure"] = round(math.cos(elapsed * 0.05) * 10 * scale + 1013, 2)
        if include_all or "humidity" in include:
            data["humidity"] = int(math.sin(elapsed * 0.2) * 30 * scale + 50)
        if include_all or "label" in include:
            data["label"] = "normal" if data.get("humidity", 50) < 70 else "high"
        if include_all or "active" in include:
            data["active"] = int(elapsed) % 60 < 50
        if include_all or "count" in include:
            data["count"] = int(elapsed)
        return jsonify(data), 200

    @app.route("/api/v1/headers", methods=["GET"])
    def api_headers() -> tuple[Response, int]:
        """Echoes back all request headers as JSON. Tests per-endpoint headers."""
        headers = {k: v for k, v in request.headers}
        return jsonify({"headers": headers}), 200

    @app.route("/api/v1/query", methods=["GET"])
    def api_query() -> tuple[Response, int]:
        """Echoes back all query parameters as JSON. Tests query param config."""
        params = dict(request.args)
        return jsonify({"params": params, "count": len(params)}), 200

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


def _generate_self_signed_cert(cert_dir: str) -> tuple[str, str]:
    """Generate a self-signed certificate and private key."""
    from cryptography import x509
    from cryptography.hazmat.primitives import hashes, serialization
    from cryptography.hazmat.primitives.asymmetric import rsa
    from cryptography.x509.oid import NameOID

    key = rsa.generate_private_key(public_exponent=65537, key_size=2048)
    subject = issuer = x509.Name(
        [x509.NameAttribute(NameOID.COMMON_NAME, "localhost")]
    )
    cert = (
        x509.CertificateBuilder()
        .subject_name(subject)
        .issuer_name(issuer)
        .public_key(key.public_key())
        .serial_number(x509.random_serial_number())
        .not_valid_before(datetime.datetime.now(datetime.timezone.utc))
        .not_valid_after(
            datetime.datetime.now(datetime.timezone.utc)
            + datetime.timedelta(days=365)
        )
        .add_extension(
            x509.SubjectAlternativeName(
                [
                    x509.DNSName("localhost"),
                    x509.IPAddress(ipaddress.IPv4Address("127.0.0.1")),
                ]
            ),
            critical=False,
        )
        .sign(key, hashes.SHA256())
    )

    cert_path = os.path.join(cert_dir, "cert.pem")
    key_path = os.path.join(cert_dir, "key.pem")
    with open(cert_path, "wb") as f:
        f.write(cert.public_bytes(serialization.Encoding.PEM))
    with open(key_path, "wb") as f:
        f.write(
            key.private_bytes(
                serialization.Encoding.PEM,
                serialization.PrivateFormat.TraditionalOpenSSL,
                serialization.NoEncryption(),
            )
        )
    return cert_path, key_path


def run_server(
    host: str = "127.0.0.1",
    port: int = 8081,
    https: bool = False,
    auth: str = "none",
) -> None:
    """Run the HTTP mock server directly."""
    app = create_app(auth_type=auth)
    scheme = "https" if https else "http"

    print(f"Starting {'HTTPS' if https else 'HTTP'} mock server on {host}:{port}")
    if auth != "none":
        print(f"Auth middleware: {auth}")
        if auth == "basic":
            print(
                f"  Credentials: {AUTH_CREDENTIALS['username']}"
                f":{AUTH_CREDENTIALS['password']}"
            )
        elif auth == "bearer":
            print(f"  Token: {AUTH_CREDENTIALS['token']}")
        elif auth == "api-key":
            print(f"  API Key: {AUTH_CREDENTIALS['api_key']}")
    print()
    print("Available endpoints:")
    print(f"  GET  {scheme}://{host}:{port}/health              - Simple health check")
    print(
        f"  GET  {scheme}://{host}:{port}/health/detailed"
        "     - Detailed health + metrics"
    )
    print(
        f"  GET  {scheme}://{host}:{port}/health/degraded"
        "     - Degraded status (200)"
    )
    print(
        f"  GET  {scheme}://{host}:{port}/health/failing"
        "      - Failing status (503)"
    )
    print(
        f"  GET  {scheme}://{host}:{port}/health/flapping"
        "     - Alternates every 10s"
    )
    print(
        f"  GET  {scheme}://{host}:{port}/api/v1/status"
        "       - Alternative status"
    )
    print(f"  GET  {scheme}://{host}:{port}/api/v1/ping         - Ping/pong")
    print(f"  POST {scheme}://{host}:{port}/api/v1/ping         - Ping/pong (POST)")
    print(f"  GET  {scheme}://{host}:{port}/api/v1/metrics      - Sensor metrics")
    print(
        f"  POST {scheme}://{host}:{port}/api/v1/echo"
        "         - Echo request body"
    )
    print(f"  GET  {scheme}://{host}:{port}/api/v1/data         - Mixed data types")
    print(
        f"  GET  {scheme}://{host}:{port}/api/v1/headers"
        "      - Echo request headers"
    )
    print(
        f"  GET  {scheme}://{host}:{port}/api/v1/query"
        "        - Echo query parameters"
    )
    print(
        f"  GET  {scheme}://{host}:{port}/auth/bearer"
        "         - Bearer auth required"
    )
    print(
        f"  GET  {scheme}://{host}:{port}/auth/api-key"
        "        - API key auth required"
    )
    print(
        f"  GET  {scheme}://{host}:{port}/auth/basic"
        "          - Basic auth required"
    )
    print()

    ssl_context = None
    if https:
        cert_dir = tempfile.mkdtemp(prefix="synnax-http-server-")
        cert_path, key_path = _generate_self_signed_cert(cert_dir)
        ssl_context = (cert_path, key_path)
        print(f"TLS certificate: {cert_path}")
        print(f"TLS private key: {key_path}")
        print()

    app.run(host=host, port=port, debug=False, ssl_context=ssl_context)


def _parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="HTTP/HTTPS mock server for testing")
    parser.add_argument("--host", default="127.0.0.1", help="Bind address")
    parser.add_argument("--port", type=int, default=8081, help="Port number")
    parser.add_argument(
        "--https", action="store_true", help="Enable HTTPS with a self-signed cert"
    )
    parser.add_argument(
        "--auth",
        choices=["none", "basic", "bearer", "api-key"],
        default="none",
        help="Auth middleware to enforce on all routes",
    )
    parser.add_argument(
        "--username",
        default="admin",
        help="Username for basic auth (default: admin)",
    )
    parser.add_argument(
        "--password",
        default="password",
        help="Password for basic auth (default: password)",
    )
    parser.add_argument(
        "--token",
        default="test-token",
        help="Token for bearer auth (default: test-token)",
    )
    parser.add_argument(
        "--api-key",
        default="test-api-key",
        help="API key for api-key auth (default: test-api-key)",
    )
    return parser.parse_args()


if __name__ == "__main__":
    args = _parse_args()
    AUTH_CREDENTIALS["username"] = args.username
    AUTH_CREDENTIALS["password"] = args.password
    AUTH_CREDENTIALS["token"] = args.token
    AUTH_CREDENTIALS["api_key"] = args.api_key
    run_server(host=args.host, port=args.port, https=args.https, auth=args.auth)
