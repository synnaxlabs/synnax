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
from typing import Any, assert_type, cast
from uuid import UUID, uuid4

import pytest
from pydantic import BaseModel, Field

import synnax as sy
from x.testutil import assert_eventually


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

    def test_rename(self, client: sy.Synnax) -> None:
        """Should rename a task and keep its config."""
        task = client.tasks.create(
            name="Before Rename",
            type="pagerduty_alert",
            config={"routing_key": "abc"},
        )
        client.tasks.rename(task.key, "After Rename")
        renamed = client.tasks.retrieve(key=task.key)
        assert renamed.name == "After Rename"
        assert renamed.config == task.config
        assert renamed.config["routing_key"] == "abc"

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
        assert res.status.details.task == task.key

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
        """Should return one task when a key, name, or type is given."""
        name = str(uuid4())
        task = client.tasks.create(name=name, type="pagerduty_alert")
        by_key = client.tasks.retrieve(key=task.key, include_status=True)
        assert_type(by_key, sy.Task)
        assert by_key.key == task.key
        by_name = client.tasks.retrieve(name=name, include_status=True)
        assert_type(by_name, sy.Task)
        assert by_name.key == task.key
        by_type = client.tasks.retrieve(type="pagerduty_alert", include_status=True)
        assert_type(by_type, sy.Task)
        assert by_type.type == "pagerduty_alert"

    def test_retrieve_no_filters(self, client: sy.Synnax) -> None:
        """Should return a list of every task when no filter is given."""
        task = client.tasks.create(name="test", type="pagerduty_alert")
        res = client.tasks.retrieve()
        assert_type(res, list[sy.Task])
        assert task.key in {t.key for t in res}

    def test_retrieve_singular_with_list_filter(self, client: sy.Synnax) -> None:
        """Should raise a ValueError when singular selectors are mixed with others."""
        retrieve = cast(Callable[..., object], client.tasks.retrieve)
        with pytest.raises(ValueError, match="cannot be combined"):
            retrieve(name="test", rack=1)
        with pytest.raises(ValueError, match="only one of"):
            retrieve(name="test", type="pagerduty_alert")

    def test_retrieve_by_internal_and_snapshot(self, client: sy.Synnax) -> None:
        """Should exclude user tasks when filtering for internal or snapshot tasks."""
        rack = client.racks.create(name=str(uuid4()))
        task = client.tasks.create(name="test", type="pagerduty_alert", rack=rack.key)
        assert client.tasks.retrieve(rack=rack.key, internal=True) == []
        assert client.tasks.retrieve(rack=rack.key, snapshot=True) == []
        res = client.tasks.retrieve(rack=rack.key, internal=False, snapshot=False)
        assert [t.key for t in res] == [task.key]

    def test_retrieve_with_limit_and_offset(self, client: sy.Synnax) -> None:
        """Should page through the tasks on a rack."""
        rack = client.racks.create(name=str(uuid4()))
        for i in range(3):
            client.tasks.create(name=f"t{i}", type="pagerduty_alert", rack=rack.key)
        assert len(client.tasks.retrieve(rack=rack.key, limit=2)) == 2
        assert len(client.tasks.retrieve(rack=rack.key, offset=2)) == 1

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
                        key=str(sy.task.ontology_id(cmd.task)),
                        variant=sy.status.VARIANT_SUCCESS,
                        message="Command executed.",
                        details=sy.task.StatusDetails(
                            task=cmd.task,
                            running=False,
                            cmd=cmd.key,
                        ),
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

    def test_create_converts_dict_config_to_json(self, client: sy.Synnax) -> None:
        """Should store a UUID in a dict config as a string."""
        ref = uuid4()
        task = client.tasks.create(
            name="test", type="pagerduty_alert", config={"routing_key": ref}
        )
        res = client.tasks.retrieve(key=task.key)
        assert res.config["routing_key"] == str(ref)

    def test_create_converts_model_config_by_alias(self, client: sy.Synnax) -> None:
        """Should store a model config by its field aliases."""

        class Config(BaseModel):
            ref: UUID = Field(alias="routing_key")

        ref = uuid4()
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

    def test_task_configure_saves_without_ack(self, client: sy.Synnax) -> None:
        """Should save the task without waiting for a driver acknowledgement."""
        tsk = sy.Task(
            name="test", type="pagerduty_alert", config={"routing_key": "rk-50"}
        )
        client.tasks.configure(tsk)
        res = client.tasks.retrieve(key=tsk.key)
        assert res.key == tsk.key
        assert res.rack != 0
        assert res.config["routing_key"] == "rk-50"

    def test_task_configure_updates_config(self, client: sy.Synnax) -> None:
        """Should overwrite the stored config when configured again."""
        tsk = sy.Task(
            name="test", type="pagerduty_alert", config={"routing_key": "rk-1"}
        )
        client.tasks.configure(tsk)
        tsk.config = {"routing_key": "rk-2"}
        client.tasks.configure(tsk)
        res = client.tasks.retrieve(key=tsk.key)
        assert res.config["routing_key"] == "rk-2"

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

    def test_copy_task(self, client: sy.Synnax) -> None:
        """Should copy a task with a new name."""
        # Create an original task
        original_name = str(uuid4())
        original = client.tasks.create(
            name=original_name, type="pagerduty_alert", config={"routing_key": "rk-c"}
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
