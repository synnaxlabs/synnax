// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, label } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { color as xcolor, id } from "@synnaxlabs/x";
import { render, waitFor } from "@testing-library/react";
import { type FC, Profiler, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Label } from "@/label";
import { createAsyncSynnaxWrapper } from "@/testutil";

const client = createTestClient();

describe("Tags", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  const createLabels = async () =>
    await client.labels.create([
      { name: `first-${id.create()}`, color: { type: "fixed", value: "#FF0000" } },
      { name: `second-${id.create()}`, color: { type: "fixed", value: "#00FF00" } },
    ]);

  it("should color a tag with the named color of its label", async () => {
    const named = await client.colors.create({
      name: `tag-${id.create()}`,
      light: "#123456",
    });
    const lbl = await client.labels.create({
      name: `named-${id.create()}`,
      color: { type: "named", key: named.key },
    });
    const { container, findByText } = render(<Label.Tags keys={[lbl.key]} />, {
      wrapper,
    });
    await findByText(lbl.name);
    await waitFor(() =>
      expect(container.querySelector("svg")?.getAttribute("color")).toEqual(
        xcolor.cssString("#123456"),
      ),
    );
  });

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

  it("should not re-render a tag when an unused named color changes", async () => {
    const used = await client.colors.create({
      name: `used-${id.create()}`,
      light: "#123456",
    });
    const unused = await client.colors.create({
      name: `unused-${id.create()}`,
      light: "#654321",
    });
    const [fixed] = await createLabels();
    const [named, control] = await client.labels.create([
      { name: `named-${id.create()}`, color: { type: "named", key: used.key } },
      { name: `control-${id.create()}`, color: { type: "named", key: unused.key } },
    ]);
    const renders = { fixed: 0, named: 0 };
    const { findByText } = render(
      <>
        <Profiler id="fixed" onRender={() => renders.fixed++}>
          <Label.Tags keys={[fixed.key]} />
        </Profiler>
        <Profiler id="named" onRender={() => renders.named++}>
          <Label.Tags keys={[named.key]} />
        </Profiler>
        <Label.Tags keys={[control.key]} />
      </>,
      { wrapper },
    );
    const dotOf = async (name: string): Promise<string | null | undefined> =>
      (await findByText(name))
        .closest(".lyra-tag")
        ?.querySelector("svg")
        ?.getAttribute("color");
    await waitFor(async () =>
      expect(await dotOf(named.name)).toEqual(xcolor.cssString("#123456")),
    );
    await waitFor(async () =>
      expect(await dotOf(control.name)).toEqual(xcolor.cssString("#654321")),
    );
    await findByText(fixed.name);
    const before = { ...renders };
    await client.colors.dispatch(unused.key, [
      color.set({ path: ["light"], value: xcolor.construct("#abcdef") }),
    ]);
    await waitFor(async () =>
      expect(await dotOf(control.name)).toEqual(xcolor.cssString("#abcdef")),
    );
    expect(renders).toEqual(before);
  });
});
