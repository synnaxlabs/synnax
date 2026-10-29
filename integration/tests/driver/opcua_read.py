#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import ssl

from examples.opcua import (
    OPCUATLSAuthSim,
    OPCUATLSDriverCertificateSim,
    OPCUATLSSim,
    read_server_certificate,
)

import synnax as sy
from tests.driver.opcua_task import (
    OPCUAReadTaskCase,
    OPCUASecuredReadTaskCase,
    pinned_certificate,
)
from tests.driver.task import create_channel, create_index

PIN_TIMEOUT = 15 * sy.TimeSpan.SECOND


class OPCUAReadFloat(OPCUAReadTaskCase):
    task_name = "OPCUA Read Float"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.opcua.ReadChannel]:
        idx = create_index(client, "opcua_float_index")
        return [
            sy.opcua.ReadChannel(
                channel=create_channel(
                    client,
                    name=f"my_float_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                node_id=f"NS=2;I={8 + i}",
                data_type=sy.DataType.FLOAT32,
            )
            for i in range(2)
        ]


class OPCUAReadBool(OPCUAReadTaskCase):
    task_name = "OPCUA Read Bool"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.opcua.ReadChannel]:
        idx = create_index(client, "opcua_bool_index")
        return [
            sy.opcua.ReadChannel(
                channel=create_channel(
                    client,
                    name=f"my_bool_{i}",
                    data_type=sy.DataType.UINT8,
                    index=idx.key,
                ),
                node_id=f"NS=2;I={13 + i}",
                data_type=sy.DataType.BOOLEAN,
            )
            for i in range(2)
        ]


class OPCUAReadArray(OPCUAReadTaskCase):
    task_name = "OPCUA Read Array"
    array_mode = True

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.opcua.ReadChannel]:
        idx = create_index(client, "opcua_array_index")
        return [
            sy.opcua.ReadChannel(
                channel=create_channel(
                    client,
                    name=f"my_array_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                node_id=f"NS=2;I={2 + i}",
                data_type=sy.DataType.FLOAT32,
            )
            for i in range(2)
        ]


class OPCUAReadTimestamp(OPCUAReadTaskCase):
    """Test reading server timestamps as the task index (is_index)."""

    task_name = "OPCUA Read Timestamp"
    array_mode = True

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.opcua.ReadChannel]:
        idx = create_index(client, "opcua_timestamp_index")
        return [
            sy.opcua.ReadChannel(
                channel=idx.key,
                node_id="NS=2;I=7",
                data_type=sy.DataType.TIMESTAMP,
                is_index=True,
            ),
        ] + [
            sy.opcua.ReadChannel(
                channel=create_channel(
                    client,
                    name=f"opcua_ts_float_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                node_id=f"NS=2;I={2 + i}",
                data_type=sy.DataType.FLOAT32,
            )
            for i in range(2)
        ]

    def test_reconfigure_rate(self) -> None:
        """With server timestamps as index, the array size is controlled by the
        simulator, not the task's sample_rate."""
        self.log("Skipping: test_reconfigure_rate (server-timestamped index)")


class OPCUAReadMixed(OPCUAReadTaskCase):
    task_name = "OPCUA Read Mixed"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.opcua.ReadChannel]:
        idx = create_index(client, "opcua_mixed_index")
        return [
            sy.opcua.ReadChannel(
                channel=create_channel(
                    client,
                    name=f"my_float_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                node_id=f"NS=2;I={8 + i}",
                data_type=sy.DataType.FLOAT32,
            )
            for i in range(2)
        ] + [
            sy.opcua.ReadChannel(
                channel=create_channel(
                    client,
                    name=f"my_bool_{i}",
                    data_type=sy.DataType.UINT8,
                    index=idx.key,
                ),
                node_id=f"NS=2;I={13 + i}",
                data_type=sy.DataType.BOOLEAN,
            )
            for i in range(2)
        ]


class OPCUATLSReadFloat(OPCUASecuredReadTaskCase):
    task_name = "OPCUA TLS Read Float"
    sim_classes = [OPCUATLSSim]

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.opcua.ReadChannel]:
        idx = create_index(client, "opcua_encrypted_float_index")
        return [
            sy.opcua.ReadChannel(
                channel=create_channel(
                    client,
                    name=f"encrypted_my_float_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                node_id=f"NS=2;I={8 + i}",
                data_type=sy.DataType.FLOAT32,
            )
            for i in range(2)
        ]


class OPCUATLSDriverCertificateReadFloat(OPCUASecuredReadTaskCase):
    """Reads over TLS with the certificate the Driver generates for itself, from a
    device that trusts no server certificate yet."""

    task_name = "OPCUA TLS Driver Certificate Read Float"
    sim_classes = [OPCUATLSDriverCertificateSim]

    def run(self) -> None:
        super().run()
        self.test_pins_server_certificate()

    def test_pins_server_certificate(self) -> None:
        """The Driver saves the certificate that the server presented to the device."""
        self.log("Testing: Pin the server certificate")
        timer = sy.Timer()
        while True:
            pinned = pinned_certificate(
                self.client.devices.retrieve(name=self.device_name)
            )
            if pinned is not None:
                break
            if timer.elapsed() > PIN_TIMEOUT:
                self.fail("The Driver did not pin the server certificate")
            sy.sleep(0.5)
        expected = ssl.PEM_cert_to_DER_cert(read_server_certificate())
        assert pinned == expected, "The Driver pinned another server certificate"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.opcua.ReadChannel]:
        idx = create_index(client, "opcua_driver_certificate_float_index")
        return [
            sy.opcua.ReadChannel(
                channel=create_channel(
                    client,
                    name=f"driver_certificate_my_float_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                node_id=f"NS=2;I={8 + i}",
                data_type=sy.DataType.FLOAT32,
            )
            for i in range(2)
        ]


class OPCUATLSAuthReadFloat(OPCUASecuredReadTaskCase):
    task_name = "OPCUA TLS Auth Read Float"
    sim_classes = [OPCUATLSAuthSim]

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.opcua.ReadChannel]:
        idx = create_index(client, "opcua_enc_userpass_float_index")
        return [
            sy.opcua.ReadChannel(
                channel=create_channel(
                    client,
                    name=f"enc_userpass_float_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                node_id=f"NS=2;I={8 + i}",
                data_type=sy.DataType.FLOAT32,
            )
            for i in range(2)
        ]
