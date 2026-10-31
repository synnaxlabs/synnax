// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { SyntheticCore } from "@synnaxlabs/client/testutil";
import { box, TimeStamp } from "@synnaxlabs/x";
import { screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Cell } from "@/table/cells";
import { telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/testutil";
import { render } from "@/testutil/render";
import { value } from "@/vis/value/aether";

describe("Cell.Value", () => {
  const closers: Array<() => Promise<void>> = [];
  afterEach(async () => {
    for (const close of closers.splice(0)) await close();
  });

  it("should mark the problem of a failed read", async () => {
    const core = new SyntheticCore({
      rate: 10,
      start: TimeStamp.seconds(100),
      now: TimeStamp.seconds(110),
    });
    const client = telemTest.createClient(core);
    closers.push(async () => await client.feed.close());
    const { providers } = render(
      <table>
        <tbody>
          <tr>
            <Cell.Value
              {...Cell.defaultConfig("value")}
              channel={999}
              cellKey="cell"
              box={box.construct(0, 0, 100, 36)}
              selected={false}
              editable={false}
              onSelect={vi.fn()}
              onChange={vi.fn()}
            />
          </tr>
        </tbody>
      </table>,
      {
        render: true,
        registry: value.REGISTRY,
        telem: { factories: [telem.createFactory(client)] },
      },
    );
    expect(screen.queryByRole("img")).toBeNull();
    await waitFor(() =>
      expect(
        screen.getByRole("img", { name: "Failed to stream channel value" }),
      ).toBeTruthy(),
    );
    expect(providers.status?.state.statuses).toEqual([]);
  });
});
