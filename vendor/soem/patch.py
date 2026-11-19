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

# A slave below SAFE-OP keeps its share of the working counter and sends zeros. These
# patches read the AL status of each slave with inputs in the process data frames, so a
# status comes back with the inputs it describes, or not at all. Reads that do not fit
# go in frames of their own in the same send.
INCLUDE_PATCHED_DECLARATIONS = (
    """#include "soem/soem.h"
""",
    """#include "soem/soem.h"
#include "soem_patched.h"
""",
)

ADD_ALSTAT_HELPERS = (
    """/** Transmit processdata to slaves.
""",
    """/* The bytes of an FPRD of the AL status of one slave in a frame. */
#define EC_ALSTATSIZE (EC_HEADERSIZE - EC_ELENGTHSIZE + sizeof(uint16) + EC_WKCSIZE)

/** Returns the first slave after slave in group with inputs, or 0 when there is none.
 */
static uint16 ecx_nextalstat(ecx_contextt *context, uint8 group, uint16 slave)
{
   while (++slave <= context->slavecount)
   {
      if ((context->slavelist[slave].group == group) &&
          (context->slavelist[slave].Ibits > 0))
      {
         return slave;
      }
   }
   return 0;
}

/** Adds an FPRD of the AL status of slave *next, and of each slave in group with
 * inputs after it, to the frame at idx while the frame has room. Leaves *next at the
 * first slave left out, or 0.
 */
static void ecx_addalstat(ecx_contextt *context, uint8 group, uint8 idx, uint16 *next)
{
   ecx_portt *port = &context->port;
   uint8 *frame = port->txbuf[idx];
   ec_comt *last = (ec_comt *)&frame[ETH_HEADERSIZE];
   uint16 status = 0;
   uint16 at;

   while (etohs(last->dlength) & EC_DATAGRAMFOLLOWS)
   {
      last = (ec_comt *)((uint8 *)last + EC_HEADERSIZE +
                         (etohs(last->dlength) & 0x07FF) + EC_WKCSIZE - EC_ELENGTHSIZE);
   }
   /* The 4-byte frame check sequence is not in the buffer. */
   while ((*next != 0) &&
          (port->txbuflength[idx] + EC_ALSTATSIZE <= EC_MAXECATFRAME - 4))
   {
      /* ecx_adddatagram flags only the first datagram as followed by another. */
      last->dlength = htoes(etohs(last->dlength) | EC_DATAGRAMFOLLOWS);
      at = (uint16)(port->txbuflength[idx] - EC_ELENGTHSIZE);
      ecx_adddatagram(port, frame, EC_CMD_FPRD, idx, FALSE,
                      context->slavelist[*next].configadr, ECT_REG_ALSTAT,
                      sizeof(status), &status);
      last = (ec_comt *)&frame[at];
      *next = ecx_nextalstat(context, group, *next);
   }
}

/** Writes the AL status of each FPRD of ECT_REG_ALSTAT in the received frame at idx
 * into alstat at the index of its slave, or 0 when the slave did not answer.
 */
static void ecx_readalstat(ecx_contextt *context, uint8 idx, uint16 *alstat)
{
   uint8 *frame = context->port.rxbuf[idx];
   ec_comt *datagram;
   int at = 0;
   int data;
   int length;
   int slave;
   uint16 adp;
   uint16 le_status;
   uint16 le_wkc;

   while (1)
   {
      datagram = (ec_comt *)&frame[at];
      length = etohs(datagram->dlength) & 0x07FF;
      data = at + EC_HEADERSIZE;
      if (data + length + EC_WKCSIZE > EC_BUFSIZE)
      {
         return;
      }
      adp = etohs(datagram->ADP);
      slave = adp - EC_NODEOFFSET;
      if ((datagram->command == EC_CMD_FPRD) &&
          (etohs(datagram->ADO) == ECT_REG_ALSTAT) && (length == sizeof(le_status)) &&
          (slave >= 1) && (slave <= context->slavecount) &&
          (context->slavelist[slave].configadr == adp))
      {
         memcpy(&le_status, &frame[data], sizeof(le_status));
         memcpy(&le_wkc, &frame[data + length], sizeof(le_wkc));
         alstat[slave] = (etohs(le_wkc) == 1) ? etohs(le_status) : 0;
      }
      if ((etohs(datagram->dlength) & EC_DATAGRAMFOLLOWS) == 0)
      {
         return;
      }
      at = data + length + EC_WKCSIZE - EC_ELENGTHSIZE;
   }
}

/** Transmit processdata to slaves.
""",
)

START_ALSTAT_READS = (
    """   uint16 DCO;

   wkc = 0;
""",
    """   uint16 DCO;
   uint16 alstat = ecx_nextalstat(context, group, 0);
   uint16 status = 0;

   wkc = 0;
""",
)

