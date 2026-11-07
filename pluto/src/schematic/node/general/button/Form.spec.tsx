// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DataType, schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { deep, id } from "@synnaxlabs/x";
import { fireEvent, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { ButtonForm } from "@/schematic/node/general/button/Form";
import { createConfig } from "@/schematic/node/registry";
import {
  createAsyncSynnaxWrapper,
  createSynnaxWrapper,
  renderForm,
  type RenderFormResult,
} from "@/testutil";

const path = deep.path<schematic.ButtonNodeConfig>();

const client = createTestClient();

describe("ButtonForm", () => {
  let wrapper: FC<PropsWithChildren>;
  let ch: channel.Channel;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    ch = await client.channels.create({
      name: id.create(),
      dataType: DataType.UINT8,
      virtual: true,
    });
  });

  it("should unbind the command channel when it is cleared", async () => {
    const config = createConfig({ variant: "button", commandChannel: ch.key });
    const { findByLabelText, value } = renderForm(
      <ButtonForm tab="control" onTabChange={vi.fn()} />,
      config,
      schematic.buttonNodeConfigZ,
      wrapper,
    );
    fireEvent.click(await findByLabelText(`Clear ${ch.name}`));
    await waitFor(() => expect(value(path.commandChannel)).toBeUndefined());
  });
});

describe("ButtonForm size", () => {
  const renderStyle = (config: schematic.ButtonNodeConfig): RenderFormResult =>
    renderForm(
      <ButtonForm tab="style" onTabChange={vi.fn()} />,
      config,
      schematic.buttonNodeConfigZ,
    );

  const SAVED = schematic.buttonNodeConfigZ.parse({
    variant: "button",
    label: { label: "Saved" },
    size: "large",
    level: "h5",
  });

  it.each([
    ["XS", "tiny", "small"],
    ["S", "small", "h5"],
    ["M", "medium", "h4"],
    ["L", "large", "h3"],
    ["XL", "huge", "h2"],
  ])("should give %s the text level labels use", async (option, size, level) => {
    const start = size === "huge" ? "tiny" : "huge";
    const { getByText, value } = renderStyle({ ...SAVED, size: start });
    fireEvent.click(getByText(option));
    await waitFor(() => {
      expect(value(path.size)).toBe(size);
      expect(value(path.level)).toBe(level);
    });
  });
});

describe("ButtonForm label", () => {
  const renderStyleTab = (): RenderFormResult =>
    renderForm(
      <ButtonForm tab="style" onTabChange={vi.fn()} />,
      createConfig({ variant: "button" }),
      schematic.buttonNodeConfigZ,
      createSynnaxWrapper({ client: null }),
    );

  it("should keep a newline in the label", async () => {
    const { container, value } = renderStyleTab();
    const field = container.querySelector("textarea");
    if (field == null) throw new Error("expected a text area");
    fireEvent.change(field, { target: { value: "LAUNCH\nPUMP VC" } });
    await waitFor(() => expect(value(path.label.label)).toBe("LAUNCH\nPUMP VC"));
  });

  it("should set the wrap width", async () => {
    const { getByText, value } = renderStyleTab();
    const field = getByText("Wrap width").parentElement?.querySelector("input");
    if (field == null) throw new Error("expected a wrap width input");
    fireEvent.change(field, { target: { value: "80" } });
    fireEvent.blur(field);
    await waitFor(() => expect(value(path.label.maxInlineSize)).toBe(80));
  });

  it("should show the default wrap width until a user sets one", () => {
    const { getByText } = renderStyleTab();
    const field = getByText("Wrap width").parentElement?.querySelector("input");
    expect(field?.value).toBe("150");
  });

  it("should stop wrapping when the wrap width is cleared", async () => {
    const { getByText, value } = renderForm(
      <ButtonForm tab="style" onTabChange={vi.fn()} />,
      createConfig({ variant: "button", label: { maxInlineSize: 80 } }),
      schematic.buttonNodeConfigZ,
      createSynnaxWrapper({ client: null }),
    );
    const field = getByText("Wrap width").parentElement?.querySelector("input");
    if (field == null) throw new Error("expected a wrap width input");
    fireEvent.change(field, { target: { value: "" } });
    fireEvent.blur(field);
    await waitFor(() => expect(value(path.label.maxInlineSize)).toBe(0));
    expect(field.value).toBe("");
    expect(field.placeholder).toBe("No wrapping");
  });

  it("should set the alignment", async () => {
    const { container, value } = renderStyleTab();
    const start = container.querySelector("button#start");
    if (start == null) throw new Error("expected a start alignment button");
    fireEvent.click(start);
    await waitFor(() => expect(value(path.label.align)).toBe("start"));
  });
});
