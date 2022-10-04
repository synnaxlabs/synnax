import pytest
from synnax import Synnax, AuthError
from synnax.exceptions import ValidationError


class TestAuthentication:
    def test_invalid_credentials(self):
        """
        Should raise an AuthError
        """
        with pytest.raises(AuthError):
            Synnax(
                host="localhost",
                port=8080,
                username="synnax",
                password="wrong",
            )

    def test_no_password_provided(self):
        """
        Should raise a ValidationError
        """
        with pytest.raises(ValidationError):
            Synnax(
                host="localhost",
                port=8080,
                username="synnax",
            )
