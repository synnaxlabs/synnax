#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Prints JSON telemetry for two turbines, one Kafka record per line as "key<TAB>value",
for kafka-console-producer to publish. Each record carries a nanosecond timestamp in
"/time" and readings in "/rpm", "/exhaust_temp", and "/vibration".
"""

import json
import math
import random
import time

UNITS = {"turbine_a": 0.0, "turbine_b": 1.3}
PERIOD = 0.05  # s, so 20 records per turbine per second

start = time.time()
while True:
    now = time.time()
    elapsed = now - start
    for unit, phase in UNITS.items():
        record = {
            "unit": unit,
            "time": int(now * 1e9),
            "rpm": 3600 + 40 * math.sin(elapsed / 3 + phase) + random.gauss(0, 4),
            "exhaust_temp": 540
            + 15 * math.sin(elapsed / 7 + phase)
            + random.gauss(0, 1),
            "vibration": abs(random.gauss(0.2, 0.05)),
        }
        print(f"{unit}\t{json.dumps(record)}", flush=True)
    time.sleep(PERIOD)
