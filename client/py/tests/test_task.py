#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import threading
from collections.abc import Callable
from typing import Any, assert_type
from uuid import UUID, uuid4

import pytest
from pydantic import BaseModel, Field

import synnax as sy
from synnax import task
from x import filter
from x.strings import random_name
from x.testutil import assert_eventually


def _device(rack: sy.rack.Key) -> sy.Device:
    key = str(uuid4())
    return sy.Device(
        key=key, name=key, rack=rack, location="dev", make="modbus", model="test"
    )


def _ai_channel(
    client: sy.Synnax, port: int, *, device: str = ""
) -> sy.ni.AIVoltageChannel:
    idx = client.channels.create(
        name=random_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
    )
    ch = client.channels.create(
        name=random_name(), data_type=sy.DataType.FLOAT32, index=idx.key
    )
    return sy.ni.AIVoltageChannel(port=port, channel=ch.key, device=device)


def _reply_to_next_command(
    client: sy.Synnax,
    ready: threading.Event,
    variant: sy.status.Variant,
    message: str,
    description: str,
) -> None:
    """Answers the next task command with one status, as a driver does."""
    with client.open_streamer("sy_task_cmd") as s:
        ready.set()
        f = s.read(timeout=5)
        assert f is not None
        cmd = sy.task.Command.model_validate(f["sy_task_cmd"][0])
        client.statuses.set(
            sy.task.Status(
                variant=variant,
                message=message,
                description=description,
                details=sy.task.StatusDetails(running=False, cmd=cmd.key),
                owner=sy.task.resource_id(cmd.task),
            )
        )


