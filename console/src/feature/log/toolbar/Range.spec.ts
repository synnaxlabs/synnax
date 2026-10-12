// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { RoleClients } from "@synnaxlabs/client/testutil";
import { TimeSpan } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Range } from "@/feature/log/toolbar/Range";
import { client, renderLog } from "@/feature/log/toolbar/testutil";

const roles = new RoleClients(client);

const trigger = async (): Promise<HTMLElement> =>
  await screen.findByRole("button", { name: /Rolling|Custom/ });

describe("log/toolbar/Range", () => {
  it("should show the rolling 7 day window for a new log", async () => {
    await renderLog(Range);
    expect((await trigger()).textContent).toContain("Rolling 7d");
  });

  it("should set the range through the server", async () => {
    const { key } = await renderLog(Range);
    const button = await trigger();
    await waitFor(() => expect(button.getAttribute("aria-disabled")).not.toBe("true"));
    fireEvent.click(button);
    fireEvent.click(await screen.findByRole("option", { name: /Rolling 1h/ }));
    await waitFor(async () =>
      expect((await client.logs.retrieve(key)).range).toBe("rolling1h"),
    );
  });

  it("should show the custom window input for the custom range", async () => {
    const custom = { variant: "dynamic" as const, span: Number(TimeSpan.minutes(45)) };
    await renderLog(Range, { log: { range: "custom", custom } });
    expect(await screen.findByDisplayValue("45m")).toBeDefined();
  });

  it("should set the custom window through the server", async () => {
    const { key } = await renderLog(Range, { log: { range: "custom" } });
    const input = await screen.findByPlaceholderText("1h 30m");
    fireEvent.change(input, { target: { value: "2h" } });
    fireEvent.blur(input);
    await waitFor(async () =>
      expect((await client.logs.retrieve(key)).custom).toEqual({
        variant: "dynamic",
        span: Number(TimeSpan.hours(2)),
      }),
    );
  });

  it("should freeze the select for a viewer", async () => {
    await renderLog(Range, { as: await roles.get("Viewer") });
    const button = await trigger();
    await waitFor(() => expect(button.getAttribute("aria-disabled")).toBe("true"));
  });
});
