#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import uuid4

from synnax import channel, device, label, ranger, secret, task
from synnax.sift.types_gen import StreamConfig
from synnax.status import VARIANT_ERROR
from synnax.telem import CrudeTimeSpan, TimeRange

MAKE = "sift"
MODEL = "sift"
INTEGRATION = "sift"
SCAN_TYPE = "sift_scan"
UPLOAD_COMMAND_TYPE = "upload"


class StreamTask(
    task.StarterStopperMixin, task.JSONConfigMixin[StreamConfig], task.Protocol
):
    """A task that streams the live data of its channels to a Sift asset. After a
    stop, it sends the data written while it was stopped.

    :param device: The key of the Sift device to send to.
    :param name: A human-readable name for the task.
    :param asset: The name of the Sift asset that receives the data.
    :param channels: The channels to send.
    :param labels: The labels that make a range a Sift run. A range with any of them
        becomes a run once Sift has its data. Empty makes no runs.
    :param auto_start: Whether to start the task automatically.
    """

    TYPE = "sift_stream"
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
        device: device.Key = "",
        name: str = "",
        asset: str = "",
        channels: list[channel.Key] | None = None,
        labels: list[label.Key] | None = None,
        auto_start: bool = False,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = StreamConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = StreamConfig(
            device=device,
            asset=asset,
            channels=channels if channels is not None else [],
            labels=labels if labels is not None else [],
            auto_start=auto_start,
        )


class Device(device.Device):
    """A connection to Sift.

    Sift tasks run inside the Core. Create the device on the rack of the Core::

        rack = client.racks.retrieve(embedded=True, integrations=sy.sift.INTEGRATION)[0]
        key = client.secrets.set(name="Sift API key", value="...").key
        dev = sy.sift.Device(
            uri="grpc-api.siftstack.com:443", api_key=key, rack=rack.key, name="Sift"
        )

    :param uri: The address of the Sift gRPC API, as host:port. A local host connects
        without TLS.
    :param api_key: The key of the secret that holds the Sift API key.
    :param name: Human-readable name for the device.
    :param rack: Key of the rack of the Core.
    :param key: Unique key. Auto-generated if empty.
    :param configured: Whether the device has been configured.
    """

    def __init__(
        self,
        *,
        uri: str,
        api_key: secret.Key,
        name: str = "",
        rack: int = 0,
        key: str = "",
        configured: bool = True,
    ):
        super().__init__(
            key=key if key else str(uuid4()),
            location=uri,
            rack=rack,
            name=name,
            make=MAKE,
            model=MODEL,
            configured=configured,
            properties={"uri": uri, "api_key": str(api_key)},
        )


def upload(
    tasks: task.Client,
    dev: device.Device,
    *,
    asset: str,
    channels: list[channel.Key],
    range_key: ranger.Key | None = None,
    time_range: TimeRange | None = None,
    timeout: CrudeTimeSpan = 15,
) -> str:
    """Queues an upload of stored data to a Sift asset. The upload runs in the Core,
    which reports its progress in a status. Deleting the status cancels the upload.

    :param tasks: The task client of the Synnax client.
    :param dev: The Sift device to upload to.
    :param asset: The name of the Sift asset that receives the data.
    :param channels: The channels to upload.
    :param range_key: The key of the range to upload. Its labels become the tags of a
        Sift run.
    :param time_range: A time span to upload in place of a range. It makes no run.
    :param timeout: The maximum time, in seconds, to wait for the Core to queue it.
    :returns: The key of the status of the upload.
    :raises RuntimeError: If the Core does not queue the upload, such as when neither
        or both of range_key and time_range are set.
    :raises TimeoutError: If the Core does not answer within timeout.
    :raises NotFoundError: If the rack of the device has no Sift scan task.
    """
    args: dict[str, object] = {
        "device": dev.key,
        "asset": asset,
        "channels": channels,
    }
    if range_key is not None:
        args["range"] = str(range_key)
    if time_range is not None:
        args["time_range"] = {
            "start": int(time_range.start),
            "end": int(time_range.end),
        }
    scan = tasks.retrieve(type=SCAN_TYPE, rack=dev.rack, one=True)
    status = scan.execute_command_sync(UPLOAD_COMMAND_TYPE, args=args, timeout=timeout)
    if status.variant == VARIANT_ERROR:
        raise RuntimeError(f"{status.message}: {status.description}")
    data = status.details.data if status.details is not None else None
    if data is None:
        raise RuntimeError("The Core queued the upload without naming its status")
    return str(data["status"])
