// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { act } from "react";

import { Session } from "@/session";
import { type TestStore } from "@/testutil";

/** Places a mosaic layout with the given key into the main window of the store. */
export const placeLayout = (
  store: TestStore,
  key: string,
  overrides: Partial<Session.Layout.State> = {},
): void =>
  act(() => {
    store.dispatch(
      Session.Layout.place({
        key,
        type: "test",
        name: key,
        location: "mosaic",
        windowKey: Drift.MAIN_WINDOW,
        window: { title: key },
        ...overrides,
      }),
    );
  });
