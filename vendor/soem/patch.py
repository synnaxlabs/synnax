#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Writes a patched copy of a SOEM source file.

Usage: patch.py <source> <output>

Each patch replaces text that must occur exactly once in the source, so an upgrade that
changes the text fails the build instead of skipping the patch.
"""

import os
import sys

# A state read whose frame is lost leaves every slave with state 0 (NONE), as if each
# slave left the bus. The patch keeps the state of each slave when the frame is lost. A
# slave that does not answer in a frame that comes back still reads NONE.
KEEP_STATE_ON_LOST_FRAME = (
    """         ecx_FPRD_multi(context, (lslave - fslave) + 1, &(slca[0]), &(sl[0]), EC_TIMEOUTRET3);
         for (slave = fslave; slave <= lslave; slave++)
         {
            configadr = context->slavelist[slave].configadr;
            rval = etohs(sl[slave - fslave].alstatus);
            context->slavelist[slave].ALstatuscode = etohs(sl[slave - fslave].alstatuscode);
""",
    """         wkc = ecx_FPRD_multi(context, (lslave - fslave) + 1, &(slca[0]), &(sl[0]), EC_TIMEOUTRET3);
         for (slave = fslave; slave <= lslave; slave++)
         {
            configadr = context->slavelist[slave].configadr;
            rval = context->slavelist[slave].state;
            if (wkc >= 0)
            {
               rval = etohs(sl[slave - fslave].alstatus);
               context->slavelist[slave].ALstatuscode = etohs(sl[slave - fslave].alstatuscode);
            }
""",
)

PATCHES = {
    "ec_main.c": [KEEP_STATE_ON_LOST_FRAME],
}

source, output = sys.argv[1:]
with open(source) as f:
    text = f.read()
for old, new in PATCHES[os.path.basename(source)]:
    if text.count(old) != 1:
        sys.exit(f"{source}: the patch text occurs {text.count(old)} times, not once")
    text = text.replace(old, new)
with open(output, "w", newline="\n") as f:
    f.write(text)
