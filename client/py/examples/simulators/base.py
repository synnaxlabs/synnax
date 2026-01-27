#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import argparse
import threading
from abc import ABC, abstractmethod

import synnax as sy


class SimDAQ(ABC):
    """Base class for thread-based hardware simulators."""

    description: str = "Run simulator standalone"
    end_cmd_channel: str | None = None

    def __init__(self, client: sy.Synnax, verbose: bool = False):
        self.client = client
        self.verbose = verbose
        self._running = False
        self._thread: threading.Thread | None = None
        self._end_cmd_thread: threading.Thread | None = None

    def _log(self, message: str) -> None:
        """Print message only when verbose mode is enabled."""
        if self.verbose:
            print(f"[{self.__class__.__name__}] {message}")

    def start(self) -> None:
        """Create channels and start simulation loop in background thread."""
        self._create_channels()
        self._running = True

        if self.end_cmd_channel is not None:
            self._end_cmd_thread = threading.Thread(
                target=self._watch_end_cmd, daemon=True
            )
            self._end_cmd_thread.start()

        self._thread = threading.Thread(target=self._run_loop, daemon=True)
        self._thread.start()

    def _watch_end_cmd(self) -> None:
        """Watch for end command and stop the simulator."""
        with self.client.open_streamer([self.end_cmd_channel]) as streamer:
            while self._running:
                frame = streamer.read(timeout=1)
                if frame is not None:
                    data = frame.get(self.end_cmd_channel)
                    if len(data) > 0:
                        val = data[-1]
                        if hasattr(val, "item"):
                            val = val.item()
                        if val != 0:
                            self._log(f"Received end command on {self.end_cmd_channel}")
                            self._running = False
                            return

    def stop(self, timeout: float = 5.0) -> None:
        """Stop simulation and wait for threads to finish."""
        self._running = False
        if self._end_cmd_thread:
            self._end_cmd_thread.join(timeout=timeout)
        if self._thread:
            self._thread.join(timeout=timeout)

    @abstractmethod
    def _create_channels(self) -> None:
        """Create required Synnax channels."""
        pass

    @abstractmethod
    def _run_loop(self) -> None:
        """Main simulation loop. Check self._running to know when to stop."""
        pass

    @classmethod
    def main(cls) -> None:
        """Standalone execution entry point. Call from if __name__ == '__main__'."""
        parser = argparse.ArgumentParser(description=cls.description)
        parser.add_argument("--host", default="", help="Synnax server host")
        parser.add_argument("--port", type=int, default=0, help="Synnax server port")
        parser.add_argument("--username", default="", help="Synnax username")
        parser.add_argument("--password", default="", help="Synnax password")
        args = parser.parse_args()

        # Use saved credentials if no host provided, otherwise use CLI args
        if args.host:
            client = sy.Synnax(
                host=args.host,
                port=args.port,
                username=args.username,
                password=args.password,
            )
        else:
            client = sy.Synnax()

        print(f"Starting {cls.__name__} (Ctrl+C to stop)...")
        sim = cls(client, verbose=True)
        sim.start()

        try:
            while sim._running:
                sy.sleep(1)
        except KeyboardInterrupt:
            print("\nStopping simulator...")
        finally:
            sim.stop()
            print("Done.")
