// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { render, screen } from "@testing-library/react";
import { type FC } from "react";
import { describe, expect, it } from "vitest";

import { License } from "@/platform/license";
import { createConsoleWrapper } from "@/testutil";

const client = createTestClient();

const renderLicense = async (Component: FC): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client });
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
});
