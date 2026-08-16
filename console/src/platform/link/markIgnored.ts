// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// The ignore flag lives in localStorage rather than Redux because a hard reload fires
// before Redux finishes persisting, and the flag must survive that reload so the
// relaunched window does not re-open the link it was opened from.
export const SHOULD_IGNORE_KEY = "shouldIgnoreLink";

// markNextIgnored flags the next deep link to be skipped. Call it before triggering a
// hard reload so the relaunched window ignores the link that triggered the reload.
export const markNextIgnored = (): void =>
  localStorage.setItem(SHOULD_IGNORE_KEY, "true");
