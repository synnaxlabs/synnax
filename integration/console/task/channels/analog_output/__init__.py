#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Analog output channel types for NI tasks."""

from console.task.channels.analog_output.current import Current
from console.task.channels.analog_output.voltage import Voltage

__all__ = ["Current", "Voltage"]
