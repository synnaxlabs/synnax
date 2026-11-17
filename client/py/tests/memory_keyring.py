#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from keyring.errors import NoKeyringError, PasswordDeleteError


class MemoryKeyring:
    """A keyring that keeps passwords in memory, so tests never touch the keyring of
    the operating system.
    """

    def __init__(self) -> None:
        self.values: dict[tuple[str, str], str] = {}

    def get_password(self, service: str, username: str) -> str | None:
        return self.values.get((service, username))

    def set_password(self, service: str, username: str, password: str) -> None:
        self.values[(service, username)] = password

    def delete_password(self, service: str, username: str) -> None:
        if self.values.pop((service, username), None) is None:
            raise PasswordDeleteError(username)


class NoKeychain:
    """Fails every call as keyring's fail backend does when the session has no OS
    keychain.
    """

    def get_password(self, service: str, username: str) -> str | None:
        raise NoKeyringError("No recommended backend was available.")

    def set_password(self, service: str, username: str, password: str) -> None:
        raise NoKeyringError("No recommended backend was available.")

    def delete_password(self, service: str, username: str) -> None:
        raise NoKeyringError("No recommended backend was available.")
