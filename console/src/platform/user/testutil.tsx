// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";
import { fireEvent, render, type RenderResult, screen } from "@testing-library/react";
import { type ReactElement } from "react";

import { Modals } from "@/platform/modals";
import { createConsoleWrapper, renderWithConsole, type TestStore } from "@/testutil";

export interface OpenModalOptions<P> {
  client?: Synnax | null;
  params?: P;
}

/**
 * Mounts a button that opens the given modal-opener hook alongside a live
 * {@link Modals.Stack}, clicks it, and returns the render result plus the console store.
 * Pass a real client to exercise the enabled/save path, or omit it (null) to exercise the
 * no-cluster branch.
 */
export const openModal = async <P,>(
  useOpen: () => Modals.Opener<P>,
  { client = null, params }: OpenModalOptions<P> = {},
): Promise<RenderResult & { store: TestStore }> => {
  const Harness = (): ReactElement => {
    const open = useOpen() as (params?: P) => void;
    return <button onClick={() => open(params)}>open</button>;
  };
  Harness.displayName = "Harness";
  const ui = (
    <>
      <Harness />
      <Modals.Stack />
    </>
  );
  let rendered: RenderResult & { store: TestStore };
  if (client == null) rendered = await renderWithConsole(ui);
  else {
    const { wrapper, store } = await createConsoleWrapper({ client });
    rendered = Object.assign(render(ui, { wrapper }), { store });
  }
  fireEvent.click(screen.getByRole("button", { name: "open" }));
  return rendered;
};