@pytest.mark.task
class TestTaskClient:
    def test_create_single(self, client: sy.Synnax) -> None:
        task = client.tasks.create(name="test", type="pagerduty_alert")
        assert isinstance(task.key, UUID)
        assert task.rack != 0

    def test_create_multiple(self, client: sy.Synnax) -> None:
        t1 = sy.Task(name="test1", type="pagerduty_alert")
        t2 = sy.Task(name="test2", type="pagerduty_alert")
        tasks = client.tasks.create(tasks=[t1, t2])
        assert len(tasks) == 2
        assert tasks[0].name == "test1"
        assert tasks[1].name == "test2"

    def test_rename(
        self, client: sy.Synnax, create_routing_key: Callable[[], UUID]
    ) -> None:
        """Should rename a task and keep its config."""
        routing_key = str(create_routing_key())
        task = client.tasks.create(
            name="Before Rename",
            type="pagerduty_alert",
            config={"routing_key": routing_key},
        )
        client.tasks.rename(task.key, "After Rename")
        renamed = client.tasks.retrieve(key=task.key)
        assert renamed.name == "After Rename"
        assert renamed.config == task.config
        assert renamed.config["routing_key"] == routing_key

    def test_retrieve_by_name(self, client: sy.Synnax) -> None:
        name = str(uuid4())
        task = client.tasks.create(name=name, type="pagerduty_alert")
        res = client.tasks.retrieve(name=name)
        assert res.name == name
        assert res.key == task.key

    def test_delete_multiple(self, client: sy.Synnax) -> None:
        """Should delete every task in a tuple of keys."""
        t1 = client.tasks.create(name=str(uuid4()), type="pagerduty_alert")
        t2 = client.tasks.create(name=str(uuid4()), type="pagerduty_alert")
        client.tasks.delete((t1.key, t2.key))
        for key in (t1.key, t2.key):
            with pytest.raises(sy.NotFoundError):
                client.tasks.retrieve(key=key)

    def test_retrieve_by_type(self, client: sy.Synnax) -> None:
        # Task types are a fixed, registered set, so the filter cannot be made unique
        # the way the name and model tests do. Assert membership instead of identity.
        task = client.tasks.create(name="test", type="labjack_scan")
        res = client.tasks.retrieve(types=["labjack_scan"])
        assert all(t.type == "labjack_scan" for t in res)
        assert task.key in {t.key for t in res}

    def test_retrieve_without_status(self, client: sy.Synnax) -> None:
        """Should leave the status unset when it is not asked for."""
        task = client.tasks.create(name=str(uuid4()), type="pagerduty_alert")
        assert client.tasks.retrieve(key=task.key).status is None

    def test_retrieve_with_status(self, client: sy.Synnax) -> None:
        """Should attach a parsed status when asked for one."""
        task = client.tasks.create(name=str(uuid4()), type="pagerduty_alert")
        res = client.tasks.retrieve(key=task.key, include_status=True)
        assert isinstance(res.status, sy.task.Status)
        assert res.status.details is not None
        assert res.status.owner == task.resource_id

    def test_retrieve_by_rack(self, client: sy.Synnax) -> None:
        """Should return a list of only the tasks on the given rack."""
        rack = client.racks.create(name=str(uuid4()))
        task = client.tasks.create(name="test", type="pagerduty_alert", rack=rack.key)
        res = client.tasks.retrieve(rack=rack.key)
        assert [t.key for t in res] == [task.key]

    def test_retrieve_by_types_and_rack(self, client: sy.Synnax) -> None:
        """Should return a list of the tasks matching both the types and the rack."""
        rack = client.racks.create(name=str(uuid4()))
        task = client.tasks.create(name="test", type="pagerduty_alert", rack=rack.key)
        res = client.tasks.retrieve(types=["pagerduty_alert"], rack=rack.key)
        assert_type(res, list[sy.Task])
        assert [t.key for t in res] == [task.key]

    def test_retrieve_by_names_with_limit(self, client: sy.Synnax) -> None:
        """Should return a list when names are combined with a limit."""
        name = str(uuid4())
        task = client.tasks.create(name=name, type="pagerduty_alert")
        res = client.tasks.retrieve(names=[name], limit=1)
        assert_type(res, list[sy.Task])
        assert [t.key for t in res] == [task.key]

    def test_retrieve_single_with_status(self, client: sy.Synnax) -> None:
        """Should return one task when a key or name is given."""
        name = str(uuid4())
        task = client.tasks.create(name=name, type="pagerduty_alert")
        by_key = client.tasks.retrieve(key=task.key, include_status=True)
        assert_type(by_key, sy.Task)
        assert by_key.key == task.key
        by_name = client.tasks.retrieve(name=name, include_status=True)
        assert_type(by_name, sy.Task)
        assert by_name.key == task.key

    def test_retrieve_single_multiple_found(self, client: sy.Synnax) -> None:
        """Should raise when more than one task has the name or type."""
        name = str(uuid4())
        for _ in range(2):
            client.tasks.create(name=name, type="pagerduty_alert")
        with pytest.raises(
            sy.MultipleFoundError,
            match=rf"Expected one task where .*{name}.*, but found 2",
        ):
            client.tasks.retrieve(name=name)
        with pytest.raises(sy.MultipleFoundError):
            client.tasks.retrieve(type="pagerduty_alert", one=True)

    def test_retrieve_no_filters(self, client: sy.Synnax) -> None:
        """Should return a list of every task when no filter is given."""
        task = client.tasks.create(name="test", type="pagerduty_alert")
        res = client.tasks.retrieve()
        assert_type(res, list[sy.Task])
        assert task.key in {t.key for t in res}

    def test_retrieve_type(self, client: sy.Synnax) -> None:
        """Should return a list for a plain type, and one task with one set."""
        rack = client.racks.create(name=str(uuid4()))
        task = client.tasks.create(name="test", type="pagerduty_alert", rack=rack.key)
        res = client.tasks.retrieve(type="pagerduty_alert", rack=rack.key)
        assert_type(res, list[sy.Task])
        assert [t.key for t in res] == [task.key]
        found = client.tasks.retrieve(type="pagerduty_alert", rack=rack.key, one=True)
        assert_type(found, sy.Task)
        assert found.key == task.key

    def test_retrieve_plain_name_beside_fields(self, client: sy.Synnax) -> None:
        """Should return one task for a plain name beside other fields."""
        rack = client.racks.create(name=str(uuid4()))
        name = str(uuid4())
        task = client.tasks.create(name=name, type="pagerduty_alert", rack=rack.key)
        assert client.tasks.retrieve(name=name, rack=rack.key).key == task.key
        with pytest.raises(sy.NotFoundError):
            client.tasks.retrieve(name=name, type="other")

    def test_retrieve_or(self, client: sy.Synnax) -> None:
        """Should keep the tasks that match any filter in or_."""
        rack = client.racks.create(name=str(uuid4()))
        a = client.tasks.create(name="a", type="pagerduty_alert", rack=rack.key)
        b = client.tasks.create(name="b", type="pagerduty_alert", rack=rack.key)
        client.tasks.create(name="c", type="pagerduty_alert", rack=rack.key)
        res = client.tasks.retrieve(rack=rack.key, or_=[{"name": "a"}, {"key": b.key}])
        assert {t.key for t in res} == {a.key, b.key}

    def test_retrieve_none_field(self, client: sy.Synnax) -> None:
        """Should reject a field set to None, since it would select every task."""
        with pytest.raises(sy.ValidationError, match="rack is None"):
            client.tasks.retrieve(rack=None)  # type: ignore[call-overload]

    def test_retrieve_by_internal(self, client: sy.Synnax) -> None:
        """Should exclude user tasks when filtering for internal tasks."""
        rack = client.racks.create(name=str(uuid4()))
        task = client.tasks.create(name="test", type="pagerduty_alert", rack=rack.key)
        assert client.tasks.retrieve(rack=rack.key, internal=True) == []
        res = client.tasks.retrieve(rack=rack.key, internal=False)
        assert [t.key for t in res] == [task.key]

    def test_retrieve_where(self, client: sy.Synnax) -> None:
        """Should keep tasks that match a filter tree and the other filters."""
        rack = client.racks.create(name=str(uuid4()))
        tasks = [
            client.tasks.create(name=f"t{i}", type="pagerduty_alert", rack=rack.key)
            for i in range(3)
        ]
        where = task.Where(
            or_=[
                task.Where(name=filter.String(eq="t0")),
                task.Where(key=filter.Set[task.Key](eq=tasks[1].key)),
            ]
        )
        res = client.tasks.retrieve(where=where, rack=rack.key)
        assert {t.key for t in res} == {tasks[0].key, tasks[1].key}

    def test_retrieve_missing_key(self, client: sy.Synnax) -> None:
        """Should treat a missing key as a non-match in a list retrieve."""
        t = client.tasks.create(name=str(uuid4()), type="pagerduty_alert")
        res = client.tasks.retrieve(keys=[t.key, uuid4()])
        assert [x.key for x in res] == [t.key]

    def test_retrieve_not_found(self, client: sy.Synnax) -> None:
        """Should raise when a single task is requested and none matches."""
        with pytest.raises(sy.NotFoundError):
            client.tasks.retrieve(uuid4())

    def test_retrieve_with_limit(self, client: sy.Synnax) -> None:
        """Should return at most the limit of the tasks on a rack."""
        rack = client.racks.create(name=str(uuid4()))
        for i in range(3):
            client.tasks.create(name=f"t{i}", type="pagerduty_alert", rack=rack.key)
        assert len(client.tasks.retrieve(rack=rack.key, limit=2)) == 2

    def test_retrieve_by_search_term(self, client: sy.Synnax) -> None:
        """Should return the tasks whose names match the search term."""
        name = f"searchable-task-{uuid4()}"
        task = client.tasks.create(name=name, type="pagerduty_alert")

        def check() -> None:
            res = client.tasks.retrieve(search_term=name)
            assert task.key in {t.key for t in res}

        assert_eventually(check)

    def test_execute_command_sync(self, client: sy.Synnax) -> None:
        def driver(ev: threading.Event) -> None:
            with client.open_streamer("sy_task_cmd") as s:
                ev.set()
                f = s.read(timeout=1)
                assert f is not None
                cmd = sy.task.Command.model_validate(f["sy_task_cmd"][0])
                client.statuses.set(
                    sy.task.Status(
                        variant=sy.status.VARIANT_SUCCESS,
                        message="Command executed.",
                        details=sy.task.StatusDetails(running=False, cmd=cmd.key),
                        owner=sy.task.resource_id(cmd.task),
                    )
                )

        ev = threading.Event()
        t = threading.Thread(target=driver, args=(ev,))
        t.start()
        tsk = client.tasks.create(name="test", type="pagerduty_alert")
        ev.wait()
        tsk.execute_command_sync("test", args={"key": "value"})
        t.join()

    def test_execute_command_sync_timeout_is_a_deadline(
        self, client: sy.Synnax
    ) -> None:
        """A status answering nothing must not renew the wait for an answer."""
        tsk = client.tasks.create(name="test", type="pagerduty_alert")
        stop = threading.Event()

        def noise() -> None:
            while not stop.is_set():
                client.statuses.set(
                    sy.task.Status(
                        key=str(uuid4()),
                        variant=sy.status.VARIANT_INFO,
                        message="unrelated",
                    )
                )
                sy.sleep(0.1)

        t = threading.Thread(target=noise)
        t.start()
        try:
            timer = sy.Timer()
            with pytest.raises(TimeoutError):
                tsk.execute_command_sync("test", timeout=1)
            assert timer.elapsed() < 5 * sy.TimeSpan.SECOND
        finally:
            stop.set()
            t.join()

    def test_execute_command_sync_ends_when_the_driver_goes_down(
        self, client: sy.Synnax
    ) -> None:
        """A down Driver never replies, so the status of its rack ends the wait."""
        rack = client.racks.create(name=f"dying-{uuid4()}")
        tsk = client.tasks.create(name="test", type="pagerduty_alert", rack=rack.key)

        def driver(ev: threading.Event) -> None:
            with client.open_streamer("sy_task_cmd") as s:
                ev.set()
                f = s.read(timeout=5)
                assert f is not None
                client.statuses.set(
                    sy.rack.Status(
                        variant=sy.status.VARIANT_WARNING,
                        message="Driver not running",
                        description="The Core has not heard from the Driver.",
                        details=sy.rack.StatusDetails(),
                        owner=sy.rack.resource_id(rack.key),
                    )
                )

        ev = threading.Event()
        t = threading.Thread(target=driver, args=(ev,))
        t.start()
        ev.wait()
        timer = sy.Timer()
        with pytest.raises(RuntimeError) as err:
            tsk.execute_command_sync("test", timeout=60)
        assert str(err.value) == (
            "Driver not running: The Core has not heard from the Driver."
        )
        assert timer.elapsed() < 10 * sy.TimeSpan.SECOND
        t.join()

    @pytest.mark.parametrize(
        "variant",
        [sy.status.VARIANT_SUCCESS, sy.status.VARIANT_WARNING, sy.status.VARIANT_ERROR],
    )
    def test_execute_command_sync_waits_through_a_live_rack(
        self, client: sy.Synnax, variant: sy.status.Variant
    ) -> None:
        """A rack status from a running Driver, a warning or an error included, does
        not end the wait."""
        rack = client.racks.create(name=f"alive-{uuid4()}")
        tsk = client.tasks.create(name="test", type="pagerduty_alert", rack=rack.key)

        def driver(ev: threading.Event) -> None:
            with client.open_streamer("sy_task_cmd") as s:
                ev.set()
                f = s.read(timeout=5)
                assert f is not None
                cmd = sy.task.Command.model_validate(f["sy_task_cmd"][0])
                client.statuses.set(
                    sy.rack.Status(
                        variant=variant,
                        message="Driver is running",
                        details=sy.rack.StatusDetails(running=True),
                        owner=sy.rack.resource_id(rack.key),
                    )
                )
                client.statuses.set(
                    sy.task.Status(
                        variant=sy.status.VARIANT_SUCCESS,
                        message="Command executed.",
                        details=sy.task.StatusDetails(running=False, cmd=cmd.key),
                        owner=sy.task.resource_id(cmd.task),
                    )
                )

        ev = threading.Event()
        t = threading.Thread(target=driver, args=(ev,))
        t.start()
        ev.wait()
        status = tsk.execute_command_sync("test", timeout=60)
        assert status.message == "Command executed."
        t.join()

    def test_execute_command_sync_waits_past_the_timeout_after_a_loading_reply(
        self, client: sy.Synnax
    ) -> None:
        """A loading reply acknowledges the command, so the wait goes on until the
        final status even after the timeout."""
        tsk = client.tasks.create(name="test", type="pagerduty_alert")

        def driver(ev: threading.Event) -> None:
            with client.open_streamer("sy_task_cmd") as s:
                ev.set()
                f = s.read(timeout=5)
                assert f is not None
                cmd = sy.task.Command.model_validate(f["sy_task_cmd"][0])
                replies: list[tuple[sy.status.Variant, str]] = [
                    (sy.status.VARIANT_LOADING, "Starting task"),
                    (sy.status.VARIANT_SUCCESS, "Started task"),
                ]
                for variant, message in replies:
                    client.statuses.set(
                        sy.task.Status(
                            variant=variant,
                            message=message,
                            details=sy.task.StatusDetails(running=False, cmd=cmd.key),
                            owner=sy.task.resource_id(cmd.task),
                        )
                    )
                    sy.sleep(2)

        ev = threading.Event()
        t = threading.Thread(target=driver, args=(ev,))
        t.start()
        ev.wait()
        status = tsk.execute_command_sync("start", timeout=1)
        assert status.variant == sy.status.VARIANT_SUCCESS
        assert status.message == "Started task"
        t.join()

    def test_start_raises_when_the_driver_fails_the_start(
        self, client: sy.Synnax
    ) -> None:
        """start() raises the error status that the driver sends."""
        tsk = sy.pagerduty.AlertTask(
            client.tasks.create(name="test", type="pagerduty_alert")
        )
        ready = threading.Event()
        t = threading.Thread(
            target=_reply_to_next_command,
            args=(client, ready, sy.status.VARIANT_ERROR, "Failed", "slave 1 refused"),
        )
        t.start()
        ready.wait()
        with pytest.raises(RuntimeError) as err:
            tsk.start(timeout=5)
        assert str(err.value) == "Failed: slave 1 refused"
        t.join()

    def test_stop_returns_when_the_stop_status_carries_an_error(
        self, client: sy.Synnax
    ) -> None:
        """stop() returns on an error status, which carries an earlier error of the
        task."""
        tsk = sy.pagerduty.AlertTask(
            client.tasks.create(name="test", type="pagerduty_alert")
        )
        ready = threading.Event()
        t = threading.Thread(
            target=_reply_to_next_command,
            args=(client, ready, sy.status.VARIANT_ERROR, "Failed to read data", ""),
        )
        t.start()
        ready.wait()
        tsk.stop(timeout=5)
        t.join()

    @pytest.mark.parametrize("command", ["start", "stop"])
    def test_start_and_stop_return_when_the_driver_runs_the_command(
        self, client: sy.Synnax, command: str
    ) -> None:
        """start() and stop() return when the driver sends a success status."""
        tsk = sy.pagerduty.AlertTask(
            client.tasks.create(name="test", type="pagerduty_alert")
        )
        ready = threading.Event()
        t = threading.Thread(
            target=_reply_to_next_command,
            args=(client, ready, sy.status.VARIANT_SUCCESS, "Done", ""),
        )
        t.start()
        ready.wait()
        getattr(tsk, command)(timeout=5)
        t.join()

    def test_execute_command_sync_ends_when_no_final_status_follows_a_loading_reply(
        self, client: sy.Synnax, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        """A Driver that restarts after a loading reply never sends the final status,
        so the wait ends at its limit."""
        monkeypatch.setattr(
            "synnax.task.client._FINAL_STATUS_TIMEOUT", sy.TimeSpan.SECOND
        )
        tsk = client.tasks.create(name="test", type="pagerduty_alert")

        def driver(ev: threading.Event) -> None:
            with client.open_streamer("sy_task_cmd") as s:
                ev.set()
                f = s.read(timeout=5)
                assert f is not None
                cmd = sy.task.Command.model_validate(f["sy_task_cmd"][0])
                client.statuses.set(
                    sy.task.Status(
                        variant=sy.status.VARIANT_LOADING,
                        message="Starting task",
                        details=sy.task.StatusDetails(running=False, cmd=cmd.key),
                        owner=sy.task.resource_id(cmd.task),
                    )
                )

        ev = threading.Event()
        t = threading.Thread(target=driver, args=(ev,))
        t.start()
        ev.wait()
        timer = sy.Timer()
        with pytest.raises(TimeoutError) as err:
            tsk.execute_command_sync("start", timeout=0.5)
        assert str(err.value) == "timed out waiting for driver to finish start command"
        assert timer.elapsed() < 5 * sy.TimeSpan.SECOND
        t.join()

    def test_execute_command_without_args_sends_empty_object(
        self, client: sy.Synnax
    ) -> None:
        """Should send an empty object for args instead of null."""
        tsk = client.tasks.create(name="test", type="pagerduty_alert")
        with client.open_streamer("sy_task_cmd") as s:
            key = tsk.execute_command("test")
            matches: list[dict[str, Any]] = []
            for _ in range(10):
                f = s.read(timeout=1)
                assert f is not None, "timed out waiting for the command"
                matches = [
                    c
                    for c in f["sy_task_cmd"]
                    if isinstance(c, dict) and c["key"] == key
                ]
                if len(matches) > 0:
                    break
            assert matches[0]["args"] == {}

    def test_execute_command_converts_args_to_json(self, client: sy.Synnax) -> None:
        """Should send a UUID in the command args as a string."""
        tsk = client.tasks.create(name="test", type="pagerduty_alert")
        target = uuid4()
        with client.open_streamer("sy_task_cmd") as s:
            key = tsk.execute_command("test", args={"target": target})
            matches: list[dict[str, Any]] = []
            for _ in range(10):
                f = s.read(timeout=1)
                assert f is not None, "timed out waiting for the command"
                matches = [
                    c
                    for c in f["sy_task_cmd"]
                    if isinstance(c, dict) and c["key"] == key
                ]
                if len(matches) > 0:
                    break
            assert matches[0]["args"] == {"target": str(target)}

    def test_create_converts_dict_config_to_json(
        self, client: sy.Synnax, create_routing_key: Callable[[], UUID]
    ) -> None:
        """Should store a UUID in a dict config as a string."""
        ref = create_routing_key()
        task = client.tasks.create(
            name="test", type="pagerduty_alert", config={"routing_key": ref}
        )
        res = client.tasks.retrieve(key=task.key)
        assert res.config["routing_key"] == str(ref)

    def test_create_converts_model_config_by_alias(
        self, client: sy.Synnax, create_routing_key: Callable[[], UUID]
    ) -> None:
        """Should store a model config by its field aliases."""

        class Config(BaseModel):
            ref: UUID = Field(alias="routing_key")

        ref = create_routing_key()
        task = client.tasks.create(
            name="test", type="pagerduty_alert", config=Config(routing_key=ref)
        )
        res = client.tasks.retrieve(key=task.key)
        assert res.config["routing_key"] == str(ref)

    def test_task_converts_config_to_json(self) -> None:
        """Should convert a UUID in a Task config to a string."""
        ref = uuid4()
        assert sy.Task(config={"ref": ref}).to_payload().config == {"ref": str(ref)}

    def test_task_converts_reassigned_config_to_json(self) -> None:
        """Should convert a UUID in a config set after construction to a string."""
        tsk = sy.Task()
        ref = uuid4()
        tsk.config = {"ref": ref}
        assert tsk.to_payload().config == {"ref": str(ref)}

    def test_task_configure_saves_without_ack(
        self, client: sy.Synnax, create_routing_key: Callable[[], UUID]
    ) -> None:
        """Should save the task without waiting for a driver acknowledgement."""
        routing_key = str(create_routing_key())
        tsk = sy.Task(
            name="test", type="pagerduty_alert", config={"routing_key": routing_key}
        )
        client.tasks.configure(tsk)
        res = client.tasks.retrieve(key=tsk.key)
        assert res.key == tsk.key
        assert res.rack != 0
        assert res.config["routing_key"] == routing_key

    def test_task_configure_updates_config(
        self, client: sy.Synnax, create_routing_key: Callable[[], UUID]
    ) -> None:
        """Should set the name and config of a task that exists, and take its new
        config hash."""
        tsk = sy.Task(
            name="test",
            type="pagerduty_alert",
            config={"routing_key": str(create_routing_key())},
        )
        client.tasks.configure(tsk)
        routing_key = str(create_routing_key())
        tsk.name = "renamed"
        tsk.config = {"routing_key": routing_key}
        client.tasks.configure(tsk)
        res = client.tasks.retrieve(key=tsk.key)
        assert res.name == "renamed"
        assert res.config["routing_key"] == routing_key
        assert tsk.config_hash == res.config_hash

    def test_configure_puts_task_on_rack_of_device(self, client: sy.Synnax) -> None:
        """Should configure a task with no rack onto the rack of its device."""
        rack = client.racks.create(name=str(uuid4()))
        dev = client.devices.create(_device(rack.key))
        tsk = sy.Task(name="test", type="modbus_read", config={"device": dev.key})
        client.tasks.configure(tsk)
        assert tsk.rack == rack.key
        assert client.tasks.retrieve(key=tsk.key).rack == rack.key

    def test_configure_puts_task_on_rack_of_channel_devices(
        self, client: sy.Synnax
    ) -> None:
        """Should configure a task whose channels name its devices onto their rack."""
        rack = client.racks.create(name=str(uuid4()))
        dev = client.devices.create(_device(rack.key))
        tsk = sy.ni.AnalogReadTask(
            name="test",
            device=dev.key,
            channels=[_ai_channel(client, 0), _ai_channel(client, 1)],
        )
        client.tasks.configure(tsk)
        assert client.tasks.retrieve(key=tsk.key).rack == rack.key

    def test_configure_moves_task_to_rack_of_new_channel_devices(
        self, client: sy.Synnax
    ) -> None:
        """Should move a configured task to the rack of the devices that its channels
        name after a change."""
        a = client.devices.create(_device(client.racks.create(name="a").key))
        rack = client.racks.create(name=str(uuid4()))
        b = client.devices.create(_device(rack.key))
        tsk = sy.ni.AnalogReadTask(
            name="test", device=a.key, channels=[_ai_channel(client, 0)]
        )
        client.tasks.configure(tsk)
        tsk.config.channels[0].device = b.key
        client.tasks.configure(tsk)
        assert client.tasks.retrieve(key=tsk.key).rack == rack.key

    def test_configure_moves_task_to_rack_of_first_channel_device(
        self, client: sy.Synnax
    ) -> None:
        """Should move a task configured with no channels to the rack of the device
        that its first channel names."""
        rack = client.racks.create(name=str(uuid4()))
        dev = client.devices.create(_device(rack.key))
        tsk = sy.ni.AnalogReadTask(name="test", channels=[])
        client.tasks.configure(tsk)
        tsk.config.channels.append(_ai_channel(client, 0, device=dev.key))
        client.tasks.configure(tsk)
        assert client.tasks.retrieve(key=tsk.key).rack == rack.key

    def test_configure_rejects_devices_on_different_racks(
        self, client: sy.Synnax
    ) -> None:
        """Should raise when the channels of a task name devices on different
        racks."""
        a = client.devices.create(_device(client.racks.create(name="a").key))
        b = client.devices.create(_device(client.racks.create(name="b").key))
        tsk = sy.ni.AnalogReadTask(
            name="test",
            channels=[
                _ai_channel(client, 0, device=a.key),
                _ai_channel(client, 1, device=b.key),
            ],
        )
        with pytest.raises(sy.ValidationError, match="must be on one rack"):
            client.tasks.configure(tsk)

    def test_configure_puts_task_without_device_on_embedded_rack(
        self, client: sy.Synnax, create_routing_key: Callable[[], UUID]
    ) -> None:
        """Should configure a task whose config names no device onto the embedded
        rack."""
        tsk = sy.Task(
            name="test",
            type="pagerduty_alert",
            config={"routing_key": str(create_routing_key())},
        )
        client.tasks.configure(tsk)
        assert tsk.rack == client.racks.retrieve_embedded_rack().key

    def test_configure_puts_task_on_rack_of_device_over_set_rack(
        self, client: sy.Synnax
    ) -> None:
        """Should configure a task onto the rack of its device when the caller sets
        another rack."""
        rack = client.racks.create(name=str(uuid4()))
        dev = client.devices.create(_device(rack.key))
        tsk = sy.Task(
            name="test",
            type="modbus_read",
            rack=client.racks.create(name=str(uuid4())).key,
            config={"device": dev.key},
        )
        client.tasks.configure(tsk)
        assert client.tasks.retrieve(key=tsk.key).rack == rack.key

    def test_configure_moves_stale_task_to_rack_of_moved_device(
        self, client: sy.Synnax
    ) -> None:
        """Should configure a task object that holds an old rack onto the rack that
        its device moved to."""
        dev = client.devices.create(_device(client.racks.create(name="a").key))
        tsk = sy.Task(name="test", type="modbus_read", config={"device": dev.key})
        client.tasks.configure(tsk)
        rack = client.racks.create(name=str(uuid4()))
        client.devices.dispatch(
            dev.key, sy.device.set_rack(sy.device.SetRackPayload(rack=rack.key))
        )
        client.tasks.configure(client.tasks.retrieve(key=tsk.key))
        client.tasks.configure(tsk)
        assert client.tasks.retrieve(key=tsk.key).rack == rack.key

    def test_configure_moves_task_off_a_wrong_rack(self, client: sy.Synnax) -> None:
        """Should move a stored task that is not on the rack of its device onto that
        rack."""
        rack = client.racks.create(name=str(uuid4()))
        dev = client.devices.create(_device(rack.key))
        created = client.tasks.create(
            name="test",
            type="modbus_read",
            rack=client.racks.retrieve_embedded_rack().key,
            config={"device": dev.key},
        )
        tsk = client.tasks.retrieve(key=created.key)
        client.tasks.configure(tsk)
        assert client.tasks.retrieve(key=tsk.key).rack == rack.key

    def test_configure_keeps_rack_of_task_without_device(
        self, client: sy.Synnax, create_routing_key: Callable[[], UUID]
    ) -> None:
        """Should keep the rack of a configured task whose config names no device."""
        rack = client.racks.create(name=str(uuid4()))
        tsk = sy.Task(
            name="test",
            type="pagerduty_alert",
            rack=rack.key,
            config={"routing_key": str(create_routing_key())},
        )
        client.tasks.configure(tsk)
        client.tasks.configure(tsk)
        assert client.tasks.retrieve(key=tsk.key).rack == rack.key

    def test_configure_rejects_missing_device(self, client: sy.Synnax) -> None:
        """Should raise when the config names a device that does not exist."""
        tsk = sy.Task(name="test", type="modbus_read", config={"device": "missing"})
        with pytest.raises(
            sy.NotFoundError, match='Found no device where key = "missing"'
        ):
            client.tasks.configure(tsk)

    def test_list_tasks(self, client: sy.Synnax) -> None:
        """Should list all tasks on the default rack."""
        # Create some tasks
        task1 = client.tasks.create(name=str(uuid4()), type="pagerduty_alert")
        task2 = client.tasks.create(name=str(uuid4()), type="pagerduty_alert")

        # List all tasks
        tasks = client.tasks.list()

        # Should contain at least the tasks we just created
        task_keys = [t.key for t in tasks]
        assert task1.key in task_keys
        assert task2.key in task_keys

    def test_copy_task(
        self, client: sy.Synnax, create_routing_key: Callable[[], UUID]
    ) -> None:
        """Should copy a task with a new name."""
        # Create an original task
        original_name = str(uuid4())
        original = client.tasks.create(
            name=original_name,
            type="pagerduty_alert",
            config={"routing_key": str(create_routing_key())},
        )

        # Copy the task
        copy_name = str(uuid4())
        copied = client.tasks.copy(
            key=original.key,
            name=copy_name,
        )

        # Verify the copy
        assert copied.key != original.key
        assert copied.name == copy_name
        assert copied.type == original.type
        assert copied.config["routing_key"] == original.config["routing_key"]


@pytest.mark.task
class TestConfigBases:
    def test_read_config_mints_a_record_key(self) -> None:
        """Should give a read config a record key it can hash on."""
        cfg = sy.task.ReadConfig()
        assert isinstance(cfg.key, UUID)
        assert hash(cfg) == hash(cfg.key)

    def test_write_config_mints_a_record_key(self) -> None:
        """Should give a write config a record key it can hash on."""
        cfg = sy.task.WriteConfig()
        assert isinstance(cfg.key, UUID)
        assert hash(cfg) == hash(cfg.key)
