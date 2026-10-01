// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { license, user } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { stubClipboardWriteText } from "@synnaxlabs/lyra/testutil";
import { Access } from "@synnaxlabs/pluto";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { License } from "@/feature/license";
import { findCommand } from "@/platform/command/testutil";
import { License as PlatformLicense } from "@/platform/license";
import { findButton } from "@/platform/modals/testutil";
import {
  assertDefined,
  createTestClientWithGrants,
  renderHookWithConsole,
} from "@/testutil";

describe("License Commands", () => {
  afterEach(() => vi.restoreAllMocks());

  it("should open the license info dialog", async () => {
    const client = createTestClient();
    const { license: lic } = await client.license.retrieve();
    if (lic == null) throw new Error("test Core has no license");
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: License.COMMANDS,
      client,
    });
    await openCommandPalette();
    await selectCommand("Show license info");
    expect(
      await screen.findByText(`${PlatformLicense.editionLabel(lic)} license`),
    ).toBeTruthy();
    expect(screen.getByText(`License ${lic.jti}`)).toBeTruthy();
  });

  it("should copy the license diagnostics", async () => {
    const writeText = stubClipboardWriteText();
    const client = createTestClient();
    const info = await client.license.retrieve();
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: License.COMMANDS,
      client,
    });
    await openCommandPalette();
    await selectCommand("Show license info");
    await screen.findByText(/^License /);
    fireEvent.click(findButton("Copy diagnostics"));
    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    const copied = JSON.parse(writeText.mock.calls[0][0] as string);
    expect(copied.license.fingerprint).toEqual(info.fingerprint);
    expect(copied.license.state).toBe(info.state);
  });

  it("should copy the host fingerprint and confirm with a checkmark", async () => {
    const writeText = stubClipboardWriteText();
    const client = createTestClient();
    const info = await client.license.retrieve();
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: License.COMMANDS,
      client,
    });
    await openCommandPalette();
    await selectCommand("Show license info");
    const button = findButton("Copy fingerprint");
    await waitFor(() => expect(button.getAttribute("aria-disabled")).toBeNull());
    fireEvent.click(button);
    await waitFor(() =>
      expect(writeText).toHaveBeenCalledWith(
        PlatformLicense.joinFingerprint(info.fingerprint),
      ),
    );
    await waitFor(() =>
      expect(button.querySelector(".pluto-icon--check")).not.toBeNull(),
    );
  });

  it("should hide the license info command without a Core", async () => {
    const { openCommandPalette } = await renderPalette({ commands: License.COMMANDS });
    await openCommandPalette();
    expect(screen.queryByText("Show license info")).toBeNull();
  });

  it("should offer the license info command with a retrieve grant", async () => {
    const gate = findCommand(License.COMMANDS, "Show license info").useVisible;
    assertDefined(gate);
    const client = await createTestClientWithGrants(createTestClient(), {
      retrieve: [license.ONTOLOGY_ID],
    });
    const { result } = await renderHookWithConsole(gate, { client });
    await waitFor(() => expect(result.current).toBe(true));
  });

  it("should withhold the license info command without a grant", async () => {
    const gate = findCommand(License.COMMANDS, "Show license info").useVisible;
    assertDefined(gate);
    const client = await createTestClientWithGrants(createTestClient());
    const { result } = await renderHookWithConsole(
      () => ({
        visible: gate(),
        loaded: Access.useRetrieveGranted(user.TYPE_ONTOLOGY_ID),
      }),
      { client },
    );
    await waitFor(() => expect(result.current.loaded).toBe(true));
    expect(result.current.visible).toBe(false);
  });
});
