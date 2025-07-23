// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, newTestClient } from "@synnaxlabs/client";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Flux } from "@/flux";
import { newSynnaxWrapper } from "@/testutil/Synnax";

const formSchema = z.object({
  key: z.string(),
  name: z.string().min(1, "Name is required"),
  age: z.number().positive("Age must be positive"),
});

interface Params {
  key?: string;
}

const client = newTestClient();

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
        { wrapper: newSynnaxWrapper(client) },
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
        { wrapper: newSynnaxWrapper(client) },
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
      { wrapper: newSynnaxWrapper(client) },
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
      { wrapper: newSynnaxWrapper(client) },
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
          })({ params: {}, afterSave }),
        { wrapper: newSynnaxWrapper(client) },
      );
      act(() => {
        result.current.save();
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
          })({ params: {}, afterSave }),
        { wrapper: newSynnaxWrapper(client) },
      );
      act(() => {
        result.current.save();
      });
      await waitFor(() => {
        expect(afterSave).not.toHaveBeenCalled();
      });
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
          })({ params: {}, afterSave }),
        { wrapper: newSynnaxWrapper(client) },
      );
      act(() => {
        result.current.save();
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
          })({ params: {} }),
        { wrapper: newSynnaxWrapper(client) },
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
      { wrapper: newSynnaxWrapper(client) },
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
        { wrapper: newSynnaxWrapper(client) },
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
        { wrapper: newSynnaxWrapper(client) },
      );
      await waitFor(() => {
        expect(retrieve).toHaveBeenCalledTimes(1);
        expect(update).not.toHaveBeenCalled();
      });
    });
  });

  describe("listeners", () => {
    it("should correctly update the form data when the listener receives changes", async () => {
      const ch = await client.channels.create({
        name: "Test Channel",
        virtual: true,
        dataType: "float32",
      });

      const initialValues = {
        key: ch.key.toString(),
        name: "Initial Name",
        age: 25,
      };

      const retrieve = vi.fn().mockReturnValue(initialValues);
      const update = vi.fn();

      const { result } = renderHook(
        () =>
          Flux.createForm<Params, typeof formSchema>({
            initialValues: {
              key: ch.key.toString(),
              name: "",
              age: 0,
            },
            schema: formSchema,
            name: "test",
            retrieve,
            update,
            listeners: [
              {
                channel: channel.SET_CHANNEL_NAME,
                onChange: async ({ client, params, onChange }) => {
                  if (ch.key.toString() !== params.key) return;
                  const updatedChannel = await client.channels.retrieve(ch.key);
                  onChange((prev) => {
                    if (prev == null) return prev;
                    return {
                      ...prev,
                      name: updatedChannel.name,
                    };
                  });
                },
              },
            ],
          })({ params: { key: ch.key.toString() } }),
        { wrapper: newSynnaxWrapper(client) },
      );

      await waitFor(() => {
        expect(result.current.form.value()).toEqual(initialValues);
        expect(result.current.variant).toEqual("success");
      });

      // Trigger a channel name change which should invoke the listener
      await act(async () => {
        await client.channels.rename(ch.key, "Updated Channel Name");
      });

      await waitFor(() => {
        expect(result.current.form.value().name).toEqual("Updated Channel Name");
        expect(result.current.variant).toEqual("success");
      });
    });

    it("should move the form into an error state when the listener throws an error", async () => {
      const ch = await client.channels.create({
        name: "Test Channel",
        virtual: true,
        dataType: "float32",
      });

      const initialValues = {
        key: ch.key.toString(),
        name: "Initial Name",
        age: 25,
      };

      const retrieve = vi.fn().mockReturnValue(initialValues);
      const update = vi.fn();

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
            update,
            listeners: [
              {
                channel: channel.SET_CHANNEL_NAME,
                onChange: async () => {
                  throw new Error("Listener error");
                },
              },
            ],
          })({ params: {} }),
        { wrapper: newSynnaxWrapper(client) },
      );

      await waitFor(() => {
        expect(result.current.form.value()).toEqual(initialValues);
        expect(result.current.variant).toEqual("success");
      });

      await act(async () => {
        await client.channels.rename(ch.key, "Updated Channel Name");
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("error");
        expect(result.current.error).toEqual(new Error("Listener error"));
      });
    });
  });
});