ADD_ALSTAT_TO_LRD = (
    """ecx_setupdatagram(&context->port, &(context->port.txbuf[idx]), EC_CMD_LRD, idx, w1, w2, sublength, data);
               if (first)
               {
                  /* FPRMW in second datagram */
                  DCO = ecx_adddatagram(&context->port, &(context->port.txbuf[idx]), EC_CMD_FRMW, idx, FALSE,
                                        context->slavelist[context->grouplist[group].DCnext].configadr,
                                        ECT_REG_DCSYSTIME, sizeof(int64), &context->DCtime);
                  first = FALSE;
               }
               /* send frame */
""",
    """ecx_setupdatagram(&context->port, &(context->port.txbuf[idx]), EC_CMD_LRD, idx, w1, w2, sublength, data);
               if (first)
               {
                  /* FPRMW in second datagram */
                  DCO = ecx_adddatagram(&context->port, &(context->port.txbuf[idx]), EC_CMD_FRMW, idx, FALSE,
                                        context->slavelist[context->grouplist[group].DCnext].configadr,
                                        ECT_REG_DCSYSTIME, sizeof(int64), &context->DCtime);
                  first = FALSE;
               }
               ecx_addalstat(context, group, idx, &alstat);
               /* send frame */
""",
)

ADD_ALSTAT_TO_LRW = (
    """ecx_setupdatagram(&context->port, &(context->port.txbuf[idx]), EC_CMD_LRW, idx, w1, w2, sublength, data);
            if (first)
            {
               /* FPRMW in second datagram */
               DCO = ecx_adddatagram(&context->port, &(context->port.txbuf[idx]), EC_CMD_FRMW, idx, FALSE,
                                     context->slavelist[context->grouplist[group].DCnext].configadr,
                                     ECT_REG_DCSYSTIME, sizeof(int64), &context->DCtime);
               first = FALSE;
            }
            /* send frame */
""",
    """ecx_setupdatagram(&context->port, &(context->port.txbuf[idx]), EC_CMD_LRW, idx, w1, w2, sublength, data);
            if (first)
            {
               /* FPRMW in second datagram */
               DCO = ecx_adddatagram(&context->port, &(context->port.txbuf[idx]), EC_CMD_FRMW, idx, FALSE,
                                     context->slavelist[context->grouplist[group].DCnext].configadr,
                                     ECT_REG_DCSYSTIME, sizeof(int64), &context->DCtime);
               first = FALSE;
            }
            ecx_addalstat(context, group, idx, &alstat);
            /* send frame */
""",
)

SEND_LEFT_ALSTAT_READS = (
    """   return wkc;
}

/** Receive processdata from slaves.""",
    """   /* Reads that did not fit in the process data frames go in frames of their
      own. */
   while (wkc && alstat)
   {
      idx = ecx_getindex(&context->port);
      ecx_setupdatagram(&context->port, &(context->port.txbuf[idx]), EC_CMD_FPRD, idx,
                        context->slavelist[alstat].configadr, ECT_REG_ALSTAT,
                        sizeof(status), &status);
      alstat = ecx_nextalstat(context, group, alstat);
      ecx_addalstat(context, group, idx, &alstat);
      ecx_outframe_red(&context->port, idx);
      ecx_pushindex(context, idx, NULL, 0, 0);
   }

   return wkc;
}

/** Receive processdata from slaves.""",
)

# The frame gives the working counter of its last datagram, which can now be an AL
# status read.
COUNT_PROCESS_DATA_WKC = (
    """               /* copy input data back to process data buffer */
               memcpy(idxstack->data[pos], &(rxbuf[idx][EC_HEADERSIZE]), idxstack->length[pos]);
               wkc += wkc2;
""",
    """               /* copy input data back to process data buffer */
               memcpy(idxstack->data[pos], &(rxbuf[idx][EC_HEADERSIZE]), idxstack->length[pos]);
               memcpy(&le_wkc, &(rxbuf[idx][EC_HEADERSIZE + idxstack->length[pos]]), EC_WKCSIZE);
               wkc += etohs(le_wkc);
""",
)

