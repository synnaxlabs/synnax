// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render, screen } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import * as Aggregator from "@/components/context-menu/aggregator";

const VERB = "delete";

interface RegistrarProps {
  enabled?: boolean;
  handler: Aggregator.Handler;
}

const Registrar = ({ enabled, handler }: RegistrarProps): null => {
  Aggregator.useRegister(VERB, { enabled, handler });
  return null;
};

const Sink = (): ReactElement | null => {
  const { count, run } = Aggregator.useRun(VERB);
  if (count === 0) return null;
  return (
    <button type="button" data-testid="delete" onClick={() => void run()}>
      Delete ({count})
    </button>
  );
};

describe("ContextMenu.Aggregator", () => {
  it("collapses heterogeneous registrars into a single aggregated item", () => {
    render(
      <Aggregator.Provider>
        <Registrar handler={vi.fn()} />
        <Registrar handler={vi.fn()} />
        <Sink />
      </Aggregator.Provider>,
    );
    const items = screen.getAllByTestId("delete");
    expect(items).toHaveLength(1);
    expect(items[0].textContent).toBe("Delete (2)");
  });

  it("renders the item on first paint without a flush", () => {
    render(
      <Aggregator.Provider>
        <Registrar handler={vi.fn()} />
        <Sink />
      </Aggregator.Provider>,
    );
    expect(screen.queryByTestId("delete")).not.toBeNull();
  });

  it("fans out to every registered handler on activation", async () => {
    const a = vi.fn();
    const b = vi.fn();
    render(
      <Aggregator.Provider>
        <Registrar handler={a} />
        <Registrar handler={b} />
        <Sink />
      </Aggregator.Provider>,
    );
    await act(async () => {
      fireEvent.click(screen.getByTestId("delete"));
    });
    expect(a).toHaveBeenCalledTimes(1);
    expect(b).toHaveBeenCalledTimes(1);
  });

  it("withholds a handler when its registrar is disabled but keeps the item", async () => {
    const a = vi.fn();
    const b = vi.fn();
    const { rerender } = render(
      <Aggregator.Provider>
        <Registrar handler={a} />
        <Registrar handler={b} enabled />
        <Sink />
      </Aggregator.Provider>,
    );
    rerender(
      <Aggregator.Provider>
        <Registrar handler={a} />
        <Registrar handler={b} enabled={false} />
        <Sink />
      </Aggregator.Provider>,
    );
    expect(screen.getByTestId("delete").textContent).toBe("Delete (1)");
    await act(async () => {
      fireEvent.click(screen.getByTestId("delete"));
    });
    expect(a).toHaveBeenCalledTimes(1);
    expect(b).not.toHaveBeenCalled();
  });

  it("hides the aggregated item when no handler is registered", () => {
    render(
      <Aggregator.Provider>
        <Registrar handler={vi.fn()} enabled={false} />
        <Sink />
      </Aggregator.Provider>,
    );
    expect(screen.queryByTestId("delete")).toBeNull();
  });
});
