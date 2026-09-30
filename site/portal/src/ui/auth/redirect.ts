// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { HOME } from "@/shell";

/** withTarget carries the redirect target from this page onto another auth page. */
export const withTarget = (path: string, target: string): string =>
  target === HOME ? path : `${path}?redirect_url=${encodeURIComponent(target)}`;
