// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/pluto";
import { useEffect, useRef } from "react";

import { Session } from "@/session";

export const UNAVAILABLE_MESSAGE = "This session will not be saved";

/**
 * Tells the user their session will not be saved. The platform refuses the store
 * outright — a cross-origin frame, private browsing on an older engine, an exhausted
 * quota — so the Console runs but forgets everything on reload. Staying quiet would
 * leave them to discover that by losing work.
 */
export const useStoreStatus = (): void => {
  const unavailable = Session.Persist.useSelectStoreUnavailable();
  const addStatus = Status.useAdder();
  const reported = useRef(false);
  useEffect(() => {
    if (!unavailable || reported.current) return;
    reported.current = true;
    addStatus({
      variant: "warning",
      message: UNAVAILABLE_MESSAGE,
      description:
        "Synnax cannot reach browser storage, so your workspace resets when you " +
        "reload. Private browsing and blocked site data are the usual causes.",
    });
  }, [unavailable, addStatus]);
};
