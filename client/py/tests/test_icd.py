#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import base64
from uuid import uuid4

import pytest
from pydantic import ValidationError

import synnax as sy
from x import payload
from x.payload import Float, Unsigned
from x.testutil import assert_eventually

DBC = b"""VERSION ""

BU_: ECU

BO_ 256 EngineStatus: 8 ECU
 SG_ rpm : 0|16@1+ (0.25,0) [0|16383.75] "rpm" Vector__XXX
 SG_ temp : 16|8@1- (1,-40) [-168|87] "degC" Vector__XXX
"""


def create_icd(name: str = "ICD") -> sy.ICD:
    return sy.ICD(
        name=name,
        messages=[
            sy.icd.Message(
                name="Status",
                payload=sy.icd.BinaryPayload(
                    identifier=sy.icd.CanIdentifier(id=0x101, extended=False, fd=False),
                    length=8,
                    fields=[
                        sy.icd.BinaryField(name="state", start_bit=0, bit_length=8),
                        sy.icd.BinaryField(
                            name="temperature",
                            start_bit=8,
                            bit_length=16,
                            kind="integer",
                            scale=0.1,
                            units="degC",
                            default=payload.Float(250),
                        ),
                    ],
                ),
            ),
        ],
    )


@pytest.mark.icd
class TestICD:
    def test_create(self, client: sy.Synnax) -> None:
        """Should create an ICD with a CAN message."""
        ic = client.icds.create(create_icd())
        assert ic.name == "ICD"
        assert len(ic.messages) == 1
        msg = ic.messages[0]
        assert msg.name == "Status"
        payload = msg.payload
        assert isinstance(payload, sy.icd.BinaryPayload)
        assert payload.identifier == sy.icd.CanIdentifier(
            id=0x101, extended=False, fd=False
        )
        assert payload.length == 8
        assert len(payload.fields) == 2
        temp = payload.fields[1]
        assert isinstance(temp, sy.icd.BinaryField)
        assert temp.name == "temperature"
        assert temp.start_bit == 8
        assert temp.bit_length == 16
        assert temp.kind == "integer"
        assert temp.scale == 0.1
        assert temp.units == "degC"
        assert temp.default == Float(250)
        keys = [msg.key, *(f.key for f in payload.fields)]
        assert len(set(keys)) == len(keys)

    def test_create_json(self, client: sy.Synnax) -> None:
        """Should create a JSON message with each field type."""
        fields: list[sy.icd.JSONField] = [
            sy.icd.FloatJSONField(name="temp", pointer="/temp", scale=0.1),
            sy.icd.IntegerJSONField(name="delta", pointer="/delta"),
            sy.icd.UnsignedJSONField(name="count", pointer="/count"),
            sy.icd.StringJSONField(name="state", pointer="/state"),
            sy.icd.BooleanJSONField(name="armed", pointer="/armed"),
            sy.icd.TimestampJSONField(
                name="time", pointer="/ts", time_format="unix_ms"
            ),
            sy.icd.JSONJSONField(name="raw", pointer="/raw"),
            sy.icd.UuidJSONField(name="id", pointer="/id"),
            sy.icd.BytesJSONField(name="blob", pointer="/blob"),
        ]
        match = sy.icd.UnsignedJSONMatch(pointer="/seq", value=Unsigned(2**64 - 1))
        ic = client.icds.create(
            sy.ICD(
                name="JSON",
                messages=[
                    sy.icd.Message(
                        name="Reading",
                        payload=sy.icd.JSONPayload(match=match, fields=fields),
                    )
                ],
            )
        )
        assert client.icds.retrieve(key=ic.key) == ic
        payload = ic.messages[0].payload
        assert isinstance(payload, sy.icd.JSONPayload)
        assert payload.match == match
        assert [type(f) for f in payload.fields] == [type(f) for f in fields]
        assert [f.pointer for f in payload.fields] == [f.pointer for f in fields]
        timestamp = payload.fields[5]
        assert isinstance(timestamp, sy.icd.TimestampJSONField)
        assert timestamp.time_format == "unix_ms"

    def test_create_json_enum_values_and_defaults(self, client: sy.Synnax) -> None:
        """Should keep the enum values and defaults of JSON fields."""
        fields: list[sy.icd.JSONField] = [
            sy.icd.FloatJSONField(
                name="level",
                pointer="/level",
                enum_values=[
                    payload.FloatEntry(label="FULL", value=payload.Float(100.5))
                ],
                default=payload.Float(-1.5),
            ),
            sy.icd.IntegerJSONField(
                name="delta",
                pointer="/delta",
                enum_values=[
                    payload.IntegerEntry(label="MIN", value=payload.Integer(-(2**63)))
                ],
                default=payload.Integer(2**63 - 1),
            ),
            sy.icd.UnsignedJSONField(
                name="count",
                pointer="/count",
                enum_values=[
                    payload.UnsignedEntry(
                        label="MAX", value=payload.Unsigned(2**64 - 1)
                    )
                ],
                default=payload.Unsigned(2**64 - 2),
            ),
            sy.icd.StringJSONField(
                name="state",
                pointer="/state",
                enum_values=[payload.StringEntry(label="on", value="ON")],
                default="OFF",
            ),
            sy.icd.BooleanJSONField(
                name="armed",
                pointer="/armed",
                enum_values=[payload.BooleanEntry(label="yes", value=True)],
                default=False,
            ),
            sy.icd.TimestampJSONField(
                name="time",
                pointer="/ts",
                time_format="unix_ns",
                default=sy.icd.ConstantTimestampDefault(value=sy.TimeStamp(2**62 + 1)),
            ),
            sy.icd.TimestampJSONField(
                name="sent",
                pointer="/sent",
                time_format="iso8601",
                default=sy.icd.NowTimestampDefault(),
            ),
            sy.icd.JSONJSONField(
                name="config", pointer="/config", default={"mode": "auto"}
            ),
            sy.icd.UuidJSONField(
                name="device",
                pointer="/device",
                default=sy.icd.ConstantUUIDDefault(value=uuid4()),
            ),
            sy.icd.UuidJSONField(
                name="request",
                pointer="/request",
                default=sy.icd.RandomUUIDDefault(),
            ),
            sy.icd.BytesJSONField(name="raw", pointer="/raw", default="AQI="),
        ]
        ic = client.icds.create(
            name="Defaults",
            messages=[
                sy.icd.Message(
                    name="Reading", payload=sy.icd.JSONPayload(fields=fields)
                )
            ],
        )
        retrieved = client.icds.retrieve(key=ic.key)
        assert retrieved == ic
        stored = retrieved.messages[0].payload
        assert isinstance(stored, sy.icd.JSONPayload)
        assert stored.fields == fields

    def test_create_text(self, client: sy.Synnax) -> None:
        """Should create a message with a text payload."""
        ic = client.icds.create(
            name="Sensor",
            messages=[
                sy.icd.Message(
                    name="Line",
                    payload=sy.icd.TextPayload(
                        prefix="$",
                        fields=[
                            sy.icd.TaggedTextField(name="temp", tag="T="),
                            sy.icd.DelimitedTextField(
                                name="rh", position=2, default=Float(0.5)
                            ),
                        ],
                    ),
                )
            ],
        )
        payload = ic.messages[0].payload
        assert isinstance(payload, sy.icd.TextPayload)
        assert (payload.delimiter, payload.prefix) == (",", "$")
        temp, rh = payload.fields
        assert isinstance(temp, sy.icd.TaggedTextField)
        assert (temp.name, temp.tag) == ("temp", "T=")
        assert isinstance(rh, sy.icd.DelimitedTextField)
        assert (rh.name, rh.position, rh.default) == ("rh", 2, 0.5)

    def test_reject_binary_field_in_text_payload(self) -> None:
        """Should not parse a binary field in a text payload."""
        with pytest.raises(
            ValidationError, match="Input tag 'binary' found using 'encoding'"
        ):
            sy.icd.TextPayload.model_validate(
                {"fields": [{"encoding": "binary", "name": "a"}]}
            )

    def test_reject_text_identifier_in_binary_payload(self) -> None:
        """Should not parse a text identifier in a binary payload."""
        with pytest.raises(
            ValidationError, match="Input tag 'token' found using 'type'"
        ):
            sy.icd.BinaryPayload.model_validate(
                {"identifier": {"type": "token", "prefix": "$"}}
            )

    def test_create_from_kwargs(self, client: sy.Synnax) -> None:
        """Should create an ICD from keyword arguments."""
        ic = client.icds.create(
            name="Kwargs",
            messages=[sy.icd.Message(name="Mode", payload=sy.icd.BinaryPayload())],
        )
        assert ic.name == "Kwargs"
        assert [m.name for m in ic.messages] == ["Mode"]

    def test_create_multiple(self, client: sy.Synnax) -> None:
        """Should create multiple ICDs."""
        icds = client.icds.create([create_icd("A"), create_icd("B")])
        assert [ic.name for ic in icds] == ["A", "B"]
        assert icds[0].key != icds[1].key

    def test_create_existing_key(self, client: sy.Synnax) -> None:
        """Should refuse an ICD whose key exists and keep the stored ICD."""
        ic = client.icds.create(create_icd())
        with pytest.raises(sy.ValidationError, match=f"icd {ic.key} already exists"):
            client.icds.create(ic.model_copy(update={"name": "Replaced"}))
        assert client.icds.retrieve(key=ic.key).name == ic.name

    def test_create_with_problems(self, client: sy.Synnax) -> None:
        """Should store an ICD with problems that a check finds."""
        ic = client.icds.create(
            sy.ICD(
                name="Duplicates",
                messages=[
                    sy.icd.Message(name="Mode", payload=sy.icd.BinaryPayload()),
                    sy.icd.Message(name="Mode", payload=sy.icd.BinaryPayload()),
                ],
            )
        )
        retrieved = client.icds.retrieve(key=ic.key)
        assert [m.name for m in retrieved.messages] == ["Mode", "Mode"]
        assert client.icds.check(retrieved) == [
            sy.icd.Problem(
                path="messages.1.name", message='duplicate message name "Mode"'
            ),
        ]

    def test_check_valid(self, client: sy.Synnax) -> None:
        """Should find no problems in a valid ICD."""
        assert client.icds.check(create_icd()) == []

    def test_check_invalid(self, client: sy.Synnax) -> None:
        """Should return a problem at the path of every bad value."""
        ic = sy.ICD(
            name="Invalid",
            messages=[
                sy.icd.Message(name="Mode", payload=sy.icd.BinaryPayload()),
                sy.icd.Message(
                    name="Mode",
                    payload=sy.icd.BinaryPayload(
                        identifier=sy.icd.CanIdentifier(id=0x101),
                        length=8,
                        fields=[
                            sy.icd.BinaryField(
                                name="rpm", start_bit=0, bit_length=16, scale=0
                            )
                        ],
                    ),
                ),
            ],
        )
        assert client.icds.check(ic) == [
            sy.icd.Problem(
                path="messages.1.name", message='duplicate message name "Mode"'
            ),
            sy.icd.Problem(
                path="messages.1.payload.fields.0.scale",
                message="scale must be finite and non-zero",
            ),
        ]

    def test_check_does_not_save(self, client: sy.Synnax) -> None:
        """Should not store the ICD it checks."""
        ic = create_icd()
        client.icds.check(ic)
        with pytest.raises(sy.NotFoundError):
            client.icds.retrieve(key=ic.key)

    def test_retrieve_by_key(self, client: sy.Synnax) -> None:
        """Should retrieve an ICD by key."""
        ic = client.icds.create(create_icd())
        assert client.icds.retrieve(key=ic.key) == ic

    def test_retrieve_by_keys(self, client: sy.Synnax) -> None:
        """Should retrieve multiple ICDs by key."""
        icds = client.icds.create([create_icd("A"), create_icd("B")])
        retrieved = client.icds.retrieve(keys=[ic.key for ic in icds])
        assert sorted(ic.name for ic in retrieved) == ["A", "B"]

    def test_retrieve_with_search_term(self, client: sy.Synnax) -> None:
        """Should search for ICDs by name."""
        name = f"{uuid4()} ICD"
        ic = client.icds.create(create_icd(name))

        def check() -> None:
            results = client.icds.retrieve(search_term=name)
            assert ic.key in [r.key for r in results]

        assert_eventually(check)

    def test_rename(self, client: sy.Synnax) -> None:
        """Should rename an ICD."""
        ic = client.icds.create(create_icd())
        client.icds.rename(ic.key, "Renamed")
        retrieved = client.icds.retrieve(key=ic.key)
        assert retrieved.name == "Renamed"
        assert retrieved.messages == ic.messages

    def test_delete(self, client: sy.Synnax) -> None:
        """Should delete an ICD."""
        ic = client.icds.create(create_icd())
        client.icds.delete(ic.key)
        with pytest.raises(sy.NotFoundError):
            client.icds.retrieve(key=ic.key)

    def test_delete_multiple(self, client: sy.Synnax) -> None:
        """Should delete multiple ICDs."""
        icds = client.icds.create([create_icd("A"), create_icd("B")])
        keys = [ic.key for ic in icds]
        client.icds.delete(keys)
        assert client.icds.retrieve(keys=keys) == []

    def test_import_dbc(self, client: sy.Synnax) -> None:
        """Should import the messages of a DBC file."""
        ic = client.icds.create(name="Vehicle")
        imported = client.icds.import_file(ic.key, DBC, "dbc")
        assert imported.key == ic.key
        msg = next(m for m in imported.messages if m.name == "EngineStatus")
        assert isinstance(msg.payload, sy.icd.BinaryPayload)
        assert msg.payload.identifier == sy.icd.CanIdentifier(id=0x100)
        rpm = msg.payload.fields[0]
        assert isinstance(rpm, sy.icd.BinaryField)
        assert (rpm.name, rpm.bit_length, rpm.scale, rpm.units) == (
            "rpm",
            16,
            0.25,
            "rpm",
        )

    def test_import_base64_symbols(self, client: sy.Synnax) -> None:
        """Should import a file whose base64 encoding holds a slash."""
        # Four '?' bytes encode to a '/' in every alignment.
        data = DBC + b' SG_ flags : 24|8@1+ (1,0) [0|255] "????" Vector__XXX\n'
        assert "/" in base64.b64encode(data).decode()
        ic = client.icds.create(name="Vehicle")
        imported = client.icds.import_file(ic.key, data, "dbc")
        msg = next(m for m in imported.messages if m.name == "EngineStatus")
        assert [f.name for f in msg.payload.fields] == ["rpm", "temp", "flags"]

    def test_import_keeps_keys(self, client: sy.Synnax) -> None:
        """Should keep message keys when the same file is imported again."""
        ic = client.icds.create(name="Vehicle")
        first = client.icds.import_file(ic.key, DBC, "dbc")
        second = client.icds.import_file(ic.key, DBC, "dbc")
        assert [m.key for m in first.messages] == [m.key for m in second.messages]

    def test_import_invalid(self, client: sy.Synnax) -> None:
        """Should reject a file that does not parse."""
        ic = client.icds.create(name="Broken")
        with pytest.raises(sy.ValidationError):
            client.icds.import_file(ic.key, b"BO_ not a message", "dbc")
