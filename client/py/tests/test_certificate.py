#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import uuid4

import pytest

import synnax as sy
from synnax import certificate, secret
from x import filter

# A self-signed P-256 certificate valid until 2126, its key, and an unrelated key.
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
OTHER_PRIVATE_KEY = """-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgCEpmg8mYhXe2mUBp
Zj5oeHcm37uA23Gal/ivbSzbF2ShRANCAAQoi1G7u+dFDR3/oMHvF2swFlLtpA9e
SUHP7pVz5CuFmHrTXcGKjGaN0e54b+eQR3sHOokr9B46aGQon8U6WZ1E
-----END PRIVATE KEY-----
"""
SHA1 = "c67f3f9384d557875ab3dc59bea3d9e11239ea73"
SHA256 = "20637df2df608327ed9cd031811c822513256444225a7bf655559f0baee4a970"


@pytest.mark.certificate
class TestCertificate:
    def test_create_stored(self, client: sy.Synnax) -> None:
        """Should read the details of a stored certificate."""
        c = client.certificates.set(certificate.NewStored(name="CA", pem=PEM))
        assert isinstance(c, certificate.StoredCertificate)
        assert c.private_key is None
        assert c.details.sha1 == SHA1
        assert c.details.sha256 == SHA256
        assert c.details.subject == "CN=Synnax Test Client,O=Synnax Labs"
        assert c.details.issuer == c.details.subject
        assert c.details.not_after > sy.TimeStamp.now()

    def test_create_with_private_key(self, client: sy.Synnax) -> None:
        """Should save a private key as a secret."""
        c = client.certificates.set(
            certificate.NewStored(name="Client", pem=PEM, private_key_pem=PRIVATE_KEY)
        )
        assert isinstance(c, certificate.StoredCertificate)
        assert c.private_key is not None
        key = client.secrets.retrieve(key=c.private_key)
        assert isinstance(key, secret.StoredSecret)

    def test_create_mismatched_key(self, client: sy.Synnax) -> None:
        """Should reject a private key that does not match the certificate."""
        with pytest.raises(
            sy.PathError,
            match=(
                r"^private_key_pem: does not match the certificate: "
                r"validation error$"
            ),
        ):
            client.certificates.set(
                certificate.NewStored(
                    name="Mismatched", pem=PEM, private_key_pem=OTHER_PRIVATE_KEY
                )
            )

    def test_create_file(self, client: sy.Synnax) -> None:
        """Should create a file certificate."""
        c = client.certificates.set(
            certificate.FileCertificate(
                name="On disk",
                certificate_path="/etc/synnax/client.crt",
                private_key_file="client.key",
            )
        )
        assert isinstance(c, certificate.FileCertificate)
        assert c.certificate_path == "/etc/synnax/client.crt"
        assert c.private_key_file == "client.key"

    def test_retrieve(self, client: sy.Synnax) -> None:
        """Should retrieve a certificate by key."""
        c = client.certificates.set(certificate.NewStored(name="Retrieve", pem=PEM))
        assert client.certificates.retrieve(key=c.key) == c

    def test_retrieve_by_where(self, client: sy.Synnax) -> None:
        """Should leave out a key that no certificate has."""
        c = client.certificates.set(certificate.NewStored(name="Kept", pem=PEM))
        where = certificate.Where(key=filter.Set[certificate.Key](in_=[c.key, uuid4()]))
        assert client.certificates.retrieve(where=where) == [c]

    def test_retrieve_key_list(self, client: sy.Synnax) -> None:
        """Should return a list for a list of keys, leaving out a key that no
        certificate has."""
        c = client.certificates.set(certificate.NewStored(name="Listed", pem=PEM))
        assert client.certificates.retrieve(key=[c.key, uuid4()]) == [c]

    def test_retrieve_by_name(self, client: sy.Synnax) -> None:
        """Should return one certificate for a plain name."""
        name = f"named-certificate-{uuid4()}"
        c = client.certificates.set(certificate.NewStored(name=name, pem=PEM))
        assert client.certificates.retrieve(name=name) == c

    def test_order_and_after(self, client: sy.Synnax) -> None:
        """Should page through the matching certificates in name order from a
        cursor."""
        prefix = uuid4()
        created = client.certificates.set(
            [certificate.NewStored(name=f"{prefix} {n}", pem=PEM) for n in "CAB"]
        )
        keys = [c.key for c in created]
        order = certificate.Order(field="name")
        first = client.certificates.retrieve(key=keys, order=order, limit=2)
        assert [c.name for c in first] == [f"{prefix} A", f"{prefix} B"]
        after = certificate.Cursor(key=first[-1].key, name=first[-1].name)
        second = client.certificates.retrieve(key=keys, order=order, after=after)
        assert [c.name for c in second] == [f"{prefix} C"]

    def test_rename(self, client: sy.Synnax) -> None:
        """Should rename a certificate."""
        c = client.certificates.set(certificate.NewStored(name="Before", pem=PEM))
        client.certificates.rename(c.key, "After")
        assert client.certificates.retrieve(key=c.key).name == "After"

    def test_delete(self, client: sy.Synnax) -> None:
        """Should delete a certificate."""
        c = client.certificates.set(certificate.NewStored(name="Delete", pem=PEM))
        client.certificates.delete(c.key)
        with pytest.raises(
            sy.NotFoundError, match=f'Found no certificate where key = "{c.key}"'
        ):
            client.certificates.retrieve(c.key)

    def test_delete_used_private_key(self, client: sy.Synnax) -> None:
        """Should refuse to delete a private key that a certificate uses."""
        c = client.certificates.set(
            certificate.NewStored(name="In use", pem=PEM, private_key_pem=PRIVATE_KEY)
        )
        assert isinstance(c, certificate.StoredCertificate)
        assert c.private_key is not None
        with pytest.raises(
            sy.ValidationError,
            match=r"^cannot delete secret In use private key because certificate "
            r"In use uses it as its private_key: validation error$",
        ):
            client.secrets.delete(c.private_key)
        assert client.secrets.retrieve(key=c.private_key).key == c.private_key
