// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type label } from "@synnaxlabs/client";
import { testutil } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Flux } from "@/flux";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const formSchema = z.object({
  key: z.string(),
  name: z.string().min(1, "Name is required"),
  age: z.number().positive("Age must be positive"),
});

interface Params {
  key?: string;
}

const client = createTestClient();
const wrapper = createSynnaxWrapper({ client });

describe("useForm", () => {
  let controller: AbortController;
  beforeEach(() => {
    controller = new AbortController();
  });
  afterEach(() => {
    controller.abort();
  });
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
          })({ query: {} }),
        { wrapper },
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
      const retrieve = vi.fn(
        async ({
          reset,
        }: Flux.FormRetrieveParams<Params, typeof formSchema, FluxStore>) =>
          reset({
            key: "123",
            name: "Apple Cat",
            age: 30,
          }),
      );
      const { result } = renderHook(
        () =>
          Flux.createForm<Params, typeof formSchema, FluxStore>({
            initialValues: {
              key: "",
              name: "",
              age: 0,
            },
            schema: formSchema,
            name: "test",
            retrieve,
            update: vi.fn(),
          })({ query: {} }),
        { wrapper },
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
        })({ query: {} }),
      { wrapper },
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
        })({ query: {} }),
      { wrapper },
    );
    act(() => {
      result.current.save({ signal: controller.signal });
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

  describe("afterSave", () => {
    it("should call the afterSave function after the form is validated and updated successfully", async () => {
      const afterSave = vi.fn();
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
            retrieve: vi.fn().mockReturnValue(null),
            update: vi.fn(),
          })({ query: {}, afterSave }),
        { wrapper },
      );
      act(() => {
        result.current.save({ signal: controller.signal });
      });
      await waitFor(() => {
        expect(afterSave).toHaveBeenCalledTimes(1);
      });
    });

    it("should not call afterSave if the update function fails", async () => {
      const afterSave = vi.fn();
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
            retrieve: vi.fn().mockReturnValue(null),
            update: vi.fn().mockRejectedValue(new Error("Update failed")),
          })({ query: {}, afterSave }),
        { wrapper },
      );
      act(() => result.current.save({ signal: controller.signal }));
      await testutil.expectAlways(() => expect(afterSave).not.toHaveBeenCalled());
    });

    it("should not call afterSave if the form is not valid", async () => {
      const afterSave = vi.fn();
      const { result } = renderHook(
        () =>
          Flux.createForm<Params, typeof formSchema>({
            initialValues: {
              key: "",
              name: "",
              age: -10,
            },
            schema: formSchema,
            name: "test",
            retrieve: vi.fn().mockReturnValue(null),
            update: vi.fn(),
          })({ query: {}, afterSave }),
        { wrapper },
      );
      act(() => {
        result.current.save({ signal: controller.signal });
      });
      await waitFor(() => {
        expect(afterSave).not.toHaveBeenCalled();
      });
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
          })({ query: {} }),
        { wrapper },
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
        })({ query: {} }),
      { wrapper },
    );
    act(() => {
      result.current.form.set("name", "Jane Doe");
      result.current.save({ signal: controller.signal });
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
            update: ({ get }) => update(get("name").value),
          })({ query: {}, autoSave: true }),
        { wrapper },
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
          })({ query: {} }),
        { wrapper },
      );
      await waitFor(() => {
        expect(retrieve).toHaveBeenCalledTimes(1);
        expect(update).not.toHaveBeenCalled();
      });
    });
  });

  interface FluxStore extends Flux.Store {
    labels: Flux.UnaryStore<label.Key, label.Label>;
  }

  describe("listeners", () => {
    it("should correctly update the form data when the listener receives changes", async () => {
      const label = await client.labels.create({
        name: "Initial Name",
        color: "#000000",
      });

      const initialValues = {
        key: label.key.toString(),
        name: "Initial Name",
        age: 25,
      };

      const retrieve = async ({
        reset,
      }: Flux.FormRetrieveParams<Params, typeof formSchema, FluxStore>) =>
        reset(initialValues);
      const update = vi.fn();

      const { result } = renderHook(
        () =>
          Flux.createForm<Params, typeof formSchema, FluxStore>({
            initialValues: {
              key: label.key.toString(),
              name: "",
              age: 0,
            },
            schema: formSchema,
            name: "test",
            retrieve,
            update,
            mountListeners: ({ store, set }) =>
              store.labels.onSet((changed) => set("name", changed.name), label.key),
          })({ query: { key: label.key } }),
        { wrapper },
      );

      await waitFor(() => {
        expect(result.current.form.value()).toEqual(initialValues);
        expect(result.current.variant).toEqual("success");
      });

      await act(async () => {
        await client.labels.create({
          ...label,
          name: "Updated Label Name",
        });
      });

      await waitFor(() => {
        expect(result.current.form.value().name).toEqual("Updated Label Name");
        expect(result.current.variant).toEqual("success");
      });
    });

    it("should not mark form fields as touched when setting them view the listener", async () => {
      const label = await client.labels.create({
        name: "Initial Name",
        color: "#000000",
      });

      const initialValues = {
        key: label.key.toString(),
        name: "Initial Name",
        age: 25,
      };

      const retrieve = async ({
        reset,
      }: Flux.FormRetrieveParams<Params, typeof formSchema, FluxStore>) =>
        reset(initialValues);
      const update = vi.fn();

      const { result } = renderHook(
        () =>
          Flux.createForm<Params, typeof formSchema, FluxStore>({
            initialValues: {
              key: label.key.toString(),
              name: "",
              age: 0,
            },
            schema: formSchema,
            name: "test",
            retrieve,
            update,
            mountListeners: ({ store, set }) =>
              store.labels.onSet((changed) => set("name", changed.name), label.key),
          })({ query: { key: label.key } }),
        { wrapper },
      );

      await waitFor(() => {
        expect(result.current.form.value()).toEqual(initialValues);
        expect(result.current.variant).toEqual("success");
      });

      await act(async () => {
        await client.labels.create({
          ...label,
          name: "Updated Label Name",
        });
      });

      await waitFor(() => {
        expect(result.current.form.value().name).toEqual("Updated Label Name");
        expect(result.current.variant).toEqual("success");
        expect(result.current.form.get("name").touched).toBe(false);
      });
    });
  });
});
