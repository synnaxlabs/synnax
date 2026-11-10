#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import sys
import time

from framework.run_with_connection import options_from_env

opts = options_from_env()
print(f"secure={opts.secure}")
print(f"ca_cert={opts.ca_cert}", file=sys.stderr)
time.sleep(60)
