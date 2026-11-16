// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label } from "@synnaxlabs/client";
import { createTestClient, waitForStreamLive } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import { type FC, Profiler, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Label } from "@/label";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

describe("Tags", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    await waitForStreamLive(client.connection);
  });

  const createLabels = async () =>
    await client.labels.create([
      { name: `first-${id.create()}`, color: "#FF0000" },
      { name: `second-${id.create()}`, color: "#00FF00" },
    ]);

  it("should show the name of each label and follow a rename", async () => {
    const [first, second] = await createLabels();
    const { findByText } = render(<Label.Tags keys={[first.key, second.key]} />, {
      wrapper,
    });
    expect(await findByText(first.name)).toBeTruthy();
    expect(await findByText(second.name)).toBeTruthy();
    const renamed = `renamed-${id.create()}`;
    await client.labels.dispatch(
      first.key,
      label.set({ path: ["name"], value: renamed }),
    );
    expect(await findByText(renamed)).toBeTruthy();
  });

  it("should re-render only the tags that hold a renamed label", async () => {
    const [held, other] = await createLabels();
    const renders = { held: 0, other: 0 };
    const { findByText } = render(
      <>
        <Profiler id="held" onRender={() => renders.held++}>
          <Label.Tags keys={[held.key]} />
        </Profiler>
        <Profiler id="other" onRender={() => renders.other++}>
          <Label.Tags keys={[other.key]} />
        </Profiler>
      </>,
      { wrapper },
    );
    await findByText(held.name);
    await findByText(other.name);
    const before = { ...renders };
    const renamed = `renamed-${id.create()}`;
    await client.labels.dispatch(
      held.key,
      label.set({ path: ["name"], value: renamed }),
    );
    await findByText(renamed);
    expect(renders.held).toBeGreaterThan(before.held);
    expect(renders.other).toEqual(before.other);
  });
});
