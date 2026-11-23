#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import UUID

import pytest

import synnax as sy
from synnax.channel.types_gen import Where
from x.strings import random_name


@pytest.mark.channel
class TestCalculatedBreaker:
    @staticmethod
    def keys(channels: list[sy.Channel]) -> set[sy.channel.Key]:
        return {ch.key for ch in channels}

    @pytest.fixture
    def channels(self, client: sy.Synnax) -> tuple[sy.Channel, sy.Channel, UUID]:
        prefix = random_name()
        plain = client.channels.create(
            name=f"{prefix}_plain", data_type=sy.DataType.FLOAT32, virtual=True
        )
        calc = client.calculations.create(name=f"{prefix}_out", expression="return 2")
        output = client.channels.retrieve(calc.output)
        return plain, output, calc.key

    def test_without_keys(
        self, client: sy.Synnax, channels: tuple[sy.Channel, sy.Channel, UUID]
    ) -> None:
        """Should keep the output and drop the plain channel with no keys given"""
        plain, output, _ = channels
        found = self.keys(client.channels.retrieve(calculated=True))
        assert output.key in found
        assert plain.key not in found
        rest = self.keys(client.channels.retrieve(calculated=False))
        assert plain.key in rest
        assert output.key not in rest

    def test_single(
        self, client: sy.Synnax, channels: tuple[sy.Channel, sy.Channel, UUID]
    ) -> None:
        """Should return one channel for a key or a name with the line"""
        plain, output, _ = channels
        assert client.channels.retrieve(output.key, calculated=True).key == output.key
        assert client.channels.retrieve(output.name, calculated=True).key == output.key
        assert client.channels.retrieve(name=plain.name, calculated=False).key == (
            plain.key
        )
        with pytest.raises(sy.NotFoundError):
            client.channels.retrieve(plain.key, calculated=True)
        with pytest.raises(sy.NotFoundError):
            client.channels.retrieve(key=output.key, calculated=False)

    def test_nested(
        self, client: sy.Synnax, channels: tuple[sy.Channel, sy.Channel, UUID]
    ) -> None:
        """Should match the line inside and, or, and not"""
        plain, output, _ = channels
        keys = [plain.key, output.key]
        assert self.keys(client.channels.retrieve(keys, not_={"calculated": True})) == {
            plain.key
        }
        assert self.keys(
            client.channels.retrieve(
                keys, and_=[{"or": [{"calculated": True}, {"key": plain.key}]}]
            )
        ) == {plain.key, output.key}
        assert self.keys(
            client.channels.retrieve(keys, or_=[{"not": {"calculated": False}}])
        ) == {output.key}
        assert self.keys(
            client.channels.retrieve(
                keys, or_=[{"calculated": True}, {"calculated": False}]
            )
        ) == {plain.key, output.key}
        assert self.keys(
            client.channels.retrieve(keys, virtual=True, calculated=False)
        ) == {plain.key}
        assert self.keys(
            client.channels.retrieve(keys, virtual=False, calculated=True)
        ) == {output.key}

    def test_where_and_field(
        self, client: sy.Synnax, channels: tuple[sy.Channel, sy.Channel, UUID]
    ) -> None:
        """Should match both a where tree and the field beside it"""
        plain, output, _ = channels
        keys = [plain.key, output.key]
        assert self.keys(
            client.channels.retrieve(keys, where=Where(calculated=True))
        ) == {output.key}
        assert (
            self.keys(
                client.channels.retrieve(
                    keys, where=Where(calculated=True), calculated=False
                )
            )
            == set()
        )

    def test_after_delete(
        self, client: sy.Synnax, channels: tuple[sy.Channel, sy.Channel, UUID]
    ) -> None:
        """Should drop the output once its calculation is gone"""
        plain, output, calc = channels
        client.calculations.delete(calc)
        keys = [plain.key, output.key]
        assert self.keys(client.channels.retrieve(keys, calculated=True)) == set()
        assert self.keys(client.channels.retrieve(keys, calculated=False)) == {
            plain.key,
            output.key,
        }

    def test_none_is_refused(self, client: sy.Synnax) -> None:
        """Should refuse calculated=None"""
        with pytest.raises(sy.ValidationError):
            client.channels.retrieve(calculated=None)  # type: ignore[call-overload]
