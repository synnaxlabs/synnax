// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createProxiedTestClient, FAST_RETRY } from "@synnaxlabs/client/testutil";
import { within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  clickWhenEnabled,
  core,
  createEnumeration,
  createSelectUser,
  openDelete,
  pickReplacement,
  renderEnumerations,
  waitForDialogToClose,
} from "@/feature/enumeration/testutil";

// The severable proxy needs a Node socket, so this spec runs in jsdom.
describe("Enumeration.Settings", () => {
  it("should keep the users and Replace while the replace runs", async () => {
    const [from, to] = await Promise.all([createEnumeration(), createEnumeration()]);
    const user = await createSelectUser(from.key);
    // While the link is severed, the replace retries, so it stays in flight.
    const { client: editor, proxy } = await createProxiedTestClient({
      retry: { ...FAST_RETRY, maxRetries: 200 },
      onRetry: () => {},
    });
    await renderEnumerations(editor);
    const dialog = await openDelete(from.name);
    expect(
      await within(await dialog.findByLabelText("Users")).findByText(user.name),
    ).toBeTruthy();
    await pickReplacement(dialog, to.name);
    await proxy.sever();
    await clickWhenEnabled(dialog, "Replace");
    expect(dialog.getByRole("button", { name: "Replace" })).toBeTruthy();
    expect(within(dialog.getByLabelText("Users")).getByText(user.name)).toBeTruthy();
    expect(dialog.getByText(/Each of these moves/)).toBeTruthy();
    await proxy.restore();
    await waitForDialogToClose();
    await expect
      .poll(async () => (await core.schematics.retrieve(user.key)).nodes[0].config)
      .toMatchObject({ variant: "select", enumeration: to.key });
  });
});
