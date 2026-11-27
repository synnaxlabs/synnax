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
from synnax import opcua


def new_server(rack: sy.Rack, name: str = "PLC") -> opcua.server.Server:
    return opcua.server.Server(
        rack=rack.key,
        name=name,
        connection=opcua.server.Connection(endpoint="opc.tcp://localhost:4840"),
    )


@pytest.mark.opcua
class TestServerClient:
    def test_create_and_retrieve(self, client: sy.Synnax, rack: sy.Rack) -> None:
        created = client.opcua.servers.create(new_server(rack))
        retrieved = client.opcua.servers.retrieve(created.key)
        assert retrieved == created
        assert isinstance(retrieved.connection.auth, opcua.server.AnonymousAuth)

    def test_create_many(self, client: sy.Synnax, rack: sy.Rack) -> None:
        servers = client.opcua.servers.create(
            [new_server(rack, "A"), new_server(rack, "B")]
        )
        assert [s.name for s in servers] == ["A", "B"]

    def test_retrieve_by_rack(self, client: sy.Synnax, rack: sy.Rack) -> None:
        created = client.opcua.servers.create(new_server(rack))
        servers = client.opcua.servers.retrieve(rack=rack.key)
        assert [s.key for s in servers] == [created.key]

    def test_retrieve_status(self, client: sy.Synnax, rack: sy.Rack) -> None:
        created = client.opcua.servers.create(new_server(rack))
        retrieved = client.opcua.servers.retrieve(created.key, include_status=True)
        assert retrieved.status is not None
        assert retrieved.status.details is not None
        assert retrieved.status.details.server == created.key

    def test_retrieve_missing(self, client: sy.Synnax) -> None:
        with pytest.raises(sy.NotFoundError):
            client.opcua.servers.retrieve(uuid4())

    def test_create_invalid_endpoint(self, client: sy.Synnax, rack: sy.Rack) -> None:
        server = new_server(rack)
        server.connection.endpoint = "http://localhost:4840"
        with pytest.raises(sy.ValidationError):
            client.opcua.servers.create(server)

    def test_update_identity(self, client: sy.Synnax, rack: sy.Rack) -> None:
        created = client.opcua.servers.create(new_server(rack))
        stored = client.opcua.servers.update(
            created.key,
            opcua.server.IdentityUpdate(
                rack=rack.key, name="Renamed", connection=created.connection
            ),
        )
        assert stored.name == "Renamed"
        assert client.opcua.servers.retrieve(created.key) == stored

    def test_update_configuration(self, client: sy.Synnax, rack: sy.Rack) -> None:
        created = client.opcua.servers.create(new_server(rack))
        index = client.channels.create(
            name=f"opcua_time_{uuid4()}", data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data = client.channels.create(
            name=f"opcua_temp_{uuid4()}",
            data_type=sy.DataType.FLOAT64,
            index=index.key,
        )
        read = opcua.server.Read(node="i=2258", index=index.key, channel=data.key)
        stored = client.opcua.servers.update(
            created.key, opcua.server.ConfigurationUpdate(reads=[read])
        )
        assert stored.reads == [read]
        assert stored.name == created.name

    def test_delete(self, client: sy.Synnax, rack: sy.Rack) -> None:
        created = client.opcua.servers.create(new_server(rack))
        client.opcua.servers.delete(created.key)
        with pytest.raises(sy.NotFoundError):
            client.opcua.servers.retrieve(created.key)
