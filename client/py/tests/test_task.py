#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import threading
from uuid import UUID, uuid4

import pytest

import synnax as sy


@pytest.mark.task
class TestTaskClient:
    def test_create_single(self, client: sy.Synnax):
        task = client.tasks.create(name="test", type="pagerduty_alert")
        assert isinstance(task.key, UUID)
        assert task.rack != 0

    def test_create_multiple(self, client: sy.Synnax):
        t1 = sy.Task(name="test1", type="pagerduty_alert")
        t2 = sy.Task(name="test2", type="pagerduty_alert")
        tasks = client.tasks.create(tasks=[t1, t2])
        assert len(tasks) == 2
        assert tasks[0].name == "test1"
        assert tasks[1].name == "test2"

    def test_retrieve_by_name(self, client: sy.Synnax):
        name = str(uuid4())
        task = client.tasks.create(name=name, type="pagerduty_alert")
        res = client.tasks.retrieve(name=name)
        assert res.name == name
        assert res.key == task.key

    def test_retrieve_by_type(self, client: sy.Synnax):
        # Task types are a fixed, registered set, so the filter cannot be made unique
        # the way the name and model tests do. Assert membership instead of identity.
        task = client.tasks.create(type="labjack_scan")
        res = client.tasks.retrieve(types=["labjack_scan"])
        assert all(t.type == "labjack_scan" for t in res)
        assert task.key in {t.key for t in res}

    def test_retrieve_without_status(self, client: sy.Synnax):
        """Should leave the status unset when it is not asked for."""
        task = client.tasks.create(name=str(uuid4()), type="pagerduty_alert")
        assert client.tasks.retrieve(key=task.key).status is None

    def test_retrieve_with_status(self, client: sy.Synnax):
        """Should attach a parsed status when asked for one."""
        task = client.tasks.create(name=str(uuid4()), type="pagerduty_alert")
        res = client.tasks.retrieve(key=task.key, include_status=True)
        assert isinstance(res.status, sy.task.Status)
        assert res.status.details is not None
        assert res.status.details.task == task.key

    def test_execute_command_sync(self, client: sy.Synnax):
        def driver(ev: threading.Event):
            with client.open_streamer("sy_task_cmd") as s:
                ev.set()
                f = s.read(timeout=1)
                cmd = f["sy_task_cmd"][0]
                client.statuses.set(
                    sy.Status(
                        key=str(sy.task.ontology_id(cmd["task"])),
                        variant=sy.status.VARIANT_SUCCESS,
                        message="Command executed.",
                        details=sy.task.StatusDetails(
                            task=cmd["task"],
                            running=False,
                            cmd=cmd["key"],
                        ),
                    )
                )

        ev = threading.Event()
        t = threading.Thread(target=driver, args=(ev,))
        t.start()
        tsk = client.tasks.create(name="test", type="pagerduty_alert")
        ev.wait()
        tsk.execute_command_sync("test", {"key": "value"})
        t.join()

    def test_execute_command_without_args_sends_empty_object(self, client: sy.Synnax):
        """Should send an empty object for args instead of null."""
        tsk = client.tasks.create(name="test", type="pagerduty_alert")
        with client.open_streamer("sy_task_cmd") as s:
            key = tsk.execute_command("test")
            for _ in range(10):
                f = s.read(timeout=1)
                assert f is not None, "timed out waiting for the command"
                matches = [c for c in f["sy_task_cmd"] if c["key"] == key]
                if len(matches) > 0:
                    break
            assert matches[0]["args"] == {}

    def test_task_configure_saves_without_ack(self, client: sy.Synnax):
        """Should save the task without waiting for a driver acknowledgement."""
        tsk = sy.Task(
            name="test", type="pagerduty_alert", config={"routing_key": "rk-50"}
        )
        client.tasks.configure(tsk)
        res = client.tasks.retrieve(key=tsk.key)
        assert res.key == tsk.key
        assert res.rack != 0
        assert res.config["routing_key"] == "rk-50"

    def test_task_configure_updates_config(self, client: sy.Synnax):
        """Should overwrite the stored config when configured again."""
        tsk = sy.Task(
            name="test", type="pagerduty_alert", config={"routing_key": "rk-1"}
        )
        client.tasks.configure(tsk)
        tsk.config = {"routing_key": "rk-2"}
        client.tasks.configure(tsk)
        res = client.tasks.retrieve(key=tsk.key)
        assert res.config["routing_key"] == "rk-2"

    def test_list_tasks(self, client: sy.Synnax):
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

    def test_copy_task(self, client: sy.Synnax):
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
    def test_read_config_mints_a_record_key(self):
        """Should give a read config a record key it can hash on."""
        cfg = sy.task.ReadConfig()
        assert isinstance(cfg.key, UUID)
        assert hash(cfg) == hash(cfg.key)

    def test_write_config_mints_a_record_key(self):
        """Should give a write config a record key it can hash on."""
        cfg = sy.task.WriteConfig()
        assert isinstance(cfg.key, UUID)
        assert hash(cfg) == hash(cfg.key)
