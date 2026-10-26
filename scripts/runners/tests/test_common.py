#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest
from botocore.exceptions import ClientError

import common
from common import instance_for, instance_prefix, secret


class TestInstanceFor:
    """Tests for matching a runner to its EC2 instance."""

    def test_should_return_the_instance_the_suffix_names(self) -> None:
        found: dict[str, dict] = {"i-0f44aa": {}, "i-0b12cc": {}}
        assert instance_for("ubuntu-build-bot-i-0f44", found) == "i-0f44aa"

    def test_should_return_none_without_a_match(self) -> None:
        assert instance_for("ubuntu-build-bot-i-0f44", {"i-0b12cc": {}}) is None

    def test_should_raise_when_the_suffix_is_ambiguous(self) -> None:
        found: dict[str, dict] = {"i-0f44aa": {}, "i-0f44bb": {}}
        with pytest.raises(LookupError):
            instance_for("ubuntu-build-bot-i-0f44", found)

    def test_should_ignore_runners_without_an_instance(self) -> None:
        assert instance_prefix("github-hosted") is None


class SSM:
    """Stands in for the SSM client: serves parameters and records each request."""

    def __init__(self, parameters: dict[str, str]) -> None:
        self.parameters = parameters
        self.requests: list[dict] = []

    def get_parameter(self, **request: str | bool) -> dict:
        self.requests.append(request)
        name = request["Name"]
        if not isinstance(name, str) or name not in self.parameters:
            raise ClientError({"Error": {"Code": "ParameterNotFound"}}, "GetParameter")
        return {"Parameter": {"Value": self.parameters[name]}}


@pytest.fixture
def ssm(monkeypatch: pytest.MonkeyPatch) -> SSM:
    client = SSM({"/runners/gh-token": "stored"})
    monkeypatch.setattr(common.boto3, "client", lambda *_, **__: client)
    monkeypatch.delenv("GH_TOKEN", raising=False)
    monkeypatch.delenv("SYNNAX_PASSWORD", raising=False)
    return client


class TestSecret:
    """Tests for reading a secret from the environment or Parameter Store."""

    def test_should_prefer_the_environment(
        self, ssm: SSM, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        monkeypatch.setenv("GH_TOKEN", "from-env")
        assert secret("GH_TOKEN") == "from-env"
        assert ssm.requests == []

    def test_should_read_the_decrypted_parameter(self, ssm: SSM) -> None:
        assert secret("GH_TOKEN") == "stored"
        assert ssm.requests == [{"Name": "/runners/gh-token", "WithDecryption": True}]

    def test_should_raise_when_the_parameter_is_missing(self, ssm: SSM) -> None:
        with pytest.raises(ClientError):
            secret("SYNNAX_PASSWORD")
