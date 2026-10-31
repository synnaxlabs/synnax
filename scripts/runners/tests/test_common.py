#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import argparse

import pytest
from botocore.exceptions import ClientError

from common import add_connection_args, instance_for, instance_prefix, secret


def parse(*argv: str) -> argparse.Namespace:
    p = argparse.ArgumentParser()
    add_connection_args(p)
    return p.parse_args(argv)


class TestConnectionArgs:
    """Tests for the connection flags every script shares."""

    def test_should_create_channels_by_default(self) -> None:
        assert parse().create_channels

    def test_should_skip_channel_creation_when_asked(self) -> None:
        assert not parse("--no-create-channels").create_channels


class TestInstanceFor:
    """Tests for matching a runner to its EC2 instance."""

    def test_should_return_the_instance_the_suffix_names(self) -> None:
        found: dict[str, dict] = {"i-0f44aa": {}, "i-0b12cc": {}}
        assert instance_for("ubuntu-build-bot-i-0f44", found=found) == "i-0f44aa"

    def test_should_return_none_without_a_match(self) -> None:
        assert instance_for("ubuntu-build-bot-i-0f44", found={"i-0b12cc": {}}) is None

    def test_should_raise_when_the_suffix_is_ambiguous(self) -> None:
        found: dict[str, dict] = {"i-0f44aa": {}, "i-0f44bb": {}}
        with pytest.raises(LookupError):
            instance_for("ubuntu-build-bot-i-0f44", found=found)

    def test_should_ignore_runners_without_an_instance(self) -> None:
        assert instance_prefix("github-hosted") is None


class SSM:
    """Stands in for the SSM client: serves parameters and records each request."""

    def __init__(self, parameters: dict[str, str]) -> None:
        self.parameters = parameters
        self.requests: list[tuple[str, bool]] = []

    def get_parameter(self, *, Name: str, WithDecryption: bool = False) -> dict:
        self.requests.append((Name, WithDecryption))
        if Name not in self.parameters:
            raise ClientError({"Error": {"Code": "ParameterNotFound"}}, "GetParameter")
        return {"Parameter": {"Value": self.parameters[Name]}}


@pytest.fixture
def ssm(monkeypatch: pytest.MonkeyPatch) -> SSM:
    monkeypatch.delenv("GH_TOKEN", raising=False)
    monkeypatch.delenv("SYNNAX_PASSWORD", raising=False)
    return SSM({"/runners/gh-token": "stored"})


class TestSecret:
    """Tests for reading a secret from the environment or Parameter Store."""

    def test_should_prefer_the_environment(
        self, ssm: SSM, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        monkeypatch.setenv("GH_TOKEN", "from-env")
        assert secret("GH_TOKEN", store=ssm) == "from-env"
        assert ssm.requests == []

    def test_should_read_the_decrypted_parameter(self, ssm: SSM) -> None:
        assert secret("GH_TOKEN", store=ssm) == "stored"
        assert ssm.requests == [("/runners/gh-token", True)]

    def test_should_raise_when_the_parameter_is_missing(self, ssm: SSM) -> None:
        with pytest.raises(ClientError):
            secret("SYNNAX_PASSWORD", store=ssm)
