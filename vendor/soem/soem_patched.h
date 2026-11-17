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

/** Like ecx_receive_processdata_group, and writes the AL status that the frames read
 * for each slave of the group with inputs into alstat, at the index of the slave. A
 * slave that did not answer, or whose frame did not come back, gets 0.
 * @param[in]  context        context struct
 * @param[in]  group          group number
 * @param[in]  timeout        Timeout in us.
 * @param[out] alstat         AL status by slave index, EC_MAXSLAVE entries.
 * @return Work counter of the process data.
 */
int ecx_receive_processdata_group_alstat(
    ecx_contextt *context,
    uint8 group,
    int timeout,
    uint16 *alstat
);

#ifdef __cplusplus
}
#endif

#endif
