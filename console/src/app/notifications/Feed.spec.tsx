// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/pluto";
import { fireEvent, screen } from "@testing-library/react";
import { type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { Feed } from "@/app/notifications/Feed";
import { renderWithConsole } from "@/testutil";

const Harness = ({ crude }: { crude: status.Crude }): ReactElement => {
  const add = Status.useAdder();
  return (
    <>
      <button onClick={() => add(crude)}>add</button>
      <Feed />
    </>
  );
};
Harness.displayName = "Harness";

const addStatus = async (crude: status.Crude): Promise<void> => {
  await renderWithConsole(<Harness crude={crude} />);
  fireEvent.click(screen.getByText("add"));
};

describe("app notifications", () => {
  beforeEach(() => {
    const root = document.createElement("div");
    root.id = "root";
    document.body.appendChild(root);
  });
  afterEach(() => document.getElementById("root")?.remove());

  it("shows a hardware error notification", async () => {
    await addStatus({ key: "rack-1", variant: "error", message: "Rack failed" });
    expect(screen.getByText("Rack failed")).toBeTruthy();
  });
});
