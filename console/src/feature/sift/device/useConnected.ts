// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Device } from "@synnaxlabs/pluto";

import { MAKE } from "@/feature/sift/device/types";

const QUERY = { where: { make: { eq: MAKE } }, limit: 1 };

/** @returns whether the Core has a Sift connection. False while it loads. */
export const useConnected = (): boolean =>
  (Device.useResultMultiple(QUERY).data?.length ?? 0) > 0;
