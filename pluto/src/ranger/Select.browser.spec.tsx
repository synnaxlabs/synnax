// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, waitForStreamLive } from "@synnaxlabs/client/testutil";
import { Haul } from "@synnaxlabs/lyra/haul";
import { id, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { fireEvent } from "@testing-library/react";
import type { FC, PropsWithChildren } from "react";
import { beforeAll, describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-react";

import { Ranger } from "@/ranger";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

describe("Ranger.Select", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    const Synnax = await createAsyncSynnaxWrapper({ client });
    // The Pluto provider mounts the drag layer outside the Synnax provider.
    wrapper = ({ children }) => (
      <Haul.Provider>
        <Synnax>{children}</Synnax>
      </Haul.Provider>
    );
    await waitForStreamLive(client.connection);
  });

  it("should fill the drag preview with the named color of the range", async () => {
    const named = await client.colors.create({
      name: `drag-${id.create()}`,
      light: "#123456",
    });
    const start = TimeStamp.now();
    const rng = await client.ranges.create({
      name: `drag-${id.create()}`,
      timeRange: { start, end: start.add(TimeSpan.hours(1)) },
      color: { type: "named", key: named.key },
    });
    const c = await render(<Ranger.Select value={rng.key} onChange={vi.fn()} />, {
      wrapper,
    });
    const source = c.getByRole("button", { name: new RegExp(rng.name) });
    await expect.element(source).toBeInTheDocument();
    // The named colors can load after the range, so each try starts a new drag.
    await expect
      .poll(() => {
        // A browser builds a drag event with no DataTransfer unless one is given.
        const dataTransfer = new DataTransfer();
        fireEvent.dragStart(source.element(), { dataTransfer });
        const tile = document.querySelector<HTMLElement>(".lyra-haul-preview__tile");
        const background = tile?.style.background;
        fireEvent.dragEnd(source.element(), { dataTransfer });
        return background;
      })
      .toBe("rgb(18, 52, 86)");
  });
});
