// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, type ontology } from "@synnaxlabs/client";
import { Menu as PMenu } from "@synnaxlabs/pluto";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Device } from "@/platform/device";
import { buildSelection } from "@/platform/ontology/testutil";
import { renderWithConsole } from "@/testutil";

const idFor = (key: string) => device.ontologyID(key);

const renderItem = async (ids: ontology.ID[], onConfigure = vi.fn()) => {
  await renderWithConsole(
    <PMenu.Menu>
      <Device.EditConnectionMenuItem
        selection={buildSelection({ ids })}
        onConfigure={onConfigure}
      />
    </PMenu.Menu>,
  );
  return onConfigure;
};

describe("EditConnectionMenuItem", () => {
  it("should render an edit item for a single selection", async () => {
    await renderItem([idFor("dev-1")]);
    expect(screen.getByText("Edit connection")).toBeTruthy();
  });

  it("should render nothing when nothing is selected", async () => {
    await renderItem([]);
    expect(screen.queryByText("Edit connection")).toBeNull();
  });

  it("should render nothing when more than one resource is selected", async () => {
    await renderItem([idFor("dev-1"), idFor("dev-2")]);
    expect(screen.queryByText("Edit connection")).toBeNull();
  });

  it("should invoke onConfigure with the selected device key when clicked", async () => {
    const onConfigure = await renderItem([idFor("dev-42")]);
    fireEvent.click(screen.getByText("Edit connection"));
    expect(onConfigure).toHaveBeenCalledWith("dev-42");
  });
});
