#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from logging import Logger as BaseLogger

from alamos.noop import noop as noopd


class Logger:
    noop: bool = True
    base: BaseLogger

    @noopd
    def debug(self, msg: str, *args, **kwargs):
        """Logs a message at the Debug level"""
        self.base.debug(msg, *args, **kwargs)

    @noopd
    def info(self, msg: str, *args, **kwargs):
        """Logs a message at the Info level"""
        self.base.info(msg, *args, **kwargs)

    @noopd
    def warn(self, msg: str, *args, **kwargs):
        """Logs a message at the Warn level"""
        self.base.warning(msg, *args, **kwargs)

    @noopd
    def error(self, msg: str, *args, **kwargs):
        """Logs a message at the Error level"""
        self.base.error(msg, *args, **kwargs)
