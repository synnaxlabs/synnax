// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { isConnectionError, MissingCredentialsError } from "@/errors";

/**
 * Whether a background read failed because the client cannot reach the Core with a
 * session: the Core is unreachable, or the client logged out. The reconcile after the
 * stream next goes live repairs what the read missed, so the failure is not reported.
 */
export const isOffline = (exc: unknown): boolean =>
  isConnectionError(exc) || MissingCredentialsError.matches(exc);
