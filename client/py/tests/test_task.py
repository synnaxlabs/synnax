#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import threading
from uuid import uuid4

import pytest

import synnax as sy
from synnax.status import ERROR_VARIANT, SUCCESS_VARIANT


@pytest.mark.task
class TestTaskClient:
    def test_create_single(self, client: sy.Synnax):
        task = client.hardware.tasks.create(name="test", type="test")
        assert task.key != 0

    def test_create_multiple(self, client: sy.Synnax):
        t1 = sy.Task(name="test1", type="test")
        t2 = sy.Task(name="test2", type="test")
        tasks = client.hardware.tasks.create(tasks=[t1, t2])
        assert len(tasks) == 2
        assert tasks[0].name == "test1"
        assert tasks[1].name == "test2"

    def test_retrieve_by_name(self, client: sy.Synnax):
        name = str(uuid4())
        task = client.hardware.tasks.create(name=name, type="test")
        res = client.hardware.tasks.retrieve(name=name)
        assert res.name == name
        assert res.key == task.key

    def test_retrieve_by_type(self, client: sy.Synnax):
        type = str(uuid4())
        task = client.hardware.tasks.create(type=type)
        res = client.hardware.tasks.retrieve(type=type)
        assert res.type == type
        assert res.key == task.key

    def test_execute_command_sync(self, client: sy.Synnax):
        def driver(ev: threading.Event):
            with client.open_streamer("sy_task_cmd") as s:
                with client.open_writer(sy.TimeStamp.now(), "sy_task_status") as w:
                    ev.set()
                    f = s.read(timeout=1)
                    cmd = f["sy_task_cmd"][0]
                    w.write(
                        "sy_task_status",
                        [
                            sy.TaskStatus(
                                key=cmd["key"],
                                variant=SUCCESS_VARIANT,
                                message="Command executed.",
                                details=sy.TaskStatusDetails(
                                    task=int(cmd["task"]),
                                ),
                            ).model_dump(),
                        ],
                    )

        ev = threading.Event()
        t = threading.Thread(target=driver, args=(ev,))
        t.start()
        tsk = client.hardware.tasks.create(name="test", type="test")
        ev.wait()
        tsk.execute_command_sync("test", {"key": "value"})
        t.join()

    def test_task_configure_success(self, client: sy.Synnax):
        """Should not throw an error when the task is configured successfully."""

        def driver(ev: threading.Event):
            with client.open_streamer("sy_task_set") as s:
                with client.open_writer(sy.TimeStamp.now(), "sy_task_status") as w:
                    ev.set()
                    f = s.read(timeout=2)
                    key = f["sy_task_set"][0]
                    w.write(
                        "sy_task_status",
                        [
                            sy.TaskStatus(
                                variant=SUCCESS_VARIANT,
                                message="Task configured.",
                                details=sy.TaskStatusDetails(task=int(key)),
                            ).model_dump(),
                        ],
                    )

        tsk = sy.Task()
        ev = threading.Event()
        t = threading.Thread(target=driver, args=(ev,))
        t.start()
        ev.wait()
        client.hardware.tasks.configure(tsk)
        t.join()

    def test_task_configure_invalid_config(self, client: sy.Synnax):
        """Should throw an error when the driver responds with an error"""

        def driver(ev: threading.Event):
            with client.open_streamer("sy_task_set") as s:
                with client.open_writer(sy.TimeStamp.now(), "sy_task_status") as w:
                    ev.set()
                    f = s.read(timeout=1)
                    key = f["sy_task_set"][0]
                    w.write(
                        "sy_task_status",
                        [
                            sy.TaskStatus(
                                variant=ERROR_VARIANT,
                                message="Invalid Configuration.",
                                details=sy.TaskStatusDetails(task=int(key)),
                            ).model_dump(),
                        ],
                    )

        tsk = sy.Task()
        ev = threading.Event()
        t = threading.Thread(target=driver, args=(ev,))
        t.start()
        ev.wait()
        with pytest.raises(sy.ConfigurationError, match="Invalid Configuration."):
            client.hardware.tasks.configure(tsk)
        t.join()

    def test_task_configure_timeout(self, client: sy.Synnax):
        """Should throw an error when the task is not configured within the timeout."""
        tsk = sy.Task()
        with pytest.raises(TimeoutError):
            client.hardware.tasks.configure(tsk, timeout=0.1)

    def test_list_tasks(self, client: sy.Synnax):
        """Should list all tasks on the default rack."""
        # Create some tasks
        task1 = client.hardware.tasks.create(name=str(uuid4()), type="test1")
        task2 = client.hardware.tasks.create(name=str(uuid4()), type="test2")

        # List all tasks
        tasks = client.hardware.tasks.list()

        # Should contain at least the tasks we just created
        task_keys = [t.key for t in tasks]
        assert task1.key in task_keys
        assert task2.key in task_keys

    def test_copy_task(self, client: sy.Synnax):
        """Should copy a task with a new name."""
        # Create an original task
        original_name = str(uuid4())
        original = client.hardware.tasks.create(
            name=original_name, type="test", config='{"foo": "bar"}'
        )

        # Copy the task
        copy_name = str(uuid4())
        copied = client.hardware.tasks.copy(
            key=original.key,
            name=copy_name,
        )

        # Verify the copy
        assert copied.key != original.key
        assert copied.name == copy_name
        assert copied.type == original.type
        assert copied.config == original.config

    def test_rack_assignment_from_device(self, client: sy.Synnax):
        """Should assign task to the same rack as its device."""
        # Get the embedded rack
        embedded_rack = client.hardware.racks.retrieve_embedded_rack()

        # Create a device on the embedded rack
        device = client.hardware.devices.create(
            key=str(uuid4()),
            name=str(uuid4()),
            rack=embedded_rack.key,
            location="test",
            model="test_model",
        )

        # Create a task with device in config
        task = sy.Task(
            name=str(uuid4()),
            type="test",
            config=f'{{"device": "{device.key}"}}',
        )

        # Configure the task - should extract rack from device
        def driver(ev: threading.Event):
            with client.open_streamer("sy_task_set") as s:
                with client.open_writer(sy.TimeStamp.now(), "sy_task_status") as w:
                    ev.set()
                    f = s.read(timeout=2)
                    task_key = f["sy_task_set"][0]
                    w.write(
                        "sy_task_status",
                        [
                            sy.TaskStatus(
                                variant=SUCCESS_VARIANT,
                                message="Task configured.",
                                details=sy.TaskStatusDetails(task=int(task_key)),
                            ).model_dump(),
                        ],
                    )

        ev = threading.Event()
        t = threading.Thread(target=driver, args=(ev,))
        t.start()
        ev.wait()
        configured_task = client.hardware.tasks.configure(task)
        t.join()

        # Extract rack from task key (upper 32 bits)
        task_rack_key = configured_task.key >> 32
        assert task_rack_key == embedded_rack.key

    def test_rack_assignment_fallback_device_not_found(self, client: sy.Synnax):
        """Should fallback to default rack gracefully when device not found."""
        # Get the embedded rack
        embedded_rack = client.hardware.racks.retrieve_embedded_rack()

        # Create a task with non-existent device in config
        task = sy.Task(
            name=str(uuid4()),
            type="test",
            config='{"device": "non-existent-device-key"}',
        )

        # Configure the task - should fall back to default rack
        def driver(ev: threading.Event):
            with client.open_streamer("sy_task_set") as s:
                with client.open_writer(sy.TimeStamp.now(), "sy_task_status") as w:
                    ev.set()
                    f = s.read(timeout=2)
                    task_key = f["sy_task_set"][0]
                    w.write(
                        "sy_task_status",
                        [
                            sy.TaskStatus(
                                variant=SUCCESS_VARIANT,
                                message="Task configured.",
                                details=sy.TaskStatusDetails(task=int(task_key)),
                            ).model_dump(),
                        ],
                    )

        ev = threading.Event()
        t = threading.Thread(target=driver, args=(ev,))
        t.start()
        ev.wait()
        configured_task = client.hardware.tasks.configure(task)
        t.join()

        # Should still get a valid task key (with default rack)
        assert configured_task.key != 0
        task_rack_key = configured_task.key >> 32
        assert task_rack_key == embedded_rack.key

    def test_rack_assignment_no_device_in_config(self, client: sy.Synnax):
        """Should use default rack when no device specified in config."""
        # Get the embedded rack
        embedded_rack = client.hardware.racks.retrieve_embedded_rack()

        # Create a task with no device in config
        task = sy.Task(
            name=str(uuid4()),
            type="test",
            config='{"some_other_field": "value"}',
        )

        # Configure the task - should use default rack
        def driver(ev: threading.Event):
            with client.open_streamer("sy_task_set") as s:
                with client.open_writer(sy.TimeStamp.now(), "sy_task_status") as w:
                    ev.set()
                    f = s.read(timeout=2)
                    task_key = f["sy_task_set"][0]
                    w.write(
                        "sy_task_status",
                        [
                            sy.TaskStatus(
                                variant=SUCCESS_VARIANT,
                                message="Task configured.",
                                details=sy.TaskStatusDetails(task=int(task_key)),
                            ).model_dump(),
                        ],
                    )

        ev = threading.Event()
        t = threading.Thread(target=driver, args=(ev,))
        t.start()
        ev.wait()
        configured_task = client.hardware.tasks.configure(task)
        t.join()

        # Extract rack from task key (upper 32 bits)
        task_rack_key = configured_task.key >> 32
        assert task_rack_key == embedded_rack.key
