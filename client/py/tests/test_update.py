#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
from collections.abc import Callable
from typing import Any
from uuid import UUID, uuid4

import pytest

from alamos import NOOP
from freighter import URL, Context, Empty, Next, UnaryClient
from freighter.mock import MockUnaryClient
from synnax import (
    access,
    arc,
    channel,
    device,
    framer,
    group,
    label,
    lineplot,
    log,
    panel,
    project,
    rack,
    ranger,
    schematic,
    signals,
    table,
    task,
    user,
    view,
)
from synnax.exceptions import create_registry
from synnax.transport import Transport
from x.codec import JSONCodec


class Recorder:
    """Holds the endpoint and the JSON body of each request that a client sends."""

    unary: MockUnaryClient
    endpoints: list[str]

    def __init__(self) -> None:
        self.unary = MockUnaryClient(responses=[Empty()])
        self.endpoints = []
        self.unary.use(self._record)

    def _record(self, ctx: Context, next_: Next) -> Context:
        self.endpoints.append(ctx.target)
        return next_(ctx)

    def sent(self) -> tuple[str, Any]:
        assert len(self.unary.requests) == 1
        return self.endpoints[0], json.loads(JSONCodec().encode(self.unary.requests[0]))


def _offline() -> Transport:
    """Returns a transport that never connects, for clients that hold one."""
    return Transport(
        url=URL(host="localhost", port=0),
        instrumentation=NOOP,
        registry=create_registry(),
    )


def _frames(unary: UnaryClient) -> framer.Client:
    transport = _offline()
    return framer.Client(
        stream_client=transport.stream,
        async_client=transport.stream_async,
        unary_client=unary,
        file_transport=transport.file_transport,
        retriever=channel.Retriever(client=unary),
        deleter=framer.Deleter(client=unary, instrumentation=NOOP),
        registry=create_registry(),
    )


def _tasks(unary: UnaryClient) -> task.Client:
    return task.Client(
        client=unary,
        frame_client=_frames(unary),
        rack_client=rack.Client(client=unary),
        device_client=device.Client(client=unary),
    )


def _ranges(unary: UnaryClient) -> ranger.Client:
    frames = _frames(unary)
    retriever = channel.Retriever(client=unary)
    return ranger.Client(
        unary_client=unary,
        frame_client=frames,
        writer=ranger.Writer(client=unary),
        retriever=ranger.Retriever(client=unary),
        channel_retriever=retriever,
        signals=signals.Registry(frame_client=frames, channels=retriever),
        tasks=_tasks(unary),
        labels=label.Client(unary),
    )


def _project(unary: UnaryClient) -> project.Client:
    return project.Client(client=unary, file_transport=_offline().file_transport)


ID = uuid4()

# Each case is a client factory, the key of a record, and the dispatch route.
CLIENTS = [
    pytest.param(lambda c: rack.Client(client=c), 12, "/rack/dispatch", id="rack"),
    pytest.param(
        lambda c: device.Client(client=c), "dev", "/device/dispatch", id="device"
    ),
    pytest.param(_tasks, ID, "/task/dispatch", id="task"),
    pytest.param(_ranges, ID, "/range/dispatch", id="range"),
    pytest.param(user.Client, ID, "/user/dispatch", id="user"),
    pytest.param(group.Client, ID, "/group/dispatch", id="group"),
    pytest.param(label.Client, ID, "/label/dispatch", id="label"),
    pytest.param(_project, ID, "/project/dispatch", id="project"),
    pytest.param(lambda c: view.Client(client=c), ID, "/view/dispatch", id="view"),
    pytest.param(panel.Client, ID, "/panel/dispatch", id="panel"),
    pytest.param(lineplot.Client, ID, "/lineplot/dispatch", id="lineplot"),
    pytest.param(log.Client, ID, "/log/dispatch", id="log"),
    pytest.param(table.Client, ID, "/table/dispatch", id="table"),
    pytest.param(schematic.Client, ID, "/schematic/dispatch", id="schematic"),
    pytest.param(
        schematic.symbol.Client, ID, "/schematic/symbol/dispatch", id="symbol"
    ),
    pytest.param(
        lambda c: access.role.Client(client=c), ID, "/access/role/dispatch", id="role"
    ),
    pytest.param(
        lambda c: access.policy.Client(client=c),
        ID,
        "/access/policy/dispatch",
        id="policy",
    ),
    pytest.param(arc.Client, ID, "/arc/dispatch", id="arc"),
]

