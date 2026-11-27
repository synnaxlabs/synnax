// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep } from "@synnaxlabs/x";
import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Form } from "@/form";
import { createLocalStore } from "@/form/testutil";

const schema = z.object({
  name: z
    .string()
    .min(1, "Name is required")
    .refine((name) => name !== "old", {
      message: "Name is old",
      params: { variant: "warning" },
    }),
  config: z.object({ rate: z.number().positive("Rate must be positive") }),
});

type Values = z.infer<typeof schema>;

const path = deep.path<Values>();

const renderForm = (values: Values) => {
  const { result } = renderHook(() =>
    Form.use({ store: createLocalStore(values), schema }),
  );
  return result.current;
};

describe("focusFirstError", () => {
  it("should ask the handler of the first error to take focus", async () => {
    const form = renderForm({ name: "", config: { rate: -1 } });
    const name = vi.fn();
    const config = vi.fn();
    act(() => {
      form.addFocusHandler(path.name, name);
      form.addFocusHandler(path.config, config);
    });
    await act(async () => {
      await form.validateAsync();
    });
    Form.focusFirstError(form);
    expect(name).toHaveBeenCalledWith("name");
    expect(config).not.toHaveBeenCalled();
  });

  it("should throw the first error when nothing takes focus", async () => {
    const form = renderForm({ name: "task", config: { rate: -1 } });
    await act(async () => {
      await form.validateAsync();
    });
    expect(() => Form.focusFirstError(form)).toThrow("Rate must be positive");
  });

  it("should do nothing when the form has only a warning", async () => {
    const form = renderForm({ name: "old", config: { rate: 1 } });
    await act(async () => {
      await form.validateAsync();
    });
    expect(form.getStatuses()).toMatchObject([{ variant: "warning" }]);
    expect(() => Form.focusFirstError(form)).not.toThrow();
  });
});
