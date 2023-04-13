#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from logging import Logger as BaseLogger


class Logger:
    base: BaseLogger
    def __init__(self, name: str):


    def debug(self, msg: str, *args, **kwargs):
        self.base.debug(msg, *args, **kwargs)

    def info(self, msg: str, *args, **kwargs):
        self.base.info(msg, *args, **kwargs)

    def warn(self, msg: str, *args, **kwargs):
        self.base.warning(msg, *args, **kwargs)

    def error(self, msg: str, *args, **kwargs):
        self.base.error(msg, *args, **kwargs)

