// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax } from "@synnaxlabs/pluto";
import { useEffect } from "react";

import { watchAccount } from "@/app/analytics/account";
import { watchScreens } from "@/app/analytics/screen";
import { Analytics } from "@/platform/analytics";
import { Session } from "@/session";

/**
 * Follows the account and the focused tab. Every window follows the account, since each
 * carries its own posthog.
 */
export const Watch = (): null => {
  const sink = Analytics.use();
  const store = Session.useStore();
  const client = Synnax.use();
  useEffect(() => watchAccount({ store, sink }), [store, sink]);
  useEffect(() => {
    if (client == null) return;
    return watchScreens({ store, client, screen: sink.screen });
  }, [store, client, sink]);
  return null;
};
