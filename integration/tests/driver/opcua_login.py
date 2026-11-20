#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""OPC UA login integration tests.

Each test sends a connection test to the scan task of the Driver, as the Console does
before it saves a device. The Driver reads the password and the client certificate of
the connection only through a grant.
"""

from collections.abc import Iterator, Mapping
from contextlib import contextmanager
from pathlib import Path
from uuid import UUID, uuid4

from examples.opcua import OPCUATLSAuthSim
from examples.opcua.sim import CERT_DIR, TLS_AUTH_PASSWORD

import synnax as sy
from synnax import certificate, rack
from tests.driver.simulator_case import SimulatorCase

TEST_CONNECTION = "test_connection"
LIST = "list"
TIMEOUT = 15 * sy.TimeSpan.SECOND


class OPCUAGrantLogin(SimulatorCase):
    """Verify that a connection test logs in with a password that a grant gives.

    Tests (run sequentially):
        1. Granted password - the server accepts the login.
        2. Granted wrong password - the server rejects the login.
        3. Password without a grant - the Driver refuses the command.
        4. Granted file password - the Driver reads it from its secrets folder.
        5. File password the grant does not name - the Driver refuses the command.
        6. Granted file client certificate - the Driver reads its key file from its
           secrets folder.
        7. File client certificate the grant does not name - the Driver refuses the
           command.
    """

    sim_classes = [OPCUATLSAuthSim]

    def setup(self) -> None:
        super().setup()
        self.rack = self.client.racks.retrieve(name=self.rack_name)
        self.scan = self.client.tasks.retrieve(
            type="opc_scan", rack=self.rack.key, one=True
        )
        device = self.client.devices.retrieve(name=self.device_name)
        connection = device.properties["connection"]
        assert isinstance(connection, dict)
        self.connection = connection
        own = self.client.certificates.retrieve(
            key=UUID(str(connection["client_certificate"]))
        )
        assert isinstance(own, certificate.StoredCertificate)
        assert own.private_key is not None
        # The rack reads the private key of the client certificate only through a grant.
        self.client_key = own.private_key
        browser = self.client.tasks.retrieve(
            type="file_browser", rack=self.rack.key, one=True
        )
        listing = browser.execute_command_sync(LIST, args={}, timeout=TIMEOUT)
        assert listing.details is not None
        data = listing.details.data
        assert isinstance(data, dict), data
        assert data["secrets_dir"], f"{self.rack.name} has no secrets folder"
        self.secrets_dir = Path(str(data["secrets_dir"]))

    def run(self) -> None:
        self.test_granted_password()
        self.test_granted_wrong_password()
        self.test_password_without_grant()
        self.test_granted_file_password()
        self.test_file_password_grant_does_not_name()
        self.test_granted_file_client_certificate()
        self.test_file_client_certificate_grant_does_not_name()

    def test_granted_password(self) -> None:
        """No device uses the secret, so only the grant lets the rack read it."""
        self.log("Testing: Granted password")
        status = self._test_with_password(TLS_AUTH_PASSWORD, granted=True)
        if status.variant != "success":
            self.fail(f"login failed: {status.message}: {status.description}")

    def test_granted_wrong_password(self) -> None:
        self.log("Testing: Granted wrong password")
        status = self._test_with_password("wrongpass", granted=True)
        if status.variant != "error" or status.message != "Failed to connect to server":
            self.fail(f"login should fail, got {status.variant}: {status.message}")

    def test_password_without_grant(self) -> None:
        self.log("Testing: Password without a grant")
        status = self._test_with_password(TLS_AUTH_PASSWORD, granted=False)
        expected = (
            "The command sends a connection with a password, so it must carry a grant."
        )
        if status.variant != "error" or status.description != expected:
            self.fail(f"command should be refused, got {status.description!r}")

    def test_granted_file_password(self) -> None:
        self.log("Testing: Granted file password")
        with self._file_password(TLS_AUTH_PASSWORD) as key:
            connection = {**self.connection, "password": str(key)}
            status = self._send(connection, [key, self.client_key])
        if status.variant != "success":
            self.fail(f"login failed: {status.message}: {status.description}")

    def test_file_password_grant_does_not_name(self) -> None:
        """The file holds the right password, so a Driver that read it would log in."""
        self.log("Testing: File password the grant does not name")
        other = self.client.secrets.set(
            name=f"OPC UA login test {sy.TimeStamp.now()}", value="unused"
        )
        try:
            with self._file_password(TLS_AUTH_PASSWORD) as key:
                connection = {**self.connection, "password": str(key)}
                status = self._send(connection, [other.key, self.client_key])
                expected = f"The grant does not name secret {key}."
        finally:
            self.client.secrets.delete(other.key)
        if status.variant != "error" or status.description != expected:
            self.fail(f"command should be refused, got {status.description!r}")

    def test_granted_file_client_certificate(self) -> None:
        self.log("Testing: Granted file client certificate")
        with self._file_client_certificate() as key:
            status = self._send_with_client_certificate(key, granted=True)
        if status.variant != "success":
            self.fail(f"login failed: {status.message}: {status.description}")

    def test_file_client_certificate_grant_does_not_name(self) -> None:
        """The files hold the right certificate, so a Driver that read them would log
        in."""
        self.log("Testing: File client certificate the grant does not name")
        with self._file_client_certificate() as key:
            status = self._send_with_client_certificate(key, granted=False)
            expected = f"The grant does not name certificate {key}."
        if status.variant != "error" or status.description != expected:
            self.fail(f"command should be refused, got {status.description!r}")

    @contextmanager
    def _file_client_certificate(self) -> Iterator[UUID]:
        """Copies the private key of the client certificate to a file in the secrets
        folder of the Driver, and yields the key of a file certificate of the rack that
        names it."""
        name = f"opc_client_key_{uuid4().hex}.pem"
        path = self.secrets_dir / name
        path.write_text((CERT_DIR / "client-private-key.pem").read_text())
        try:
            c = self.client.certificates.set(
                certificate.FileCertificate(
                    name=f"OPC UA file login test {name}",
                    certificate_path=str(CERT_DIR / "client-certificate.der"),
                    private_key_file=name,
                    rack=self.rack.key,
                )
            )
            try:
                yield c.key
            finally:
                self.client.certificates.delete(c.key)
        finally:
            path.unlink()

    def _send_with_client_certificate(
        self, key: UUID, *, granted: bool
    ) -> sy.task.Status:
        """Logs in with the password in a new secret and the file client certificate
        with key, which the grant names when granted is true."""
        s = self.client.secrets.set(
            name=f"OPC UA login test {sy.TimeStamp.now()}", value=TLS_AUTH_PASSWORD
        )
        try:
            connection = {
                **self.connection,
                "password": str(s.key),
                "client_certificate": str(key),
            }
            return self._send(
                connection, [s.key], certificates=[key] if granted else []
            )
        finally:
            self.client.secrets.delete(s.key)

    @contextmanager
    def _file_password(self, password: str) -> Iterator[UUID]:
        """Writes password to a file in the secrets folder of the Driver, and yields
        the key of a file secret of the rack that names it."""
        name = f"opc_password_{uuid4().hex}.txt"
        path = self.secrets_dir / name
        path.write_text(password)
        try:
            s = self.client.secrets.set(
                name=f"OPC UA file login test {name}",
                file_name=name,
                rack=self.rack.key,
            )
            try:
                yield s.key
            finally:
                self.client.secrets.delete(s.key)
        finally:
            path.unlink()

    def _test_with_password(self, password: str, *, granted: bool) -> sy.task.Status:
        s = self.client.secrets.set(
            name=f"OPC UA login test {sy.TimeStamp.now()}", value=password
        )
        try:
            connection = {**self.connection, "password": str(s.key)}
            return self._send(connection, [s.key, self.client_key] if granted else [])
        finally:
            self.client.secrets.delete(s.key)

    def _send(
        self,
        connection: Mapping[str, object],
        secrets: list[UUID],
        *,
        certificates: list[UUID] | None = None,
    ) -> sy.task.Status:
        """Sends a connection test of connection. With secrets, the command carries a
        grant of them and the certificates."""
        args: dict[str, object] = {"connection": connection}
        if secrets:
            grant = self.client.secrets.grant(
                rack.resource_id(self.rack.key),
                secrets,
                task=self.scan.key,
                type_=TEST_CONNECTION,
                args=args,
                certificates=certificates,
            )
            args = {"grant": str(grant)}
        return self.scan.execute_command_sync(
            TEST_CONNECTION, args=args, timeout=TIMEOUT
        )
