// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { newTestClient, type Synnax } from "@synnaxlabs/client";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Flux } from "@/flux";
import { Sync } from "@/flux/sync";
import { Synnax as PSynnax } from "@/synnax";

const formSchema = z.object({
  key: z.string(),
  name: z.string().min(1, "Name is required"),
  age: z.number(),
});

interface Params extends Flux.Params {
  key?: string;
}

const client = newTestClient();

const newWrapper =
  (client: Synnax | null): FC<PropsWithChildren> =>
  // eslint-disable-next-line react/display-name
  (props) => (
    <PSynnax.TestProvider client={client}>
      <Sync.Provider {...props} />
    </PSynnax.TestProvider>
  );

describe("useForm", () => {
  describe("no existing entity", () => {
    it("should return the initial values as the form values", async () => {
      const retrieve = vi.fn().mockReturnValue(null);
      const update = vi.fn();
      const { result } = renderHook(
        () =>
          Flux.createForm<Params, typeof formSchema>({
            initialValues: {
              key: "",
              name: "John Doe",
              age: 25,
            },
            schema: formSchema,
            name: "test",
            retrieve,
            update,
          })({ params: {} }),
        { wrapper: newWrapper(client) },
      );
      expect(result.current.form.value()).toEqual({
        key: "",
        name: "John Doe",
        age: 25,
      });
      await waitFor(() => {
        expect(retrieve).toHaveBeenCalledTimes(1);
        expect(update).not.toHaveBeenCalled();
      });
    });
  });

  describe("existing entity", () => {
    it("should return the existing entity as the form values", async () => {
      const retrieve = vi.fn().mockReturnValue({
        key: "123",
        name: "Apple Cat",
        age: 30,
      });
      const { result } = renderHook(
        () =>
          Flux.createForm<Params, typeof formSchema>({
            initialValues: {
              key: "",
              name: "",
              age: 0,
            },
            schema: formSchema,
            name: "test",
            retrieve,
            update: vi.fn(),
          })({ params: {} }),
        { wrapper: newWrapper(client) },
      );
      await waitFor(() => {
        expect(retrieve).toHaveBeenCalledTimes(1);
      });
      expect(result.current.form.value()).toEqual({
        key: "123",
        name: "Apple Cat",
        age: 30,
      });
    });
  });

  it("should validate form values as they are set", async () => {
    const update = vi.fn();
    const retrieve = vi.fn().mockReturnValue(null);
    const { result } = renderHook(
      () =>
        Flux.createForm<Params, typeof formSchema>({
          initialValues: {
            key: "",
            name: "John Doe",
            age: 25,
          },
          schema: formSchema,
          name: "test",
          retrieve,
          update,
        })({ params: {} }),
      { wrapper: newWrapper(client) },
    );

    act(() => {
      result.current.form.set("name", "");
    });

    const fieldStatus = result.current.form.get("name").status;
    expect(fieldStatus.variant).toEqual("error");
    expect(fieldStatus.message).toEqual("Name is required");
  });

  it("should validate form values before saving", async () => {
    const update = vi.fn();
    const retrieve = vi.fn().mockReturnValue(null);
    const { result } = renderHook(
      () =>
        Flux.createForm<Params, typeof formSchema>({
          initialValues: {
            key: "",
            name: "",
            age: 25,
          },
          schema: formSchema,
          name: "test",
          retrieve,
          update,
        })({ params: {} }),
      { wrapper: newWrapper(client) },
    );
    act(() => {
      result.current.save();
    });
    const status = result.current.form.get("name").status;
    expect(status.variant).toEqual("success");
    await waitFor(() => {
      expect(retrieve).toHaveBeenCalledTimes(1);
      expect(update).not.toHaveBeenCalled();
      const fieldStatus = result.current.form.get("name").status;
      expect(fieldStatus.variant).toEqual("error");
      expect(fieldStatus.message).toEqual("Name is required");
    });
  });

  describe("autoSave = false", () => {
    it("should allow the caller to modify the form values without calling update", async () => {
      const update = vi.fn();
      const retrieve = vi.fn().mockReturnValue(null);
      const { result } = renderHook(
        () =>
          Flux.createForm<Params, typeof formSchema>({
            initialValues: {
              key: "",
              name: "John Doe",
              age: 25,
            },
            schema: formSchema,
            name: "test",
            retrieve,
            update,
          })({ params: {} }),
        { wrapper: newWrapper(client) },
      );
      expect(result.current.form.value()).toEqual({
        key: "",
        name: "John Doe",
        age: 25,
      });

      act(() => {
        result.current.form.set("name", "Jane Doe");
      });
      expect(result.current.form.value()).toEqual({
        key: "",
        name: "Jane Doe",
        age: 25,
      });
      await waitFor(() => {
        expect(retrieve).toHaveBeenCalledTimes(1);
        expect(update).not.toHaveBeenCalled();
      });
    });
  });

  it("should call update when the form is saved", async () => {
    const update = vi.fn();
    const retrieve = vi.fn().mockReturnValue(null);
    const { result } = renderHook(
      () =>
        Flux.createForm<Params, typeof formSchema>({
          initialValues: {
            key: "",
            name: "John Doe",
            age: 25,
          },
          schema: formSchema,
          name: "test",
          retrieve,
          update,
        })({ params: {} }),
      { wrapper: newWrapper(client) },
    );
    act(() => {
      result.current.form.set("name", "Jane Doe");
      result.current.save();
    });
    await waitFor(() => {
      expect(retrieve).toHaveBeenCalledTimes(1);
      expect(update).toHaveBeenCalledTimes(1);
    });
  });

  describe("autoSave = true", () => {
    it("should call update when any form values are modified", async () => {
      const update = vi.fn();
      const retrieve = vi.fn().mockReturnValue(null);
      const { result } = renderHook(
        () =>
          Flux.createForm<Params, typeof formSchema>({
            initialValues: {
              key: "",
              name: "John Doe",
              age: 25,
            },
            schema: formSchema,
            name: "test",
            retrieve,
            update: ({ value }) => update(value.name),
          })({ params: {}, autoSave: true }),
        { wrapper: newWrapper(client) },
      );
      act(() => {
        result.current.form.set("name", "Jane Doe");
      });
      await waitFor(() => {
        expect(retrieve).toHaveBeenCalledTimes(1);
        expect(update).toHaveBeenCalledTimes(1);
        expect(update).toHaveBeenCalledWith("Jane Doe");
      });
    });

    it("should not call update when initial values are resolved via retrieve", async () => {
      const update = vi.fn();
      const retrieve = vi.fn().mockReturnValue({
        key: "123",
        name: "Apple Cat",
        age: 30,
      });
      renderHook(
        () =>
          Flux.createForm<Params, typeof formSchema>({
            initialValues: {
              key: "",
              name: "",
              age: 0,
            },
            schema: formSchema,
            name: "test",
            retrieve,
            update: ({ value }) => update(value.name),
          })({ params: {} }),
        { wrapper: newWrapper(client) },
      );
      await waitFor(() => {
        expect(retrieve).toHaveBeenCalledTimes(1);
        expect(update).not.toHaveBeenCalled();
      });
    });
  });
});
