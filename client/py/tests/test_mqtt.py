#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest
from pydantic import ValidationError

import synnax as sy


@pytest.mark.mqtt
class TestMQTTReadTask:
    """Tests for MQTT read task configuration."""

    def test_parse_plain_entry(self):
        """Should parse a stored config into a plain entry with its defaults."""
        cfg = sy.mqtt.ReadConfig.model_validate(
            {
                "device": "broker-1",
                "entries": [
                    {
                        "key": "e-1",
                        "type": "plain",
                        "topic": "plant/line1",
                        "fields": [
                            {"key": "f-1", "channel": 12, "pointer": "/temperature"}
                        ],
                    }
                ],
            }
        )
        entry = cfg.entries[0]
        assert isinstance(entry, sy.mqtt.PlainReadEntry)
        assert entry.topic == "plant/line1"
        assert entry.qos == "at_most_once"
        assert entry.retained_ignored is False
        assert entry.fields[0].pointer == "/temperature"
        assert cfg.auto_start is False
        assert cfg.data_saving_disabled is False

    def test_parse_sparkplug_entry(self):
        """Should select the Sparkplug variant from the type field."""
        cfg = sy.mqtt.ReadConfig.model_validate(
            {
                "device": "broker-1",
                "entries": [
                    {
                        "type": "sparkplug",
                        "group": "Plant",
                        "edge_node": "Line7",
                        "tag": "Motor/RPM",
                        "channel": 9,
                    }
                ],
            }
        )
        entry = cfg.entries[0]
        assert isinstance(entry, sy.mqtt.SparkplugReadEntry)
        assert entry.edge_node == "Line7"

    def test_reject_unknown_entry_type(self):
        """Should reject an entry whose type is not a known variant."""
        with pytest.raises(ValidationError):
            sy.mqtt.ReadConfig.model_validate(
                {"device": "broker-1", "entries": [{"type": "kafka"}]}
            )

    def test_reject_unknown_qos(self):
        """Should reject a quality of service that MQTT does not define."""
        with pytest.raises(ValidationError):
            sy.mqtt.PlainReadEntry.model_validate({"topic": "a", "qos": "twice"})

    def test_entries_and_fields_get_keys(self):
        """Should give each entry and field a unique key."""
        task = sy.mqtt.ReadTask(
            device="broker-1",
            entries=[
                sy.mqtt.PlainReadEntry(
                    topic="plant/a",
                    fields=[
                        sy.mqtt.ReadField(channel=1, pointer="/a"),
                        sy.mqtt.ReadField(channel=2, pointer="/b"),
                    ],
                ),
                sy.mqtt.PlainReadEntry(topic="plant/b"),
            ],
        )
        entries = task.config.entries
        assert len({e.key for e in entries}) == 2
        assert all(e.key for e in entries)
        assert len({f.key for f in entries[0].fields}) == 2

    def test_payload_type(self):
        """Should create a task of the MQTT read type."""
        task = sy.mqtt.ReadTask(device="broker-1", name="Line 1")
        pld = task.to_payload()
        assert pld.type == "mqtt_read"
        assert pld.name == "Line 1"

    def test_create_and_retrieve_read_task(self, client: sy.Synnax):
        """Should store a read config that the task class parses again."""
        task = sy.mqtt.ReadTask(
            device="some-broker",
            data_saving_disabled=True,
            entries=[
                sy.mqtt.PlainReadEntry(
                    topic="plant/line1",
                    qos="at_least_once",
                    fields=[sy.mqtt.ReadField(channel=1234, pointer="/v")],
                )
            ],
        )
        created = client.tasks.create(
            name="test-mqtt-read-task",
            type="mqtt_read",
            config=task.config.model_dump(exclude_none=True),
        )
        parsed = sy.mqtt.ReadTask(created)
        assert parsed.config.data_saving_disabled is True
        entry = parsed.config.entries[0]
        assert isinstance(entry, sy.mqtt.PlainReadEntry)
        assert entry.qos == "at_least_once"
        assert entry.fields[0].channel == 1234


