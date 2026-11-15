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

# An abort that answers an earlier request for another object passes as the abort of
# this one. The patches check the index and sub-index of an abort, so a stale one fails
# the transfer with SOEM packet error 1 (unexpected frame).
CHECK_SDO_READ_ABORT = (
    """         /* other slave response */
         else
         {
            if ((aSDOp->Command) == ECT_SDO_ABORT) /* SDO abort frame received */
            {
               ecx_SDOerror(context, slave, index, subindex, etohl(aSDOp->ldata[0]));
""",
    """         /* other slave response */
         else
         {
            if (((aSDOp->Command) == ECT_SDO_ABORT) && /* SDO abort frame received */
                (aSDOp->Index == SDOp->Index) && (aSDOp->SubIndex == SDOp->SubIndex))
            {
               ecx_SDOerror(context, slave, index, subindex, etohl(aSDOp->ldata[0]));
""",
)

CHECK_SDO_EXPEDITED_WRITE_ABORT = (
    """               /* all OK */
            }
            /* unexpected response from slave */
            else
            {
               if (aSDOp->Command == ECT_SDO_ABORT) /* SDO abort frame received */
               {
""",
    """               /* all OK */
            }
            /* unexpected response from slave */
            else
            {
               if ((aSDOp->Command == ECT_SDO_ABORT) && /* SDO abort frame received */
                   (aSDOp->Index == SDOp->Index) && (aSDOp->SubIndex == SDOp->SubIndex))
               {
""",
)

CHECK_SDO_NORMAL_WRITE_ABORT = (
    """                  toggle = toggle ^ 0x10; /* toggle bit for segment request */
               }
            }
            /* unexpected response from slave */
            else
            {
               if (aSDOp->Command == ECT_SDO_ABORT) /* SDO abort frame received */
               {
""",
    """                  toggle = toggle ^ 0x10; /* toggle bit for segment request */
               }
            }
            /* unexpected response from slave */
            else
            {
               if ((aSDOp->Command == ECT_SDO_ABORT) && /* SDO abort frame received */
                   (aSDOp->Index == SDOp->Index) && (aSDOp->SubIndex == SDOp->SubIndex))
               {
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

# A late thread can find the receive timer expired before it reads a frame that came in
# time. The patch reads each packet that came in, past frames of other indexes, before
# the wait gives up.
READ_LATE_FRAMES = (
    """   } while (((wkc <= EC_NOFRAME) || (wkc2 <= EC_NOFRAME)) && !osal_timer_is_expired(timer));
""",
    """   } while (((wkc <= EC_NOFRAME) || (wkc2 <= EC_NOFRAME)) && !osal_timer_is_expired(timer));
   if (wkc <= EC_NOFRAME)
   {
      do
         wkc = ecx_inframe(port, idx, 0);
      while (wkc == EC_OTHERFRAME);
   }
   if ((port->redstate != ECT_RED_NONE) && (wkc2 <= EC_NOFRAME))
   {
      do
         wkc2 = ecx_inframe(port, idx, 1);
      while (wkc2 == EC_OTHERFRAME);
   }
""",
)

READ_LATE_SECONDARY_FRAMES = (
    """         } while ((wkc2 <= EC_NOFRAME) && !osal_timer_is_expired(&timer2));
""",
    """         } while ((wkc2 <= EC_NOFRAME) && !osal_timer_is_expired(&timer2));
         if (wkc2 <= EC_NOFRAME)
         {
            do
               wkc2 = ecx_inframe(port, idx, 1);
            while (wkc2 == EC_OTHERFRAME);
         }
""",
)

# The OSAL requests the 1 ms timer period on its first clock read and never ends it,
# and ecx_closenic ends it once per close. The patches make each bus session request it
# in ecx_setupnic and end it in ecx_closenic.
REQUEST_TIMER_PERIOD_PER_SESSION = (
    """      printf("interface %s could not open with pcap\\n", ifname);
      return 0;
   }
""",
    """      printf("interface %s could not open with pcap\\n", ifname);
      return 0;
   }
   if (!secondary)
   {
      timeBeginPeriod(1);
   }
""",
)

END_TIMER_PERIOD_PER_SESSION = (
    """int ecx_closenic(ecx_portt *port)
{
   timeEndPeriod(1);

   if (port->sockhandle != NULL)
   {
""",
    """int ecx_closenic(ecx_portt *port)
{
   if (port->sockhandle != NULL)
   {
      timeEndPeriod(1);
""",
)

OSAL_REQUESTS_NO_TIMER_PERIOD = (
    """   {
      timeBeginPeriod(1);
      QueryPerformanceFrequency(&sysfrequency);
""",
    """   {
      QueryPerformanceFrequency(&sysfrequency);
""",
)

PATCHES = {
    "ec_coe.c": [
        CHECK_SDO_READ_SUBINDEX,
        CHECK_SDO_READ_ABORT,
        CHECK_SDO_EXPEDITED_WRITE_ABORT,
        CHECK_SDO_NORMAL_WRITE_ABORT,
    ],
    "ec_main.c": [FAIL_REFUSED_EEPROM_READ],
    "nicdrv.c": [
        REQUEST_TIMER_PERIOD_PER_SESSION,
        END_TIMER_PERIOD_PER_SESSION,
        READ_LATE_FRAMES,
        READ_LATE_SECONDARY_FRAMES,
    ],
    "osal.c": [OSAL_REQUESTS_NO_TIMER_PERIOD],
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
