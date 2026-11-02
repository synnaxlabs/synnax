// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Triggers } from "@synnaxlabs/lyra/triggers";
import { fireEvent, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { client, project } from "@/feature/lineplot/testutil";
import { useTriggerHold } from "@/feature/lineplot/useTriggerHold";
import { Session } from "@/session";
import { createConsoleWrapper, uniqueName } from "@/testutil";

const renderTriggerHold = async (enabled: boolean) => {
  const plot = await client.lineplots.create(await project(), {
    name: uniqueName("plot"),
  });
  const { wrapper: Console } = await createConsoleWrapper({ client });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Triggers.Provider>{children}</Triggers.Provider>
    </Console>
  );
  const { result } = renderHook(
    () => {
      useTriggerHold({ key: plot.key, enabled: () => enabled });
      return Session.LinePlot.useSelectControlState({ key: plot.key }).hold;
    },
    { wrapper: Wrapper },
  );
  return result;
};

const pressHold = () => {
  fireEvent.keyDown(window, { code: "KeyH", key: "h" });
  fireEvent.keyUp(window, { code: "KeyH", key: "h" });
};

describe("lineplot useTriggerHold", () => {
  it("should toggle the plot's hold when enabled", async () => {
    const hold = await renderTriggerHold(true);
    expect(hold.current).toBe(false);
    pressHold();
    await waitFor(() => expect(hold.current).toBe(true));
  });

  it("should leave the plot's hold alone when not enabled", async () => {
    const hold = await renderTriggerHold(false);
    pressHold();
    await expect.poll(() => hold.current).toBe(false);
  });
});
