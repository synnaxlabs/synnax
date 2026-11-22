// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { enumeration } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { fireEvent, render, waitFor } from "@testing-library/react";
import type { FC, PropsWithChildren } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Enumeration } from "@/enumeration";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

describe("Enumeration select", () => {
  let wrapper: FC<PropsWithChildren>;
  let created: enumeration.Enumeration[];
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    const prefix = id.create();
    created = await client.enumerations.create([
      { name: `${prefix} b`, entries: [] },
      { name: `${prefix} a`, entries: [] },
    ]);
  });

  const query = (): enumeration.RetrieveMultipleParams => ({
    where: { key: { in: created.map((e) => e.key) } },
  });

  it("should list the enumerations by name and pick one", async () => {
    const [b, a] = created;
    const onChange = vi.fn();
    const c = render(
      <Enumeration.SelectSingle
        value={undefined}
        onChange={onChange}
        query={query()}
        virtual={false}
      />,
      { wrapper },
    );
    fireEvent.click(c.getByRole("button", { name: "Enumeration" }));
    await waitFor(() =>
      expect(c.getAllByRole("option").map((o) => o.textContent)).toEqual([
        a.name,
        b.name,
      ]),
    );
    fireEvent.click(c.getByText(a.name));
    expect(onChange).toHaveBeenCalledWith(a.key, expect.anything());
  });

  it("should leave out the enumerations that the filter rejects", async () => {
    const [b, a] = created;
    const c = render(
      <Enumeration.SelectSingle
        value={undefined}
        onChange={vi.fn()}
        query={query()}
        filter={(e) => e.key !== a.key}
        virtual={false}
      />,
      { wrapper },
    );
    fireEvent.click(c.getByRole("button", { name: "Enumeration" }));
    expect(await c.findByText(b.name)).toBeTruthy();
    expect(c.queryByText(a.name)).toBeNull();
  });
});