@pytest.mark.mqtt
class TestMQTTWriteTask:
    """Tests for MQTT write task configuration."""

    def test_plain_target_defaults(self):
        """Should default a target to at-least-once delivery and no retain flag."""
        target = sy.mqtt.PlainWriteTarget(topic="plant/valve/set")
        assert target.type == "plain"
        assert target.qos == "at_least_once"
        assert target.retained is False
        assert target.channel.json_type == "number"
        assert target.fields == []

    def test_parse_static_and_generated_fields(self):
        """Should select the field variant from the type field."""
        target = sy.mqtt.PlainWriteTarget.model_validate(
            {
                "topic": "plant/pump/set",
                "channel": {"channel": 7, "pointer": "/rpm"},
                "fields": [
                    {
                        "type": "static",
                        "pointer": "/unit",
                        "json_type": "string",
                        "value": "rpm",
                    },
                    {"type": "generated", "pointer": "/id", "generator": "uuid"},
                ],
            }
        )
        static, generated = target.fields
        assert isinstance(static, sy.mqtt.StaticWriteField)
        assert static.value == "rpm"
        assert isinstance(generated, sy.mqtt.GeneratedWriteField)
        assert generated.generator == "uuid"

    def test_payload_type(self):
        """Should create a task of the MQTT write type."""
        assert sy.mqtt.WriteTask(device="broker-1").to_payload().type == "mqtt_write"

    def test_create_and_retrieve_write_task(self, client: sy.Synnax):
        """Should store a write config that the task class parses again."""
        task = sy.mqtt.WriteTask(
            device="some-broker",
            targets=[
                sy.mqtt.PlainWriteTarget(
                    topic="plant/valve/set",
                    retained=True,
                    channel=sy.mqtt.ChannelField(
                        channel=4321, pointer="/state", json_type="boolean"
                    ),
                )
            ],
        )
        created = client.tasks.create(
            name="test-mqtt-write-task",
            type="mqtt_write",
            config=task.config.model_dump(exclude_none=True),
        )
        target = sy.mqtt.WriteTask(created).config.targets[0]
        assert isinstance(target, sy.mqtt.PlainWriteTarget)
        assert target.retained is True
        assert target.channel.json_type == "boolean"


@pytest.mark.mqtt
class TestMQTTDevice:
    """Tests for the MQTT broker device."""

    def test_device_defaults(self):
        """Should describe a plain broker on the default port."""
        dev = sy.mqtt.Device(host="broker.local", name="Broker")
        assert dev.make == "mqtt"
        assert dev.model == "MQTT broker"
        assert dev.location == "broker.local"
        assert dev.key != ""
        assert dev.properties == {
            "port": 0,
            "secure": False,
            "verification_skipped": False,
            "ca_file": "",
            "cert_file": "",
            "key_file": "",
            "username": "",
            "password": "",
            "client_id": "",
            "keep_alive": 0,
            "version": 1,
        }

    def test_device_custom_config(self):
        """Should carry the TLS and credential settings in its properties."""
        dev = sy.mqtt.Device(
            host="broker.local",
            port=8884,
            secure=True,
            ca_file="/etc/synnax/ca.crt",
            username="operator",
            password="valve-7",
            client_id="synnax-line-7",
            keep_alive=15,
            rack=42,
        )
        assert dev.rack == 42
        assert dev.properties["port"] == 8884
        assert dev.properties["secure"] is True
        assert dev.properties["ca_file"] == "/etc/synnax/ca.crt"
        assert dev.properties["username"] == "operator"
        assert dev.properties["client_id"] == "synnax-line-7"
        assert dev.properties["keep_alive"] == 15

    @pytest.mark.parametrize("port", [-1, 65536])
    def test_device_rejects_port_out_of_range(self, port: int):
        with pytest.raises(ValueError, match="port must be 0 to 65535"):
            sy.mqtt.Device(host="broker.local", port=port)

    def test_device_rejects_negative_keep_alive(self):
        with pytest.raises(ValueError, match="keep_alive must not be negative"):
            sy.mqtt.Device(host="broker.local", keep_alive=-1)

    def test_device_auto_key_generation(self):
        a = sy.mqtt.Device(host="broker.local")
        b = sy.mqtt.Device(host="broker.local")
        assert a.key != b.key


@pytest.mark.mqtt
class TestMQTTTaskRack:
    """Tests that MQTT tasks run on the rack of their broker device."""

    def test_task_follows_the_rack_of_its_device(self, client: sy.Synnax):
        rack = client.racks.retrieve(integration=sy.mqtt.INTEGRATION)
        dev = client.devices.create(
            sy.mqtt.Device(host="127.0.0.1", rack=rack.key, name="Rack Test Broker")
        )
        ch = client.channels.create(
            name="mqtt_rack_test_cmd", data_type="float64", virtual=True
        )
        task = sy.mqtt.WriteTask(
            device=dev.key,
            name="mqtt-rack-test",
            targets=[
                sy.mqtt.PlainWriteTarget(
                    topic="plant/rack/set",
                    channel=sy.mqtt.ChannelField(channel=ch.key, pointer="/v"),
                )
            ],
        )
        client.tasks.configure(task)
        assert client.tasks.retrieve(key=task.key).rack == rack.key
