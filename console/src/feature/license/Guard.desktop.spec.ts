// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax } from "@synnaxlabs/pluto";
import { render, screen } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { License } from "@/feature/license";
import { createConsoleWrapper, UNLICENSED_STATUS } from "@/testutil";

describe("License.Guard", () => {
  it("should offer no log out action", async () => {
    const { wrapper: Console } = await createConsoleWrapper({ client: null });
    const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
      <Console>
        <Synnax.TestProvider client={null} status={UNLICENSED_STATUS}>
          {children}
        </Synnax.TestProvider>
      </Console>
    );
    Wrapper.displayName = "GuardWrapper";
    render(
      <License.Guard>
        <span>licensed content</span>
      </License.Guard>,
      { wrapper: Wrapper },
    );
    expect(screen.getByText(UNLICENSED_STATUS.message)).toBeTruthy();
    expect(screen.queryByText("Log out")).toBeNull();
  });
});