# A USB adapter can bring a frame back a cycle late. These functions keep the frames of
# each send in flight until all of them come back, so a late frame is not lost.
ADD_INFLIGHT = (
    """   /* if no frames has arrived */
   if (valid_wkc == 0)
   {
      return EC_NOFRAME;
   }
   return wkc;
}
""",
    """   /* if no frames has arrived */
   if (valid_wkc == 0)
   {
      return EC_NOFRAME;
   }
   return wkc;
}

/** Frees the frames of the oldest send in inflight and takes them off the index stack.
 */
static void ecx_popinflight(ecx_contextt *context, ec_inflightT *inflight)
{
   ec_idxstackT *idxstack = &context->idxstack;
   uint8 frames = inflight->frames[0];
   uint8 left = (uint8)(idxstack->pushed - frames);
   uint8 pos;

   for (pos = 0; pos < frames; pos++)
   {
      ecx_setbufstat(&context->port, idxstack->idx[pos], EC_BUF_EMPTY);
   }
   memmove(idxstack->idx, &idxstack->idx[frames], left * sizeof(idxstack->idx[0]));
   memmove(idxstack->data, &idxstack->data[frames], left * sizeof(idxstack->data[0]));
   memmove(idxstack->length, &idxstack->length[frames],
           left * sizeof(idxstack->length[0]));
   memmove(idxstack->dcoffset, &idxstack->dcoffset[frames],
           left * sizeof(idxstack->dcoffset[0]));
   memmove(idxstack->type, &idxstack->type[frames], left * sizeof(idxstack->type[0]));
   memmove(inflight->received, &inflight->received[frames],
           left * sizeof(inflight->received[0]));
   idxstack->pushed = left;
   idxstack->pulled = 0;
   inflight->sends--;
   memmove(inflight->frames, &inflight->frames[1],
           inflight->sends * sizeof(inflight->frames[0]));
}

int ecx_send_processdata_inflight(ecx_contextt *context, uint8 group,
                                  ec_inflightT *inflight)
{
   uint8 first = context->idxstack.pushed;
   uint8 pos;
   int sent = ecx_send_processdata_group(context, group);

   if (sent > 0)
   {
      for (pos = first; pos < context->idxstack.pushed; pos++)
      {
         inflight->received[pos] = FALSE;
      }
      inflight->frames[inflight->sends] = (uint8)(context->idxstack.pushed - first);
      return inflight->frames[inflight->sends++];
   }
   return sent;
}

int ecx_receive_processdata_inflight(ecx_contextt *context, uint8 group, int timeout,
                                     ec_inflightT *inflight, uint16 *alstat)
{
   ec_idxstackT *idxstack = &context->idxstack;
   ec_bufT *rxbuf = context->port.rxbuf;
   uint16 slave = 0;
   uint16 le_wkc;
   int64 le_DCtime;
   uint8 command;
   uint8 idx;
   uint8 pos;
   int wkc = 0;

   while ((slave = ecx_nextalstat(context, group, slave)) != 0)
   {
      alstat[slave] = 0;
   }
   if (inflight->sends == 0)
   {
      return EC_NOFRAME;
   }
   for (pos = 0; pos < inflight->frames[0]; pos++)
   {
      /* ecx_waitinframe gives a frame once only. */
      if (!inflight->received[pos])
      {
         inflight->received[pos] =
            (ecx_waitinframe(&context->port, idxstack->idx[pos], timeout) > EC_NOFRAME);
      }
      if (!inflight->received[pos])
      {
         return EC_NOFRAME;
      }
   }
   for (pos = 0; pos < inflight->frames[0]; pos++)
   {
      idx = idxstack->idx[pos];
      command = rxbuf[idx][EC_CMDOFFSET];
      if ((command == EC_CMD_LRD) || (command == EC_CMD_LRW) || (command == EC_CMD_LWR))
      {
         if (command != EC_CMD_LWR)
         {
            memcpy(idxstack->data[pos], &(rxbuf[idx][EC_HEADERSIZE]),
                   idxstack->length[pos]);
         }
         memcpy(&le_wkc, &(rxbuf[idx][EC_HEADERSIZE + idxstack->length[pos]]),
                EC_WKCSIZE);
         /* output WKC counts 2 times when using LRW, emulate the same for LWR */
         wkc += etohs(le_wkc) * ((command == EC_CMD_LWR) ? 2 : 1);
         if (idxstack->dcoffset[pos] > 0)
         {
            memcpy(&le_DCtime, &(rxbuf[idx][idxstack->dcoffset[pos]]),
                   sizeof(le_DCtime));
            context->DCtime = etohll(le_DCtime);
         }
      }
      ecx_readalstat(context, idx, alstat);
   }
   ecx_popinflight(context, inflight);
   return wkc;
}

void ecx_drop_processdata_inflight(ecx_contextt *context, ec_inflightT *inflight)
{
   if (inflight->sends > 0)
   {
      ecx_popinflight(context, inflight);
   }
}
""",
)

PATCHES = {
    "ec_coe.c": [
        CHECK_SDO_READ_SUBINDEX,
        CHECK_SDO_READ_ABORT,
        CHECK_SDO_EXPEDITED_WRITE_ABORT,
        CHECK_SDO_NORMAL_WRITE_ABORT,
    ],
    "ec_main.c": [
        FAIL_REFUSED_EEPROM_READ,
        INCLUDE_PATCHED_DECLARATIONS,
        ADD_ALSTAT_HELPERS,
        START_ALSTAT_READS,
        ADD_ALSTAT_TO_LRD,
        ADD_ALSTAT_TO_LRW,
        SEND_LEFT_ALSTAT_READS,
        COUNT_PROCESS_DATA_WKC,
        ADD_INFLIGHT,
    ],
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
