#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import uuid4

import pytest

import synnax as sy
from x.testutil import assert_eventually

DBC = b"""VERSION ""

BU_: ECU

BO_ 256 EngineStatus: 8 ECU
 SG_ rpm : 0|16@1+ (0.25,0) [0|16383.75] "rpm" Vector__XXX
 SG_ temp : 16|8@1- (1,-40) [-168|87] "degC" Vector__XXX
"""


def create_library(name: str = "Library") -> sy.Library:
    return sy.Library(
        name=name,
        entries=[
            sy.library.EnumEntry(
                name="State",
                values=[
                    sy.library.EnumValue(value=0, name="Off"),
                    sy.library.EnumValue(value=1, name="On"),
                ],
            ),
            sy.library.MessageEntry(
                name="Status",
                identifier=sy.library.CanIdentifier(id=0x101, extended=False, fd=False),
                length=8,
                fields=[
                    sy.library.BinaryField(name="state", start_bit=0, bit_length=8),
                    sy.library.BinaryField(
                        name="temperature",
                        start_bit=8,
                        bit_length=16,
                        signed=True,
                        scale=0.1,
                        units="degC",
                    ),
                ],
            ),
        ],
    )


@pytest.mark.library
class TestLibrary:
    def test_create(self, client: sy.Synnax) -> None:
        """Should create a library with an enum and a CAN message."""
        lib = client.libraries.create(create_library())
        assert lib.name == "Library"
        assert len(lib.entries) == 2
        en, msg = lib.entries
        assert isinstance(en, sy.library.EnumEntry)
        assert en.name == "State"
        assert [(v.value, v.name) for v in en.values] == [(0, "Off"), (1, "On")]
        assert isinstance(msg, sy.library.MessageEntry)
        assert msg.identifier == sy.library.CanIdentifier(
            id=0x101, extended=False, fd=False
        )
        assert msg.format == "binary"
        assert msg.length == 8
        assert len(msg.fields) == 2
        temp = msg.fields[1]
        assert isinstance(temp, sy.library.BinaryField)
        assert temp.name == "temperature"
        assert temp.start_bit == 8
        assert temp.bit_length == 16
        assert temp.signed
        assert temp.scale == 0.1
        assert temp.units == "degC"
        keys = [en.key, msg.key, *(f.key for f in msg.fields)]
        assert len(set(keys)) == len(keys)

    def test_create_from_kwargs(self, client: sy.Synnax) -> None:
        """Should create a library from keyword arguments."""
        lib = client.libraries.create(
            name="Kwargs", entries=[sy.library.EnumEntry(name="Mode")]
        )
        assert lib.name == "Kwargs"
        assert [e.name for e in lib.entries] == ["Mode"]

    def test_create_multiple(self, client: sy.Synnax) -> None:
        """Should create multiple libraries."""
        libs = client.libraries.create([create_library("A"), create_library("B")])
        assert [lib.name for lib in libs] == ["A", "B"]
        assert libs[0].key != libs[1].key

    def test_create_replace(self, client: sy.Synnax) -> None:
        """Should replace a library with an existing key."""
        lib = client.libraries.create(create_library())
        client.libraries.create(
            sy.Library(key=lib.key, name=lib.name, entries=[lib.entries[0]])
        )
        retrieved = client.libraries.retrieve(key=lib.key)
        assert len(retrieved.entries) == 1
        assert retrieved.entries[0].key == lib.entries[0].key

    def test_create_duplicate_entry_names(self, client: sy.Synnax) -> None:
        """Should reject entries that share a name."""
        lib = sy.Library(
            name="Duplicates",
            entries=[
                sy.library.EnumEntry(name="Mode"),
                sy.library.EnumEntry(name="Mode"),
            ],
        )
        with pytest.raises(sy.PathError) as exc_info:
            client.libraries.create(lib)
        assert exc_info.value.path == ["entries", "1", "name"]
        assert (
            str(exc_info.value)
            == 'entries.1.name: duplicate entry name "Mode": validation error'
        )

    def test_retrieve_by_key(self, client: sy.Synnax) -> None:
        """Should retrieve a library by key."""
        lib = client.libraries.create(create_library())
        assert client.libraries.retrieve(key=lib.key) == lib

    def test_retrieve_by_keys(self, client: sy.Synnax) -> None:
        """Should retrieve multiple libraries by key."""
        libs = client.libraries.create([create_library("A"), create_library("B")])
        retrieved = client.libraries.retrieve(keys=[lib.key for lib in libs])
        assert sorted(lib.name for lib in retrieved) == ["A", "B"]

    def test_retrieve_with_search_term(self, client: sy.Synnax) -> None:
        """Should search for libraries by name."""
        name = f"{uuid4()} Library"
        lib = client.libraries.create(create_library(name))

        def check() -> None:
            results = client.libraries.retrieve(search_term=name)
            assert lib.key in [r.key for r in results]

        assert_eventually(check)

    def test_rename(self, client: sy.Synnax) -> None:
        """Should rename a library."""
        lib = client.libraries.create(create_library())
        client.libraries.rename(lib.key, "Renamed")
        retrieved = client.libraries.retrieve(key=lib.key)
        assert retrieved.name == "Renamed"
        assert retrieved.entries == lib.entries

    def test_delete(self, client: sy.Synnax) -> None:
        """Should delete a library."""
        lib = client.libraries.create(create_library())
        client.libraries.delete(lib.key)
        with pytest.raises(sy.NotFoundError):
            client.libraries.retrieve(key=lib.key)

    def test_delete_multiple(self, client: sy.Synnax) -> None:
        """Should delete multiple libraries."""
        libs = client.libraries.create([create_library("A"), create_library("B")])
        keys = [lib.key for lib in libs]
        client.libraries.delete(keys)
        with pytest.raises(sy.NotFoundError):
            client.libraries.retrieve(keys=keys)

    def test_import_dbc(self, client: sy.Synnax) -> None:
        """Should import the messages and value tables of a DBC file."""
        lib = client.libraries.create(name="Vehicle")
        imported = client.libraries.import_icd(lib.key, DBC, "dbc")
        assert imported.key == lib.key
        msg = next(e for e in imported.entries if e.name == "EngineStatus")
        assert isinstance(msg, sy.library.MessageEntry)
        assert msg.identifier == sy.library.CanIdentifier(id=0x100)
        rpm = msg.fields[0]
        assert isinstance(rpm, sy.library.BinaryField)
        assert (rpm.name, rpm.bit_length, rpm.scale, rpm.units) == (
            "rpm",
            16,
            0.25,
            "rpm",
        )

    def test_import_keeps_keys(self, client: sy.Synnax) -> None:
        """Should keep entry keys when the same file is imported again."""
        lib = client.libraries.create(name="Vehicle")
        first = client.libraries.import_icd(lib.key, DBC, "dbc")
        second = client.libraries.import_icd(lib.key, DBC, "dbc")
        assert [e.key for e in first.entries] == [e.key for e in second.entries]

    def test_import_invalid(self, client: sy.Synnax) -> None:
        """Should reject a file that does not parse."""
        lib = client.libraries.create(name="Broken")
        with pytest.raises(sy.ValidationError):
            client.libraries.import_icd(lib.key, b"BO_ not a message", "dbc")
