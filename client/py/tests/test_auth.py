#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

import synnax as sy


@pytest.mark.auth
class TestAuthentication:
    def test_invalid_credentials(self):
        """
        Should raise an AuthError
        """
        with pytest.raises(sy.AuthError):
            sy.Synnax(
                host="localhost",
                port=9090,
                username="synnax",
                password="wrong",
            )

    def test_no_password_provided(self):
        """
        Should raise a ValidationError
        """
        with pytest.raises(sy.FieldError):
            sy.Synnax(
                host="localhost",
                port=9090,
                username="synnax",
            )
