// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { license, type Synnax as Client } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, render, screen } from "@testing-library/react";
import { type FC } from "react";
import { describe, expect, it, vi } from "vitest";

import { License } from "@/platform/license";
import { createConsoleWrapper, createTestClientWithGrants } from "@/testutil";

const client = createTestClient();

const renderLicense = async (Component: FC, as: Client = client): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client: as });
  render(<Component />, { wrapper });
};

describe("License", () => {
  describe("Details", () => {
    it("should show the terms of the license the Core reports", async () => {
      const { license: lic } = await client.license.retrieve();
      if (lic == null) throw new Error("test Core has no license");
      await renderLicense(License.Details);
      expect(await screen.findByText(License.editionLabel(lic))).toBeTruthy();
      expect(screen.getByText(lic.organization)).toBeTruthy();
      expect(screen.getByText(License.describeTerm(lic))).toBeTruthy();
      expect(screen.getByText(String(lic.machines))).toBeTruthy();
      expect(screen.getByText(License.describeChannels(lic))).toBeTruthy();
    });

    it("should show why the license could not be read", async () => {
      const failing = createTestClient();
      vi.spyOn(failing.license, "retrieve").mockRejectedValue(
        new Error("license store unavailable"),
      );
      await renderLicense(License.Details, failing);
      expect(await screen.findByText("Failed to read the license")).toBeTruthy();
      expect(screen.getByText("license store unavailable")).toBeTruthy();
    });
  });

  describe("Summary", () => {
    it("should show the edition and term on one line", async () => {
      const { license: lic } = await client.license.retrieve();
      if (lic == null) throw new Error("test Core has no license");
      await renderLicense(License.Summary);
      const term = License.describeTerm(lic).toLowerCase();
      expect(
        await screen.findByText(`${License.editionLabel(lic)} license, ${term}`),
      ).toBeTruthy();
    });
  });

  describe.each([
    ["Details", License.Details],
    ["Summary", License.Summary],
  ])("%s access", (_, Component) => {
    it("should show the license with a retrieve grant on it", async () => {
      const granted = await createTestClientWithGrants(client, {
        retrieve: [license.ONTOLOGY_ID],
      });
      const { license: lic } = await client.license.retrieve();
      if (lic == null) throw new Error("test Core has no license");
      await renderLicense(Component, granted);
      expect(
        await screen.findByText(License.editionLabel(lic), { exact: false }),
      ).toBeTruthy();
    });

    it("should neither read nor show the license without a grant on it", async () => {
      const denied = await createTestClientWithGrants(client);
      const retrieve = vi.spyOn(denied.license, "retrieve");
      const { wrapper } = await createConsoleWrapper({ client: denied });
      const { container } = render(
        <>
          <Component />
          mounted
        </>,
        { wrapper },
      );
      await screen.findByText("mounted");
      await act(async () => {});
      expect(container.textContent).toBe("mounted");
      expect(retrieve).not.toHaveBeenCalled();
    });
  });
});
