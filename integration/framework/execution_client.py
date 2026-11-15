#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random
import threading
import time
import traceback
from collections.abc import Callable
from concurrent.futures import Future
from pathlib import Path

import synnax as sy
from framework import resources, server_log
from framework.config_client import ConfigClient, Sequence, TestDefinition
from framework.cores import Core
from framework.models import STATUS, Log, RunOptions, Test
from framework.resources import Resources, Unavailable
from framework.test_case import TestCase
from framework.watchdog import Slots, Timeout, Watchdog, interrupt

# Range coloring
_COLORS: list[str] = [
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

_TIMEOUT_GRACE = 60.0

# Seconds a test can wait for its resources before the conductor logs the wait.
_WAIT_LOGGED = 1.0


class ExecutionClient:
    """Executes test sequences with timeout monitoring and thread management. Each
    sequence runs on one Core, and as many sequences run at once as there are
    Cores."""

    def __init__(
        self,
        *,
        config_client: ConfigClient,
        cores: list[Core],
        resources: Resources,
        options: RunOptions,
        run_dir: Path,
        client: sy.Synnax,
        conductor_range: sy.Range | None,
        tests: list[Test],
        tests_lock: threading.Lock,
        active_tests: list[
            tuple[TestDefinition, TestCase, sy.Range | None, threading.Thread]
        ],
        active_tests_lock: threading.Lock,
        log: Log,
        on_status_change: Callable[[Test], None],
        on_test_ran: Callable[[], None],
        timeout_grace: float = _TIMEOUT_GRACE,
    ) -> None:
        """:param cores: The Cores to run tests on. The first one is the only Core
        whose Driver reads NI devices.
        :param timeout_grace: Seconds past its timeout a test has to finish before the
        client stops waiting on it.
        """
        self._config_client = config_client
        self._cores = cores
        self._resources = resources
        self._options = options
        self._run_dir = run_dir
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
        self._watchdog = Watchdog(timeout_grace)
        self._abandoned: dict[threading.Thread, str] = {}

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
            f"with {self._total_tests} total tests on {len(self._cores)} "
            f"{'Core' if len(self._cores) == 1 else 'Cores'}...\n"
        )

        self._run_sequences(sequences)

        if not self.should_stop:
            self._retry_failed(sequences)

        self.is_running = False

        if self._timeout_thread.is_alive():
            self._timeout_thread.join(timeout=2.0)

    def stop(self) -> None:
        """Signal stop and kill active tests."""
        self._log("Stopping test sequence...")
        self.should_stop = True
        killed = self.kill_active_tests()
        if killed > 0:
            self._log(f"Killed {killed} active test(s)")

    def _run_sequences(self, sequences: list[Sequence]) -> None:
        """Runs the sequences in order on the first free Core that fits them, then
        runs the isolated sequences one at a time."""
        pending = [s for s in sequences if not s.isolated]
        lock = threading.Lock()
        errors: list[BaseException] = []

        def lane(core: Core) -> None:
            try:
                while not self.should_stop and not errors:
                    with lock:
                        seq = next((s for s in pending if self._fits(s, core)), None)
                        if seq is None:
                            return
                        pending.remove(seq)
                    self._run_sequence(seq, core)
            except BaseException as e:
                errors.append(e)

        lanes = [
            threading.Thread(target=lane, args=(core,), name=f"lane {core.address}")
            for core in self._cores
        ]
        for t in lanes:
            t.start()
        for t in lanes:
            while t.is_alive():
                t.join(timeout=1.0)
        if errors:
            raise errors[0]

        for seq in sequences:
            if not seq.isolated or self.should_stop:
                continue
            with self._tests_lock:
                running = sorted(
                    {n for t, n in self._abandoned.items() if t.is_alive()}
                )
            if not running:
                self._run_sequence(seq, self._cores[0])
                continue
            message = f"Abandoned tests still run: {', '.join(running)}"
            self._log(f"Skipping isolated sequence '{seq.name}'. {message}")
            for td in seq.tests:
                self._record(td, self._result(td, STATUS.FAILED, message))

    def _fits(self, sequence: Sequence, core: Core) -> bool:
        if core is self._cores[0]:
            return True
        return not any(
            resources.NI in self._config_client.resources_of(td)
            for td in sequence.tests
        )

    def _run_sequence(self, sequence: Sequence, core: Core) -> None:
        if self.should_stop:
            self._log("Test execution stopped by user request")
            return
        self._log(f"==== SEQUENCE {sequence.name} on {core.address} ====")
        self._log(
            f"Executing {len(sequence.tests)} tests with {sequence.order} order...\n"
        )
        tests = sequence.tests.copy()
        if sequence.order == "asynchronous":
            self._execute_async(sequence.name, tests, sequence.pool_size, core)
        else:
            if sequence.order == "random":
                random.shuffle(tests)
                self._log("Tests randomized for execution")
            self._execute_sequential(tests, core)
        self._log(f"Completed sequence '{sequence.name}'\n")

    def _retry_failed(self, sequences: list[Sequence]) -> None:
        with self._tests_lock:
            failed = [
                t for t in self._tests if t.status in (STATUS.FAILED, STATUS.TIMEOUT)
            ]

        if not failed:
            return

        failed_keys = {str(result) for result in failed}
        # A test in an asynchronous sequence can depend on its peers running at the
        # same time, so each retry keeps its own sequence's order.
        retries: list[Sequence] = []
        retry_keys: set[str] = set()
        for seq in sequences:
            defs = [
                td
                for td in seq.tests
                if str(td) in failed_keys and str(td) not in retry_keys
            ]
            if len(defs) == 0:
                continue
            retry_keys.update(str(td) for td in defs)
            retries.append(seq.model_copy(update={"tests": defs}))

        if len(retries) == 0:
            return

        retry_total = sum(len(seq.tests) for seq in retries)
        self._log(f"==== RETRYING {retry_total} FAILED TEST(S) ====\n")

        with self._tests_lock:
            for result in failed:
                if str(result) in retry_keys:
                    self._tests.remove(result)
            self._tests_offset = len(self._tests)

        self._total_tests = retry_total
        self._run_sequences(retries)

        with self._tests_lock:
            for test in self._tests:
                if str(test) in retry_keys and test.status == STATUS.PASSED:
                    test.status = STATUS.FLAKY
                    self._on_status_change(test)

    # ----- Sequential execution -----

    def _execute_sequential(self, tests: list[TestDefinition], core: Core) -> None:
        for test_def in tests:
            if self.should_stop:
                self._log("Test execution stopped by user request")
                break

            global_idx = len(self._tests) - self._tests_offset + 1
            self._log(f"[{global_idx}/{self._total_tests}] {test_def}")

            future: Future[Test] = Future()
            t = threading.Thread(
                target=self._test_runner_thread,
                args=(test_def, core, future),
            )
            t.start()
            while t.is_alive() and not self._watchdog.abandoned(t):
                t.join(timeout=1.0)
            if t.is_alive():
                self._abandon(t, test_def)

            self._collect_result(test_def, future)

    # ----- Async execution -----

    def _execute_async(
        self,
        seq_name: str,
        tests: list[TestDefinition],
        pool_size: int,
        core: Core,
    ) -> None:
        use_pool = 0 < pool_size < len(tests)
        slots = Slots(pool_size) if use_pool else None

        if use_pool:
            self._log(
                f"Running tests with pool size of {pool_size} "
                f"(max {pool_size} concurrent tests)..."
            )

        pending: list[tuple[threading.Thread, Future[Test], TestDefinition]] = []

        def _run(td: TestDefinition, ft: Future[Test], idx: int) -> None:
            if slots is not None:
                slots.acquire()
            try:
                if slots is not None:
                    self._log(f"[{idx}/{self._total_tests}] {td}")
                self._test_runner_thread(td, core, ft)
            finally:
                if slots is not None:
                    slots.free(threading.current_thread())

        for i, test_def in enumerate(tests):
            if self.should_stop:
                self._log("Test execution stopped by user request")
                break

            global_idx = len(self._tests) - self._tests_offset + i + 1
            if slots is None:
                self._log(f"[{global_idx}/{self._total_tests}] {test_def}")

            future: Future[Test] = Future()
            t = threading.Thread(
                target=_run,
                args=(test_def, future, global_idx),
            )
            pending.append((t, future, test_def))
            t.start()

        self._log(
            f"Waiting for {len(pending)} tests in sequence '{seq_name}' to complete..."
        )
        while pending:
            still_pending = []
            for t, ft, td in pending:
                if self._watchdog.abandoned(t):
                    if slots is not None:
                        slots.free(t)
                    self._abandon(t, td)
                    self._collect_result(td, ft)
                elif not t.is_alive():
                    self._collect_result(td, ft)
                else:
                    still_pending.append((t, ft, td))
            pending = still_pending
            if pending:
                pending[0][0].join(timeout=1.0)

    # ----- Single test execution -----

    def _abandon(self, thread: threading.Thread, test_def: TestDefinition) -> None:
        self._resources.abandon(thread, str(test_def))
        with self._tests_lock:
            self._abandoned[thread] = str(test_def)

    def _collect_result(self, test_def: TestDefinition, future: Future[Test]) -> None:
        if future.done():
            self._record(test_def, future.result())
            return
        message = f"Test exceeded its timeout ({test_def.timeout}s) and did not stop"
        self._record(test_def, self._result(test_def, STATUS.TIMEOUT, message))

    def _record(self, test_def: TestDefinition, result: Test) -> None:
        with self._tests_lock:
            if any(str(r) == str(test_def) for r in self._tests):
                return
            self._tests.append(result)
            self._on_test_ran()

    @staticmethod
    def _result(test_def: TestDefinition, status: STATUS, message: str) -> Test:
        return Test(
            test_name=test_def.case,
            name=test_def.display_name,
            status=status,
            error_message=message,
        )

    def _test_runner_thread(
        self, test_def: TestDefinition, core: Core, future: Future[Test]
    ) -> None:
        held = self._config_client.resources_of(test_def)
        start = time.monotonic()
        try:
            self._resources.acquire(held)
        except Unavailable as e:
            future.set_result(self._result(test_def, STATUS.FAILED, str(e)))
            return
        waited = time.monotonic() - start
        if waited > _WAIT_LOGGED:
            self._log(f"{test_def} waited {waited:.1f}s for {', '.join(sorted(held))}")
        try:
            if self.should_stop:
                message = "The run stopped before the test started"
                future.set_result(self._result(test_def, STATUS.KILLED, message))
                return
            future.set_result(self._execute_single_test(test_def, core))
        finally:
            self._resources.free(threading.current_thread())

    def _execute_single_test(self, test_def: TestDefinition, core: Core) -> Test:
        self._watchdog.watch(threading.current_thread(), timeout=test_def.timeout)
        test = Test(
            test_name=test_def.case,
            name=test_def.display_name,
            status=STATUS.PENDING,
            core=core.address,
        )

        test_index = len(self._tests)
        color = _COLORS[test_index % len(_COLORS)]

        if self._conductor_range is not None:
            test.range = self._conductor_range.create_child_range(
                name=test.name or test.test_name,
                time_range=sy.TimeRange(start=sy.TimeStamp.now(), end=sy.TimeStamp.MAX),
                color=color,
            )
        else:
            test.range = None

        bundle_dir = self._create_bundle_dir(test)
        log_start_offset = server_log.current_offset(core.log_path)
        test.started_at = time.time()

        test_instance: TestCase | None = None
        try:
            test_class = self._config_client.load_test_class(test_def)
            test_instance = test_class(
                synnax_connection=core.connection,
                name=test_def.display_name,
                options=self._options,
                results_dir=bundle_dir,
                params=test_def.parameters,
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
            if test_instance._error_message is not None:
                test.error_message = test_instance._error_message
            if test_instance._error_traceback is not None:
                test.error_traceback = test_instance._error_traceback
            test.skip_reason = test_instance.skip_reason

        except Timeout:
            test.status = STATUS.TIMEOUT
            test.error_message = f"Test exceeded its timeout ({test_def.timeout}s)"
            self._log(f"{test_def} TIMEOUT after {test_def.timeout}s")

        except Exception as e:
            test.status = STATUS.FAILED
            test.error_message = str(e)
            self._log(f"{test_def.case} FAILED: {e}")
            self._log(f"Traceback: {traceback.format_exc()}")

        finally:
            self._watchdog.disarm(threading.current_thread())
            test.ended_at = time.time()
            self._slice_server_log_if_failed(
                test, bundle_dir, log_start_offset, core.log_path
            )

            if test.range is not None:
                try:
                    test.range = self._finalize_range(test.range)
                except Exception as e:
                    self._log(
                        f"Warning: Could not finalize range '{test.range.name}': {e}"
                    )

            if test_instance is not None and test_instance._status != STATUS.KILLED:
                if test.status in (
                    STATUS.FAILED,
                    STATUS.TIMEOUT,
                ) or test_instance._status in (STATUS.FAILED, STATUS.TIMEOUT):
                    test_instance.log_client.dump()
                else:
                    test_instance.log_client.discard()
                test_instance.log_client.close()

            if test_instance is not None and test.status == STATUS.RUNNING:
                test.status = test_instance._status
            if test_instance is not None:
                test.metrics = test_instance.metrics

            with self._active_tests_lock:
                self._active_tests[:] = [
                    (td, t, tr, th)
                    for td, t, tr, th in self._active_tests
                    if t != test_instance
                ]

            self._on_status_change(test)
            self._watchdog.unwatch(threading.current_thread())

        return test

    def _create_bundle_dir(self, test: Test) -> Path:
        slug = test.name or test.test_name.replace("/", "_")
        bundle = self._run_dir / "tests" / slug
        bundle.mkdir(parents=True, exist_ok=True)
        test.bundle_dir = str(bundle)
        return bundle

    def _slice_server_log_if_failed(
        self,
        test: Test,
        bundle: Path,
        start_offset: int | None,
        log_path: Path,
    ) -> None:
        """Capture the server-log window for this test when it fails.

        Passing tests skip slicing to avoid bloating the bundle with noise.
        Slicing failures are silent. The trace and full server.log are still
        sufficient to debug.
        """
        if start_offset is None:
            return
        if test.status not in (STATUS.FAILED, STATUS.TIMEOUT, STATUS.KILLED):
            return
        end_offset = server_log.current_offset(log_path)
        server_log.slice_to(
            log_path,
            bundle / "server.log",
            start=start_offset,
            end=end_offset,
        )

    def _finalize_range(self, test_range: sy.Range) -> sy.Range:
        self._client.ranges.set_end(test_range.key, sy.TimeStamp.now())
        return self._client.ranges.retrieve(key=test_range.key)

    # ----- Timeout monitoring -----

    def _timeout_monitor_loop(
        self,
        monitor_interval: sy.TimeSpan = 500 * sy.TimeSpan.MILLISECOND,
    ) -> None:
        while self.is_running and not self.should_stop:
            for thread in self._watchdog.check():
                self._log(f"Interrupted timed-out test thread {thread.name}")
            sy.sleep(monitor_interval)

    # ----- Kill / terminate -----

    def kill_active_tests(self) -> int:
        with self._active_tests_lock:
            if not self._active_tests:
                return 0

            killed_results = []
            threads_to_terminate = []
            killed_instances = []

            for test_def, test_instance, test_range, thread in self._active_tests:
                status = STATUS.KILLED
                error_msg = "Test was manually killed"
                if test_range is None:
                    finalized_range = None
                else:
                    try:
                        finalized_range = self._finalize_range(test_range)
                    except Exception as e:
                        self._log(
                            f"Warning: Could not finalize range '{test_range.name}' "
                            f"for killed test: {e}"
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

        for _ in killed_results:
            self._on_test_ran()

        for thread in threads_to_terminate:
            self._terminate_thread(thread)

        for instance in killed_instances:
            self._log(f"--- Logs for {instance.name} ---")
            instance.log_client.dump()
            instance.log_client.close()
            self._log(f"--- End logs for {instance.name} ---")

        return len(killed_results)

    def _terminate_thread(self, thread: threading.Thread) -> None:
        thread.join(timeout=0.1)
        if thread.is_alive() and not interrupt(thread, SystemExit):
            self._log(f"Warning: Could not terminate thread {thread.name}")
