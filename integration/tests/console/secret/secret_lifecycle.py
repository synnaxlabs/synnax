#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


import synnax as sy
from console.case import ConsoleCase
from synnax import certificate, secret
from x import random_name

# A self-signed client certificate and its key. The Core stores the key as a secret
# that the certificate uses.
PEM = """-----BEGIN CERTIFICATE-----
MIIBnDCCAUKgAwIBAgIUTlojETDboAh4nmfahlzAemoZ4g4wCgYIKoZIzj0EAwIw
MzEbMBkGA1UEAwwSU3lubmF4IFRlc3QgQ2xpZW50MRQwEgYDVQQKDAtTeW5uYXgg
TGFiczAgFw0yNjExMDQwMjUyMTBaGA8yMTI2MTAxMTAyNTIxMFowMzEbMBkGA1UE
AwwSU3lubmF4IFRlc3QgQ2xpZW50MRQwEgYDVQQKDAtTeW5uYXggTGFiczBZMBMG
ByqGSM49AgEGCCqGSM49AwEHA0IABAttV7/ZZaP/ck5ZSEibyz8Bq/FOha2BqRFx
KcHLLjh+GByF19Xf9tZdf+9TfDlOPwyGPp+zArW6WOjeHdzzPv6jMjAwMB0GA1Ud
DgQWBBQnJ8GnAtkFwtBOUq0b4+pWyuXWQTAPBgNVHRMBAf8EBTADAQH/MAoGCCqG
SM49BAMCA0gAMEUCIQDucqkUixw8A44mdBmPFZIhYtq3tzNMzPF5zPD9ONNZPwIg
I3WFxQbCO86E+bgOOHH1NSjmhCFtORY5jSOGw6yL3m4=
-----END CERTIFICATE-----
"""

PRIVATE_KEY = """-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgjJtOV99NCzRnmMoB
wui9Lv+9a8aqZ+LwPX2PyGfVEa+hRANCAAQLbVe/2WWj/3JOWUhIm8s/AavxToWt
gakRcSnByy44fhgchdfV3/bWXX/vU3w5Tj8Mhj6fswK1uljo3h3c8z7+
-----END PRIVATE KEY-----
"""


class SecretLifecycle(ConsoleCase):
    """Test the Secrets and Certificates pages of Settings."""

    secret_name: str
    certificate_name: str
    private_key: secret.Key | None = None

    def setup(self) -> None:
        super().setup()
        self.secret_name = random_name()
        self.certificate_name = random_name()
        c = self.client.certificates.set(
            certificate.NewStored(
                name=self.certificate_name, pem=PEM, private_key_pem=PRIVATE_KEY
            )
        )
        assert isinstance(c, certificate.StoredCertificate)
        assert c.private_key is not None
        self.certificate_key = c.key
        self.private_key = c.private_key

    def teardown(self) -> None:
        self.page.keyboard.press("Escape")
        if self.console.certificates.exists(self.certificate_name):
            self.console.certificates.delete(self.certificate_name)
        if self.private_key is not None:
            self.client.secrets.delete(self.private_key)
        if self.console.secrets.exists(self.secret_name):
            self.console.secrets.delete(self.secret_name)
        super().teardown()

    def run(self) -> None:
        self.test_create_secret()
        self.test_rename_secret()
        self.test_replace_secret_value()
        self.test_open_certificate_from_used_by()
        self.test_rename_certificate()
        self.test_delete_certificate()
        self.test_delete_secret()

    def test_create_secret(self) -> None:
        self.log("Testing: Create secret")
        self.console.secrets.create(self.secret_name, value="hunter2")
        created = self.client.secrets.retrieve(name=[self.secret_name])
        assert len(created) == 1, f"The Core should hold secret {self.secret_name}"

    def test_rename_secret(self) -> None:
        self.log("Testing: Rename secret")
        new_name = random_name()
        self.console.secrets.rename(old_name=self.secret_name, new_name=new_name)
        self.secret_name = new_name

    def test_replace_secret_value(self) -> None:
        self.log("Testing: Replace secret value")
        [before] = self.client.secrets.retrieve(name=[self.secret_name])
        self.console.secrets.replace_value(self.secret_name, value="hunter3")
        deadline = sy.TimeStamp.now() + sy.TimeSpan.SECOND * 5
        while self.client.secrets.retrieve(key=before.key).updated <= before.updated:
            assert sy.TimeStamp.now() < deadline, "The Core kept the old value"
            self.page.wait_for_timeout(100)

    def test_open_certificate_from_used_by(self) -> None:
        self.log("Testing: Open a certificate from the uses of its key")
        assert self.private_key is not None
        key_name = self.client.secrets.retrieve(key=self.private_key).name
        self.console.secrets.open_used_by(
            secret=key_name, resource=self.certificate_name
        )
        assert self.console.certificates.is_selected(self.certificate_name), (
            f"Certificate {self.certificate_name} should be selected"
        )

    def test_rename_certificate(self) -> None:
        self.log("Testing: Rename certificate")
        new_name = random_name()
        self.console.certificates.rename(
            old_name=self.certificate_name, new_name=new_name
        )
        self.certificate_name = new_name

    def test_delete_certificate(self) -> None:
        self.log("Testing: Delete certificate")
        self.console.certificates.delete(self.certificate_name)
        assert not self.console.certificates.exists(self.certificate_name)

    def test_delete_secret(self) -> None:
        self.log("Testing: Delete secret")
        self.console.secrets.delete(self.secret_name)
        assert not self.console.secrets.exists(self.secret_name)
