// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/*
 * macOS OSHW header for SOEM
 * Based on Linux implementation
 */

#ifndef _oshw_
#define _oshw_

#ifdef __cplusplus
extern "C" {
#endif

// clang-format off
// Order matters: soem.h defines types used by nicdrv.h
#include "soem/soem.h"
#include "nicdrv.h"
// clang-format on

uint16 oshw_htons(uint16 hostshort);
uint16 oshw_ntohs(uint16 networkshort);
ec_adaptert *oshw_find_adapters(void);
void oshw_free_adapters(ec_adaptert *adapter);

#ifdef __cplusplus
}
#endif

#endif
