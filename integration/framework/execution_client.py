#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import ctypes
import random
import threading
import traceback
from collections.abc import Callable
from concurrent.futures import Future

import synnax as sy

from framework.config_client import ConfigClient, Sequence, TestDefinition
from framework.models import STATUS, SynnaxConnection, Test
from framework.test_case import TestCase

# Range coloring
COLORS: list[str] = [
    "#001833",
    "#003333",
    "#003318",
    "#223322",
    "#183318",
    "#333300",
    "#331800",
    "#330000",
    "#330018",
    "#330033",
    "#180033",
    "#000033",
]


class ExecutionClient:
    """Executes test sequences with timeout monitoring and thread management."""

    def __init__(
        self,
        config_client: ConfigClient,
        synnax_connection: SynnaxConnection,
        client: sy.Synnax,
        conductor_range: sy.Range | None,
        tests: list[Test],
        tests_lock: threading.Lock,
        active_tests: list[tuple[TestDefinition, TestCase, sy.Range, threading.Thread]],
        active_tests_lock: threading.Lock,
        log: Callable[[str, bool], None],
        on_status_change: Callable[[Test], None],
        on_test_ran: Callable[[], None],
    ) -> None:
        self._config_client = config_client
        self._synnax_connection = synnax_connection
        self._client = client
        self._conductor_range = conductor_range
        self._tests = tests
        self._tests_lock = tests_lock
        self._active_tests = active_tests
        self._active_tests_lock = active_tests_lock
        self._log = log
        self._on_status_change = on_status_change
        self._on_test_ran = on_test_ran
        self.should_stop = False
        self.is_running = False
        self._total_tests = 0
        self._tests_offset = 0
        self._timeout_thread: threading.Thread | None = None

    def run(self, sequences: list[Sequence]) -> None:
        """Execute all sequences."""
        self.is_running = True
        self.should_stop = False
        self._total_tests = sum(len(seq.tests) for seq in sequences)
        self._tests_offset = 0

        self._timeout_thread = threading.Thread(
            target=self._timeout_monitor_loop,
            args=(1.0,),
            daemon=True,
        )
        self._timeout_thread.start()

        self._log(
            f"Starting execution of {len(sequences)} sequences "
            f"with {self._total_tests} total tests...\n",
            True,
        )

        for seq_idx, sequence in enumerate(sequences):
            if self.should_stop:
                self._log("Test execution stopped by user request", True)
                break

            self._log(
                f"==== SEQUENCE {seq_idx + 1}/{len(sequences)}: {sequence.name} ====",
                True,
            )
            self._log(
                f"Executing {len(sequence.tests)} tests with {sequence.order} order...\n",
                True,
            )

            tests_to_execute = sequence.tests.copy()

            if sequence.order == "asynchronous":
                self._execute_async(sequence.name, tests_to_execute, sequence.pool_size)
            else:
                if sequence.order == "random":
                    random.shuffle(tests_to_execute)
                    self._log("Tests randomized for execution", True)
                self._execute_sequential(tests_to_execute)

            self._log(f"Completed sequence '{sequence.name}'\n", True)

        if not self.should_stop:
            self._retry_failed(sequences)

        self.is_running = False

        if self._timeout_thread is not None and self._timeout_thread.is_alive():
            self._timeout_thread.join(timeout=2.0)

    def stop(self) -> None:
        """Signal stop and kill active tests."""
        self._log("Stopping test sequence...", True)
        self.should_stop = True
        killed = self.kill_active_tests()
        if killed > 0:
            self._log(f"Killed {killed} active test(s)", True)

    def _retry_failed(self, sequences: list[Sequence]) -> None:
        all_defs: dict[str, TestDefinition] = {}
        for seq in sequences:
            for td in seq.tests:
                key = str(td)
                all_defs[key] = td

        with self._tests_lock:
            failed = [
                t for t in self._tests if t.status in (STATUS.FAILED, STATUS.TIMEOUT)
            ]

        if not failed:
            return

        retry_defs: list[TestDefinition] = []
        for result in failed:
            key = str(result)
            if key in all_defs:
                retry_defs.append(all_defs[key])

        if not retry_defs:
            return

        self._log(f"==== RETRYING {len(retry_defs)} FAILED TEST(S) ====\n", True)

        retry_keys = {str(td) for td in retry_defs}

        with self._tests_lock:
            for result in failed:
                self._tests.remove(result)
            self._tests_offset = len(self._tests)

        self._total_tests = len(retry_defs)
        self._execute_sequential(retry_defs)

        with self._tests_lock:
            for test in self._tests:
                if str(test) in retry_keys and test.status == STATUS.PASSED:
                    test.status = STATUS.FLAKY
                    self._on_status_change(test)

    # ----- Sequential execution -----

    def _execute_sequential(self, tests: list[TestDefinition]) -> None:
        for test_def in tests:
            if self.should_stop:
                self._log("Test execution stopped by user request", True)
                break

            global_idx = len(self._tests) - self._tests_offset + 1
            self._log(f"[{global_idx}/{self._total_tests}] {test_def}", True)

            future: Future[Test] = Future()
            t = threading.Thread(
                target=self._test_runner_thread,
                args=(test_def, future),
            )
            t.start()
            while t.is_alive():
                t.join(timeout=1.0)

            self._collect_result(test_def, future)

    # ----- Async execution -----

    def _execute_async(
        self,
        seq_name: str,
        tests: list[TestDefinition],
        pool_size: int = -1,
    ) -> None:
        use_pool = 0 < pool_size < len(tests)
        semaphore = threading.Semaphore(pool_size) if use_pool else None

        if use_pool:
            self._log(
                f"Running tests with pool size of {pool_size} "
                f"(max {pool_size} concurrent tests)...",
                True,
            )

        threads: list[threading.Thread] = []
        futures: list[Future[Test]] = []

        def _run(td: TestDefinition, ft: Future[Test], idx: int) -> None:
            if semaphore is not None:
                semaphore.acquire()
            try:
                if semaphore is not None:
                    self._log(f"[{idx}/{self._total_tests}] {td}", True)
                self._test_runner_thread(td, ft)
            finally:
                if semaphore is not None:
                    semaphore.release()

        for i, test_def in enumerate(tests):
            if self.should_stop:
                self._log("Test execution stopped by user request", True)
                break

            global_idx = len(self._tests) - self._tests_offset + i + 1
            if semaphore is None:
                self._log(f"[{global_idx}/{self._total_tests}] {test_def}", True)

            future: Future[Test] = Future()
            t = threading.Thread(
                target=_run,
                args=(test_def, future, global_idx),
            )
            threads.append(t)
            futures.append(future)
            t.start()

        self._log(
            f"Waiting for {len(threads)} tests in sequence '{seq_name}' to complete...",
            True,
        )
        pending = list(zip(threads, futures, tests))
        while pending:
            still_pending = []
            for t, ft, td in pending:
                if not t.is_alive():
                    self._collect_result(td, ft)
                else:
                    still_pending.append((t, ft, td))
            pending = still_pending
            if pending:
                pending[0][0].join(timeout=1.0)

    # ----- Single test execution -----

    def _collect_result(self, test_def: TestDefinition, future: Future[Test]) -> None:
        with self._tests_lock:
            already_recorded = any(str(r) == str(test_def) for r in self._tests)
        if already_recorded:
            return

        if future.done():
            result = future.result()
        else:
            result = Test(
                test_name=test_def.case,
                name=test_def.display_name,
                status=STATUS.TIMEOUT,
                error_message="Test was terminated due to timeout",
            )

        with self._tests_lock:
            self._tests.append(result)
        self._on_test_ran()

    def _test_runner_thread(
        self, test_def: TestDefinition, future: Future[Test]
    ) -> None:
        result = self._execute_single_test(test_def)
        future.set_result(result)

    def _execute_single_test(self, test_def: TestDefinition) -> Test:
        test = Test(
            test_name=test_def.case,
            name=test_def.display_name,
            status=STATUS.PENDING,
        )

        test_index = len(self._tests)
        color = COLORS[test_index % len(COLORS)]

        if self._conductor_range is not None:
            test.range = self._conductor_range.create_child_range(
                name=test.name or test.test_name,
                time_range=sy.TimeRange(start=sy.TimeStamp.now(), end=sy.TimeStamp.MAX),
                color=color,
            )
        else:
            test.range = None

        test_instance: TestCase | None = None
        try:
            test_class = self._config_client.load_test_class(test_def)
            test_instance = test_class(
                synnax_connection=self._synnax_connection,
                name=test_def.display_name,
                **test_def.parameters,
            )

            current_thread = threading.current_thread()
            with self._active_tests_lock:
                self._active_tests.append(
                    (test_def, test_instance, test.range, current_thread)
                )

            test.status = STATUS.RUNNING
            self._on_status_change(test)

            test_instance.execute()
            test.status = test_instance._status

        except Exception as e:
            test.status = STATUS.FAILED
            test.error_message = str(e)
            self._log(f"{test_def.case} FAILED: {e}", True)
            self._log(f"Traceback: {traceback.format_exc()}", True)

        finally:
            if test.range is not None:
                try:
                    test.range = self._finalize_range(test.range)
                except RuntimeError as e:
                    self._log(f"Warning: Could not finalize range: {e}", True)

            if test_instance is not None and test_instance._status != STATUS.KILLED:
                if test.status in (
                    STATUS.FAILED,
                    STATUS.TIMEOUT,
                ) or test_instance._status in (STATUS.FAILED, STATUS.TIMEOUT):
                    test_instance.log_client.dump()
                else:
                    test_instance.log_client.discard()
                test_instance.log_client.close()

            with self._active_tests_lock:
                self._active_tests[:] = [
                    (td, t, tr, th)
                    for td, t, tr, th in self._active_tests
                    if t != test_instance
                ]

            self._on_status_change(test)

        return test

    def _finalize_range(self, test_range: sy.Range) -> sy.Range:
        try:
            return self._client.ranges.create(
                key=test_range.key,
                name=test_range.name,
                time_range=sy.TimeRange(
                    start=test_range.time_range.start,
                    end=sy.TimeStamp.now(),
                ),
            )
        except Exception as e:
            raise RuntimeError(
                f"Failed to finalize range '{test_range.name}': {e}"
            ) from e

    # ----- Timeout monitoring -----

    def _timeout_monitor_loop(
        self,
        monitor_interval: sy.CrudeTimeSpan = 500 * sy.TimeSpan.MILLISECOND,
    ) -> None:
        while self.is_running and not self.should_stop:
            self._check_test_timeouts()
            sy.sleep(monitor_interval)

    def _check_test_timeouts(self) -> None:
        threads_to_terminate: list[threading.Thread] = []

        with self._active_tests_lock:
            if not self._active_tests:
                return

            to_remove = []
            for _test_def, test_instance, test_range, thread in self._active_tests:
                expected: sy.CrudeTimeSpan | None = getattr(
                    test_instance, "Expected_Timeout", None
                )
                if expected is None:
                    continue

                elapsed = sy.TimeStamp.now() - test_range.time_range.start
                timeout_span = sy.TimeSpan.from_seconds(expected)
                if elapsed <= timeout_span:
                    continue

                self._log(
                    f"{test_instance.name} timeout detected "
                    f"({elapsed} > {timeout_span})",
                    True,
                )
                test_instance._status = STATUS.TIMEOUT
                to_remove.append((_test_def, test_instance, test_range, thread))
                threads_to_terminate.append(thread)

            for item in to_remove:
                self._active_tests.remove(item)

        for thread in threads_to_terminate:
            self._terminate_thread(thread)

    # ----- Kill / terminate -----

    def kill_active_tests(self) -> int:
        with self._active_tests_lock:
            if not self._active_tests:
                return 0

            killed_results = []
            threads_to_terminate = []
            killed_instances = []

            for test_def, test_instance, test_range, thread in self._active_tests:
                if test_range is None:
                    status = STATUS.KILLED
                    error_msg = "Test was killed (no range available)"
                    finalized_range = None
                else:
                    status, error_msg = self._determine_kill_status(
                        test_instance, test_range
                    )
                    try:
                        finalized_range = self._finalize_range(test_range)
                    except RuntimeError as e:
                        self._log(
                            f"Warning: Could not finalize range for killed test: {e}",
                            True,
                        )
                        finalized_range = test_range

                test_instance._status = status
                killed_results.append(
                    Test(
                        test_name=test_def.case,
                        name=test_def.display_name,
                        status=status,
                        error_message=error_msg,
                        range=finalized_range,
                    )
                )
                threads_to_terminate.append(thread)
                killed_instances.append(test_instance)

            self._active_tests.clear()

        with self._tests_lock:
            self._tests.extend(killed_results)

        for thread in threads_to_terminate:
            self._terminate_thread(thread)

        for instance in killed_instances:
            self._log(f"--- Logs for {instance.name} ---", True)
            instance.log_client.dump()
            instance.log_client.close()
            self._log(f"--- End logs for {instance.name} ---", True)

        return len(killed_results)

    def _determine_kill_status(
        self, test_instance: TestCase, test_range: sy.Range
    ) -> tuple[STATUS, str]:
        elapsed_time = (
            sy.TimeStamp.now() - test_range.time_range.start
        ) / sy.TimeSpan.SECOND

        expected_timeout = getattr(test_instance, "Expected_Timeout", -1)

        if expected_timeout > 0 and elapsed_time > expected_timeout:
            self._log(
                f"Test {test_instance.name} exceeded timeout ({expected_timeout}s)",
                True,
            )
            return STATUS.TIMEOUT, f"Test exceeded timeout ({expected_timeout}s)"

        self._log(f"Test {test_instance.name} was manually killed", True)
        return STATUS.KILLED, "Test was manually killed"

    def _terminate_thread(self, thread: threading.Thread) -> None:
        thread.join(timeout=0.1)
        if not thread.is_alive():
            return

        try:
            thread_id = thread.ident
            if thread_id is None:
                return

            res = ctypes.pythonapi.PyThreadState_SetAsyncExc(
                ctypes.c_long(thread_id), ctypes.py_object(SystemExit)
            )
            if res == 0:
                self._log(f"Warning: Could not terminate thread {thread.name}", True)
            elif res > 1:
                ctypes.pythonapi.PyThreadState_SetAsyncExc(
                    ctypes.c_long(thread_id), None
                )
        except Exception as e:
            self._log(f"Warning: Failed to force-terminate thread: {e}", True)
