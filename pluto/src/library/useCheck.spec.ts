// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd, type library, type Synnax as Client } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Form } from "@synnaxlabs/lyra/form";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Library } from "@/library";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

type Check = (client: Client, value: icd.ICD) => Promise<library.Problem[]>;

const checkWithCore: Check = async (c, value) => await c.icds.check(value);

const useChecked = (values: icd.ICD, check: Check) => {
  const form = Form.use({ values, schema: icd.icdZ });
  Library.useCheck({ form, check });
  return form;
};

describe("Library.useCheck", () => {
  let wrapper: FC<PropsWithChildren>;
  let values: icd.ICD;
  let mode: string;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    values = icd.icdZ.parse({
      name: "ICD",
      entries: [
        { kind: "enum", name: "Mode" },
        { kind: "enum", name: "Gear" },
      ],
    });
    mode = `entries.${values.entries[1].key}.name`;
  });

  it("should put each problem on the form path it names", async () => {
    const { result } = renderHook(() => useChecked(values, checkWithCore), {
      wrapper,
    });
    act(() => result.current.set(mode, "Mode"));
    await waitFor(() =>
      expect(result.current.get(mode)?.status).toMatchObject({
        variant: "error",
        message: 'duplicate entry name "Mode"',
      }),
    );
  });

  it("should clear the problems on the next edit", async () => {
    const { result } = renderHook(() => useChecked(values, checkWithCore), {
      wrapper,
    });
    act(() => result.current.set(mode, "Mode"));
    await waitFor(() =>
      expect(result.current.get(mode)?.status.variant).toEqual("error"),
    );
    act(() => result.current.set(mode, "Gear"));
    expect(result.current.get(mode)?.status.variant).not.toEqual("error");
  });

  it("should check once after a burst of edits", async () => {
    const check = vi.fn<Check>(async () => []);
    const { result } = renderHook(() => useChecked(values, check), { wrapper });
    act(() => {
      result.current.set(mode, "A");
      result.current.set(mode, "AB");
      result.current.set(mode, "ABC");
    });
    await waitFor(() => expect(check).toHaveBeenCalledOnce());
    expect(check.mock.calls[0][1].entries[1].name).toEqual("ABC");
  });

  it("should drop a result when the form changed while the check ran", async () => {
    let release: () => void = () => {};
    const gate = new Promise<void>((resolve) => (release = resolve));
    const check = vi
      .fn<Check>(async () => [])
      .mockImplementationOnce(async () => {
        await gate;
        return [{ path: "entries.1.name", message: "out of date" }];
      });
    const { result } = renderHook(() => useChecked(values, check), { wrapper });
    act(() => result.current.set(mode, "A"));
    await waitFor(() => expect(check).toHaveBeenCalledOnce());
    act(() => result.current.set(mode, "B"));
    await act(async () => release());
    await waitFor(() => expect(check).toHaveBeenCalledTimes(2));
    expect(result.current.get(mode)?.status.variant).not.toEqual("error");
  });
});
