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

# A late reply to an earlier read of another sub-index of the same object passes as the
# reply to this read. The patch checks the sub-index, as ecx_SDOwrite does, so the read
# fails with SOEM packet error 1 (unexpected frame).
CHECK_SDO_READ_SUBINDEX = (
    """         /* slave response should be CoE, SDO response and the correct index */
         if (((aSDOp->MbxHeader.mbxtype & 0x0f) == ECT_MBXT_COE) &&
             ((etohs(aSDOp->CANOpen) >> 12) == ECT_COES_SDORES) &&
             (aSDOp->Index == SDOp->Index))
""",
    """         /* slave response should be CoE, SDO response and the correct index */
         if (((aSDOp->MbxHeader.mbxtype & 0x0f) == ECT_MBXT_COE) &&
             ((etohs(aSDOp->CANOpen) >> 12) == ECT_COES_SDORES) &&
             (aSDOp->Index == SDOp->Index) &&
             (aSDOp->SubIndex == SDOp->SubIndex))
""",
)

# When the slave refuses an EEPROM read, EEPDAT still holds the word of the read before,
# so the slave gets the wrong identity. The patch returns 0, as for a slave that does not
# answer, and pushes SOEM packet error 30 for the slave.
FAIL_REFUSED_EEPROM_READ = (
    """   if (ecx_eeprom_waitnotbusyFP(context, configadr, &estat, timeout))
   {
      do
      {
         wkc = ecx_FPRD(&context->port, configadr, ECT_REG_EEPDAT, sizeof(edat), &edat, EC_TIMEOUTRET);
""",
    """   if (ecx_eeprom_waitnotbusyFP(context, configadr, &estat, timeout))
   {
      if (estat & EC_ESTAT_NACK)
      {
         ecx_packeterror(context, slave, 0, 0, 30); /* EEPROM read refused */
         return 0;
      }
      do
      {
         wkc = ecx_FPRD(&context->port, configadr, ECT_REG_EEPDAT, sizeof(edat), &edat, EC_TIMEOUTRET);
""",
)

PATCHES = {
    "ec_coe.c": [CHECK_SDO_READ_SUBINDEX],
    "ec_main.c": [FAIL_REFUSED_EEPROM_READ],
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
