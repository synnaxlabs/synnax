// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Form } from "@/form";
import { createLocalStore } from "@/form/store";

const schema = z.object({
  name: z.string(),
  config: z.object({
    rate: z.number(),
    channels: z.object({ key: z.string(), port: z.number() }).array(),
    tags: z.string().array(),
  }),
});

type Values = z.infer<typeof schema>;

const createValues = (): Values => ({
  name: "task",
  config: {
    rate: 10,
    channels: [
      { key: "a", port: 1 },
      { key: "b", port: 2 },
    ],
    tags: ["x"],
  },
});

const renderForm = () => {
  const store = createLocalStore(createValues());
  const set = vi.spyOn(store, "set");
  const onChange = vi.fn();
  store.onChange(onChange);
  const { result } = renderHook(() => Form.use({ store, schema }));
  return { form: result.current, set, onChange };
};

describe("setChanged", () => {
  it("should set only the leaves that differ", () => {
    const { form, set } = renderForm();
    const next = createValues().config;
    next.rate = 20;
    next.channels[1].port = 3;
    act(() => Form.setChanged(form, "config", createValues().config, next));
    expect(set.mock.calls).toEqual([
      ["config.rate", 20],
      ["config.channels.b.port", 3],
    ]);
    expect(form.value().config).toEqual(next);
  });

  it("should keep a form value that changed after the base", () => {
    const { form, set } = renderForm();
    const base = form.value().config;
    act(() => form.set("config.rate", 30));
    set.mockClear();
    const next = createValues().config;
    next.channels[0].port = 4;
    act(() => Form.setChanged(form, "config", base, next));
    expect(set.mock.calls).toEqual([["config.channels.a.port", 4]]);
    expect(form.value().config.rate).toBe(30);
  });

  it("should write nothing for an equal value", () => {
    const { form, set, onChange } = renderForm();
    act(() =>
      Form.setChanged(form, "config", createValues().config, createValues().config),
    );
    expect(set).not.toHaveBeenCalled();
    expect(onChange).not.toHaveBeenCalled();
  });

  it("should set a keyed list whole when its keys change", () => {
    const { form, set } = renderForm();
    const channels = [{ key: "b", port: 2 }];
    act(() =>
      Form.setChanged(
        form,
        "config.channels",
        createValues().config.channels,
        channels,
      ),
    );
    expect(set.mock.calls).toEqual([["config.channels", channels]]);
  });

  it("should set an object whole when its fields change", () => {
    const { form, set } = renderForm();
    const config = { rate: 10, channels: [], tags: ["x"], extra: true };
    act(() => Form.setChanged(form, "config", createValues().config, config));
    expect(set.mock.calls).toEqual([["config", config]]);
  });

  it("should set a list without keys whole", () => {
    const { form, set } = renderForm();
    act(() => Form.setChanged(form, "config.tags", ["x"], ["x", "y"]));
    expect(set.mock.calls).toEqual([["config.tags", ["x", "y"]]]);
  });

  it("should make every write in one change", () => {
    const { form, onChange } = renderForm();
    act(() =>
      Form.setChanged(form, "", createValues(), {
        ...createValues(),
        name: "renamed",
        config: { ...createValues().config, rate: 20 },
      }),
    );
    expect(onChange).toHaveBeenCalledTimes(1);
    expect(form.value().name).toBe("renamed");
    expect(form.value().config.rate).toBe(20);
  });
});
