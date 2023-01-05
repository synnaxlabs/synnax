#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pathlib

import pytest
from synnax.io import IOFactory

factory = IOFactory()

BASE_DIR = pathlib.Path("./tests/testdata/io")
VALID_FILE = BASE_DIR / "valid"


@pytest.mark.parametrize("ext", factory.extensions())
class TestFactory:
    def test_new_reader_valid_file(self, ext):
        r = factory.new_reader(pathlib.Path(f"{VALID_FILE}.{ext}"))
        assert ext in r.extensions()
