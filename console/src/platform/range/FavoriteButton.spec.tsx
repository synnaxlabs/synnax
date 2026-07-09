// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Range } from "@/platform/range";
import { createTestRange } from "@/platform/range/testutil";
import { Session } from "@/session";
import { createConsoleWrapper } from "@/testutil";

const client = createTestClient();

const createRange = async () => await createTestRange(client);

describe("Range.FavoriteButton", () => {
  it("should not mark the range as favorite when it is not in the slice", async () => {
    const range = await createRange();
    const { wrapper, store } = await createConsoleWrapper({ client });
    render(<Range.FavoriteButton range={range} />, { wrapper });
    await waitFor(() => expect(screen.getByRole("button")).toBeTruthy());
    expect(Session.Range.selectState(store.getState(), range.key)).toBeUndefined();
    expect(screen.getByRole("button").className).not.toContain(
      "favorite-button--favorite",
    );
  });

  it("should add the range to the slice when favorited", async () => {
    const range = await createRange();
    const { wrapper, store } = await createConsoleWrapper({ client });
    render(<Range.FavoriteButton range={range} />, { wrapper });
    await waitFor(() => expect(screen.getByRole("button")).toBeTruthy());
    fireEvent.click(screen.getByRole("button"));
    const state = Session.Range.selectState(store.getState(), range.key);
    expect(state).toBeDefined();
    expect(state?.name).toEqual(range.name);
    expect(state?.persisted).toBe(true);
    expect(state?.variant).toBe("static");
  });

  it("should remove the range from the slice when unfavorited", async () => {
    const range = await createRange();
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: {
        [Session.Range.SLICE_NAME]: {
          version: 0,
          ranges: Session.Range.fromClient(range),
        },
      },
    });
    render(<Range.FavoriteButton range={range} />, { wrapper });
    await waitFor(() => expect(screen.getByRole("button")).toBeTruthy());
    expect(Session.Range.selectState(store.getState(), range.key)).toBeDefined();
    fireEvent.click(screen.getByRole("button"));
    expect(Session.Range.selectState(store.getState(), range.key)).toBeUndefined();
  });
});
