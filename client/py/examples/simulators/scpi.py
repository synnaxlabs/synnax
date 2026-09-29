#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""A bench power supply that speaks an SCPI-like text protocol over TCP.

Each command is one line ending in a newline. A line may hold several commands
separated by semicolons, and the replies to its queries come back on one line, also
separated by semicolons.

- ``*IDN?`` replies with ``IDN``.
- ``READ?`` replies with ``<count>,<voltage>,<current>``, the values ``reading`` gives.
  The count starts at 1 and grows by one per reply.
- ``STAT?`` replies with ``T=<temperature>,P=<pressure>`` from ``STATUS``.
- ``MEAS:VOLT?`` replies with the output voltage, which equals the voltage setpoint.
- ``VOLT <value>`` and ``CURR <value>`` set the setpoints, and ``VOLT?`` and ``CURR?``
  reply with them.

A muted simulator reads commands and ignores them, so the Driver sees a device that
stops answering.
"""

import asyncio
import multiprocessing
import socket

import synnax as sy
from examples.simulators.bus_sim import BusSim
from synnax import tcp

IDN = "Synnax Labs,SCPI Simulator,0,1.0"
INITIAL_VOLTAGE = 5.0
INITIAL_CURRENT = 1.0
STATUS = (23.5, 101.25)


def reading(count: int) -> tuple[float, float]:
    """Returns the voltage and current of the READ? reply with the given count. Both
    are exact in binary, and the current is zero or negative."""
    return 10 + (count % 8) * 0.25, -(count % 5) * 0.125


class _Instrument:
    def __init__(self) -> None:
        self.voltage = INITIAL_VOLTAGE
        self.current = INITIAL_CURRENT
        self.count = 0

    def execute(self, line: str) -> str | None:
        """Runs every command on the line and returns the joined query replies, or
        None when the line holds no query this instrument answers."""
        replies = [
            reply
            for command in line.split(";")
            if (reply := self._command(command.strip())) is not None
        ]
        return ";".join(replies) if replies else None

    def _command(self, command: str) -> str | None:
        header, _, argument = command.partition(" ")
        header = header.upper()
        if header == "*IDN?":
            return IDN
        if header == "READ?":
            self.count += 1
            voltage, current = reading(self.count)
            return f"{self.count},{voltage!r},{current!r}"
        if header == "STAT?":
            return f"T={STATUS[0]!r},P={STATUS[1]!r}"
        if header in ("MEAS:VOLT?", "VOLT?"):
            return repr(self.voltage)
        if header == "CURR?":
            return repr(self.current)
        if header == "VOLT" and argument:
            self.voltage = float(argument)
        elif header == "CURR" and argument:
            self.current = float(argument)
        return None


class SCPISim(BusSim[str]):
    """SCPI instrument simulator on TCP port 5025. It records each line it receives,
    without the newline."""

    description = "SCPI instrument simulator on TCP port 5025"
    host = "127.0.0.1"
    port = 5025
    device_name = "SCPI Test Instrument"

    def __init__(self, rate: sy.Rate = 50 * sy.Rate.HZ, verbose: bool = False):
        super().__init__(rate=rate, verbose=verbose)
        self._muted = multiprocessing.Event()

    def mute(self) -> None:
        """Stops answering and executing commands until unmute is called."""
        self._muted.set()

    def unmute(self) -> None:
        """Resumes answering and executing commands."""
        self._muted.clear()

    def query(self, command: str, timeout: float = 2) -> str:
        """Sends one query on a new connection and returns the reply line.

        :raises TimeoutError: If no full reply arrives within the timeout.
        """
        with socket.create_connection((self.host, self.port), timeout=timeout) as conn:
            conn.sendall(f"{command}\n".encode())
            with conn.makefile("r", encoding="ascii", newline="\n") as reply:
                line = reply.readline()
        if not line.endswith("\n"):
            raise TimeoutError(f"No reply to {command}")
        return line.rstrip("\r\n")

    async def _run_server(self) -> None:
        instrument = _Instrument()

        async def serve(
            reader: asyncio.StreamReader, writer: asyncio.StreamWriter
        ) -> None:
            try:
                while line := await reader.readline():
                    if self._muted.is_set():
                        continue
                    text = line.decode("ascii").rstrip("\r\n")
                    self._record(text)
                    reply = instrument.execute(text)
                    if reply is not None:
                        writer.write(f"{reply}\n".encode("ascii"))
                        await writer.drain()
            except (ConnectionError, UnicodeDecodeError):
                pass
            finally:
                writer.close()

        server = await asyncio.start_server(serve, self.host, self.port)
        self._ready.set()
        async with server:
            await server.serve_forever()

    @staticmethod
    def create_device(rack_key: int) -> tcp.Device:
        return tcp.Device(
            host=SCPISim.host,
            port=SCPISim.port,
            name=SCPISim.device_name,
            location=f"{SCPISim.host}:{SCPISim.port}",
            rack=rack_key,
        )


if __name__ == "__main__":
    asyncio.run(SCPISim(verbose=True)._run_server())
