#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from synnax.hardware.ni import AnalogReadTask

client = sy.Synnax()

# Retrieve USB-6289 Analog Read
task = AnalogReadTask(client.hardware.tasks.retrieve(key=281479271677953))
print(task.config.channels)



