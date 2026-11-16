// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/* The functions that vendor/soem/patch.py adds to SOEM. */

#ifndef _ec_patched_
#define _ec_patched_

#include "soem/soem.h"

#ifdef __cplusplus
extern "C" {
#endif

/** The process data sends of a context that are in flight, oldest first. Starts zeroed.
 */
typedef struct {
    /** The number of sends in flight. */
    uint8 sends;
    /** The number of index stack entries of each send. */
    uint8 frames[EC_MAXBUF];
    /** Whether the frame of each index stack entry came back. */
    boolean received[EC_MAXBUF];
} ec_inflightT;

/** Like ecx_send_processdata_group, and keeps the frames in flight after the frames of
 * the earlier sends in inflight.
 * @return The number of frames sent, or 0 when the group has no process data.
 */
int ecx_send_processdata_inflight(
    ecx_contextt *context,
    uint8 group,
    ec_inflightT *inflight
);

/** Waits up to timeout us for each frame of the oldest send in inflight. When all of
 * them came back, copies their inputs, frees them, and writes the AL status that they
 * read for each slave of group with inputs into alstat, at the index of the slave.
 * Otherwise keeps them in flight. A slave that did not answer gets 0, and every slave
 * gets 0 when this returns EC_NOFRAME.
 * @param[out] alstat         AL status by slave index, EC_MAXSLAVE entries.
 * @return Work counter of the process data, or EC_NOFRAME when a frame of the send has
 * not come back or no send is in flight.
 */
int ecx_receive_processdata_inflight(
    ecx_contextt *context,
    uint8 group,
    int timeout,
    ec_inflightT *inflight,
    uint16 *alstat
);

/** Frees the frames of the oldest send in inflight. Does nothing when no send is in
 * flight.
 */
void ecx_drop_processdata_inflight(ecx_contextt *context, ec_inflightT *inflight);

#ifdef __cplusplus
}
#endif

#endif