# Each case is a client factory, the key of a record, and an optional field.
CLEARS = [
    pytest.param(lambda c: rack.Client(client=c), 12, "group", id="rack"),
    pytest.param(lambda c: device.Client(client=c), "dev", "group", id="device"),
    pytest.param(_ranges, ID, "color", id="range"),
    pytest.param(panel.Client, ID, "project", id="panel"),
    pytest.param(schematic.symbol.Client, ID, "group", id="symbol"),
]


def _key(key: Any) -> Any:
    return str(key) if isinstance(key, UUID) else key


def _request(key: Any, value: dict[str, Any]) -> dict[str, Any]:
    return {
        "key": _key(key),
        "dispatch_key": "",
        "actions": [{"type": "update", "update": {"value": value, "guards": []}}],
    }


class TestUpdate:
    """Tests that each client sends an update in the form that the Core takes."""

    @pytest.mark.parametrize(("factory", "key", "endpoint"), CLIENTS)
    def test_one(
        self, factory: Callable[[UnaryClient], Any], key: Any, endpoint: str
    ) -> None:
        """Should send one update to the dispatch route of the record."""
        r = Recorder()
        factory(r.unary).update(key=key, name="Renamed")
        assert r.sent() == (endpoint, _request(key, {"name": "Renamed"}))

    @pytest.mark.parametrize(("factory", "key", "endpoint"), CLIENTS)
    def test_list(
        self, factory: Callable[[UnaryClient], Any], key: Any, endpoint: str
    ) -> None:
        """Should send a list of updates in one batch."""
        r = Recorder()
        factory(r.unary).update([{"key": key, "name": "A"}, {"key": key, "name": "B"}])
        assert r.sent() == (
            "/batch",
            {
                "requests": [
                    {"endpoint": endpoint, "payload": _request(key, {"name": n})}
                    for n in ("A", "B")
                ]
            },
        )

    @pytest.mark.parametrize(("factory", "key", "field"), CLEARS)
    def test_clear(
        self, factory: Callable[[UnaryClient], Any], key: Any, field: str
    ) -> None:
        """Should keep a None that clears an optional field."""
        r = Recorder()
        factory(r.unary).update(key=key, **{field: None})
        assert r.sent()[1] == _request(key, {field: None})

    def test_operators(self) -> None:
        """Should send list operators and a struct merge as the Core reads them."""
        r = Recorder()
        rack.Client(client=r.unary).update(
            key=12, integrations={"insert": ["labjack"], "remove": ["ni"]}
        )
        assert r.sent()[1] == _request(
            12, {"integrations": {"insert": ["labjack"], "remove": ["ni"]}}
        )

    def test_variant(self) -> None:
        """Should send the type of a union record that a list update changes."""
        r = Recorder()
        change: arc.ModuleUpdateInput = {"key": ID, "type": "module", "name": "M"}
        arc.Client(r.unary).update([change])
        assert r.sent()[1] == _request(ID, {"type": "module", "name": "M"})

    def test_set_end(self) -> None:
        """Should merge the end into the time range of a range."""
        r = Recorder()
        _ranges(r.unary).set_end(ID, 1_700_000_000_123_456_789)
        assert r.sent()[1] == _request(
            ID, {"time_range": {"end": 1_700_000_000_123_456_789}}
        )

    def test_set_group(self) -> None:
        """Should move each rack with an update in one batch."""
        r = Recorder()
        group_key = uuid4()
        rack.Client(client=r.unary).set_group([1, 2], group_key)
        endpoint, body = r.sent()
        assert endpoint == "/batch"
        assert [e["payload"] for e in body["requests"]] == [
            _request(k, {"group": str(group_key)}) for k in (1, 2)
        ]


def _type_errors(racks: rack.Client, tasks: task.Client, users: user.Client) -> None:
    """Holds calls that must not type check. mypy fails on an unused ignore."""
    racks.update(key=1, nmae="Rack")  # type: ignore[call-overload]
    racks.update(name="Rack")  # type: ignore[call-overload]
    racks.update(key=1, integrations=["ni"])  # type: ignore[call-overload]
    racks.update([{"name": "Rack"}])  # type: ignore[typeddict-item]
    tasks.update(key=uuid4(), config_hash="abc")  # type: ignore[call-overload]
    users.update(key=uuid4(), root_user=True)  # type: ignore[call-overload]
