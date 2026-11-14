// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Form } from "@synnaxlabs/lyra/form";
import { type deep } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ICD } from "@/icd";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

const useChecked = ({ key, name, messages }: icd.ICD) => {
  const form = Form.use({ values: { name, messages }, schema: ICD.formSchema });
  ICD.useCheck({ key, form });
  return form;
};

describe("ICD.useCheck", () => {
  let wrapper: FC<PropsWithChildren>;
  let values: icd.ICD;
  let mode: deep.Target<string>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    values = icd.icdZ.parse({
      name: "ICD",
      messages: [
        { name: "Mode", payload: { format: "binary" } },
        { name: "Gear", payload: { format: "binary" } },
      ],
    });
    mode = ICD.formPath.messages[values.messages[1].key].name;
  });
  afterEach(() => vi.restoreAllMocks());

  it("should put each problem on the form path it names", async () => {
    const { result } = renderHook(() => useChecked(values), { wrapper });
    act(() => result.current.set(mode, "Mode"));
    await waitFor(() =>
      expect(result.current.get(mode)?.status).toMatchObject({
        variant: "error",
        message: 'duplicate message name "Mode"',
      }),
    );
  });

  it("should check the ICD under its own key", async () => {
    const check = vi.spyOn(client.icds, "check");
    renderHook(() => useChecked(values), { wrapper });
    await waitFor(() => expect(check).toHaveBeenCalled());
    expect(check.mock.calls[0][0].key).toBe(values.key);
  });

  it("should show the problems the ICD has when it mounts", async () => {
    values.messages[1].name = "Mode";
    const { result } = renderHook(() => useChecked(values), { wrapper });
    await waitFor(() =>
      expect(result.current.get(mode)?.status).toMatchObject({
        variant: "error",
        message: 'duplicate message name "Mode"',
      }),
    );
  });

  it("should clear a problem that an edit on another path fixes", async () => {
    values.messages[1].name = "Mode";
    const { result } = renderHook(() => useChecked(values), { wrapper });
    await waitFor(() =>
      expect(result.current.get(mode)?.status.variant).toEqual("error"),
    );
    act(() =>
      result.current.set(ICD.formPath.messages[values.messages[0].key].name, "Other"),
    );
    await waitFor(() =>
      expect(result.current.get(mode)?.status.variant).not.toEqual("error"),
    );
  });

  it("should clear the problems on the next edit", async () => {
    const { result } = renderHook(() => useChecked(values), { wrapper });
    act(() => result.current.set(mode, "Mode"));
    await waitFor(() =>
      expect(result.current.get(mode)?.status.variant).toEqual("error"),
    );
    act(() => result.current.set(mode, "Gear"));
    expect(result.current.get(mode)?.status.variant).not.toEqual("error");
  });

  it("should check once after a burst of edits", async () => {
    const check = vi.spyOn(client.icds, "check");
    const { result } = renderHook(() => useChecked(values), { wrapper });
    await waitFor(() => expect(check).toHaveBeenCalledOnce());
    act(() => {
      result.current.set(mode, "A");
      result.current.set(mode, "AB");
      result.current.set(mode, "ABC");
    });
    await waitFor(() => expect(check).toHaveBeenCalledTimes(2));
    expect(check.mock.calls[1][0].messages?.[1].name).toEqual("ABC");
  });

  it("should drop a result when the form changed while the check ran", async () => {
    let release: () => void = () => {};
    const gate = new Promise<void>((resolve) => (release = resolve));
    const check = vi.spyOn(client.icds, "check");
    const { result } = renderHook(() => useChecked(values), { wrapper });
    await waitFor(() => expect(check).toHaveBeenCalledOnce());
    check.mockImplementationOnce(async () => {
      await gate;
      return [{ path: "messages.1.name", message: "out of date" }];
    });
    act(() => result.current.set(mode, "A"));
    await waitFor(() => expect(check).toHaveBeenCalledTimes(2));
    act(() => result.current.set(mode, "B"));
    await act(async () => release());
    await waitFor(() => expect(check).toHaveBeenCalledTimes(3));
    expect(result.current.get(mode)?.status.variant).not.toEqual("error");
  });
});
