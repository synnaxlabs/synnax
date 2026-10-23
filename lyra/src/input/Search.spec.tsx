// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Dialog } from "@/dialog";
import { Input } from "@/input";
import { Triggers } from "@/triggers";

const ESCAPE = { key: "Escape", code: "Escape" };

describe("Search", () => {
  it("should take focus when it mounts", () => {
    const c = render(<Input.Search value="" onChange={vi.fn()} />);
    expect(document.activeElement).toBe(c.getByRole("textbox"));
  });

  it("should close the enclosing dialog on Escape", () => {
    const c = render(
      <Triggers.Provider>
        <Dialog.Frame initialVisible>
          <Dialog.Dialog>
            <Input.Search value="" onChange={vi.fn()} />
          </Dialog.Dialog>
        </Dialog.Frame>
      </Triggers.Provider>,
    );
    fireEvent.keyDown(c.getByRole("textbox"), ESCAPE);
    expect(c.queryByRole("textbox")).toBeNull();
  });

  it("should close only the innermost dialog", () => {
    const c = render(
      <Triggers.Provider>
        <Dialog.Frame initialVisible>
          <Dialog.Dialog>
            <p>Outer</p>
            <Dialog.Frame initialVisible>
              <Dialog.Dialog>
                <Input.Search value="" onChange={vi.fn()} />
              </Dialog.Dialog>
            </Dialog.Frame>
          </Dialog.Dialog>
        </Dialog.Frame>
      </Triggers.Provider>,
    );
    fireEvent.keyDown(c.getByRole("textbox"), ESCAPE);
    expect(c.queryByRole("textbox")).toBeNull();
    expect(c.getByText("Outer")).toBeTruthy();
  });

  it("should close a dialog open inside its dialog first", () => {
    const c = render(
      <Triggers.Provider>
        <Dialog.Frame initialVisible>
          <Dialog.Dialog>
            <Input.Search value="" onChange={vi.fn()} />
            <Dialog.Frame initialVisible>
              <Dialog.Dialog>
                <p>Child</p>
              </Dialog.Dialog>
            </Dialog.Frame>
          </Dialog.Dialog>
        </Dialog.Frame>
      </Triggers.Provider>,
    );
    fireEvent.keyDown(c.getByRole("textbox"), ESCAPE);
    expect(c.queryByText("Child")).toBeNull();
    expect(c.queryByRole("textbox")).not.toBeNull();
  });

  it("should pass the key to onKeyDown", () => {
    const onKeyDown = vi.fn();
    const c = render(
      <Input.Search value="" onChange={vi.fn()} onKeyDown={onKeyDown} />,
    );
    fireEvent.keyDown(c.getByRole("textbox"), ESCAPE);
    expect(onKeyDown).toHaveBeenCalledOnce();
  });
});
