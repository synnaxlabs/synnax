// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { query } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { color, testutil, TimeSpan } from "@synnaxlabs/x";
import { act, render, renderHook, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Errors } from "@/errors";
import { Flux } from "@/flux";
import { renderHookSuspended } from "@/testutil/render";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const formSchema = z.object({
  key: z.string(),
  name: z.string().min(1, "Name is required"),
  age: z.number().positive("Age must be positive"),
});

type Params = {
  key?: string;
};

const client = createTestClient();
const Wrapper = createSynnaxWrapper({ client });

const DEBOUNCE = TimeSpan.milliseconds(100);
// Long enough that a debounced save would have landed by now.
const SETTLE = 500;

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
          })({ query: null }),
        { wrapper: Wrapper },
      );
      expect(result.current.form.value()).toEqual({
        key: "",
        name: "John Doe",
        age: 25,
      });
      await waitFor(() => {
        expect(retrieve).not.toHaveBeenCalled();
        expect(update).not.toHaveBeenCalled();
      });
    });
  });

  describe("existing entity", () => {
    it("should return the existing entity as the form values", async () => {
      const retrieve = vi.fn(async () => ({
        key: "123",
        name: "Apple Cat",
        age: 30,
      }));
      const useForm = Flux.createForm<Params, typeof formSchema>({
        initialValues: {
          key: "",
          name: "",
          age: 0,
        },
        schema: formSchema,
        name: "test",
        retrieve,
        update: vi.fn(),
      });
      const { result } = await renderHookSuspended(
        () => useForm({ query: { key: "123" } }),
        { wrapper: Wrapper },
      );
      expect(retrieve).toHaveBeenCalledTimes(1);
      expect(result.current.form.value()).toEqual({
        key: "123",
        name: "Apple Cat",
        age: 30,
      });
    });

    it("should route a failed read to the error boundary", async () => {
      const useForm = Flux.createForm<Params, typeof formSchema>({
        initialValues: { key: "", name: "", age: 0 },
        schema: formSchema,
        name: "test",
        retrieve: async () => {
          throw new Error("boom");
        },
        update: vi.fn(),
      });
      const Display = (): ReactElement => {
        useForm({ query: { key: "gone" } });
        return <div data-testid="loaded" />;
      };
      let utils!: ReturnType<typeof render>;
      await act(async () => {
        utils = render(
          <Wrapper>
            <Errors.SuspenseBoundary
              loading={<div>loading</div>}
              FallbackComponent={({ error }) => (
                <div data-testid="error">{error.message}</div>
              )}
            >
              <Display />
            </Errors.SuspenseBoundary>
          </Wrapper>,
        );
      });
      await act(async () => {
        await Promise.resolve();
      });
      expect(utils.queryByTestId("error")?.textContent).toBe("Failed to retrieve test");
    });

    it("should serve a cached answer without fetching", () => {
      const retrieve = vi.fn();
      const useForm = Flux.createForm<Params, typeof formSchema>({
        initialValues: { key: "", name: "", age: 0 },
        schema: formSchema,
        name: "test",
        retrieve,
        getCached: () => ({ key: "123", name: "Cached Cat", age: 30 }),
        update: vi.fn(),
      });
      const { result } = renderHook(() => useForm({ query: { key: "123" } }), {
        wrapper: Wrapper,
      });
      expect(retrieve).not.toHaveBeenCalled();
      expect(result.current.form.value()).toEqual({
        key: "123",
        name: "Cached Cat",
        age: 30,
      });
    });

    it("should read the cache once per query rather than once per render", () => {
      const getCached = vi.fn(() => ({ key: "123", name: "Cached Cat", age: 30 }));
      const useForm = Flux.createForm<Params, typeof formSchema>({
        initialValues: { key: "", name: "", age: 0 },
        schema: formSchema,
        name: "test",
        retrieve: vi.fn(),
        getCached,
        update: vi.fn(),
      });
      const { rerender } = renderHook(() => useForm({ query: { key: "123" } }), {
        wrapper: Wrapper,
      });
      const onMount = getCached.mock.calls.length;
      rerender();
      rerender();
      expect(getCached).toHaveBeenCalledTimes(onMount);
    });

    it("should not refetch when the fetch lands the answer in the cache", async () => {
      const cat = { key: "123", name: "Apple Cat", age: 30 };
      let cached: typeof cat | undefined;
      const retrieve = vi.fn(async () => {
        cached = cat;
        return cat;
      });
      const useForm = Flux.createForm<Params, typeof formSchema>({
        initialValues: { key: "", name: "", age: 0 },
        schema: formSchema,
        name: "test",
        retrieve,
        getCached: () => cached,
        update: vi.fn(),
      });
      const { result } = await renderHookSuspended(
        () => useForm({ query: { key: "123" } }),
        { wrapper: Wrapper },
      );
      expect(result.current.form.value()).toEqual(cat);
      expect(retrieve).toHaveBeenCalledTimes(1);
    });

    it("should replace the form values when the query points at another record", async () => {
      const cats: Record<string, z.infer<typeof formSchema>> = {
        "1": { key: "1", name: "Apple Cat", age: 30 },
        "2": { key: "2", name: "Banana Cat", age: 12 },
      };
      const useForm = Flux.createForm<Params, typeof formSchema>({
        initialValues: { key: "", name: "", age: 0 },
        schema: formSchema,
        name: "test",
        retrieve: async ({ query: { key } }) => cats[key as string],
        update: vi.fn(),
      });
      const { result, rerender } = await renderHookSuspended(
        ({ key }: { key: string }) => useForm({ query: { key } }),
        { wrapper: Wrapper, initialProps: { key: "1" } },
      );
      expect(result.current.form.value().name).toEqual("Apple Cat");
      await act(async () => {
        rerender({ key: "2" });
      });
      expect(result.current.form.value()).toEqual(cats["2"]);
    });
  });

  describe("getCached", () => {
    const CACHED = { key: "123", name: "Apple Cat", age: 30 };
    const FETCHED = { key: "123", name: "Fetched", age: 40 };
    const INITIAL = { key: "", name: "", age: 0 };

    // The definition is minted once, as production does: a per-render one would
    // hand every render a fresh fetch-dedup cache.
    const renderForm = async (
      getCached: (params: { query: Params }) => typeof CACHED | undefined,
      params: Partial<Flux.UseFormParams<Params, typeof formSchema>> = {},
    ) => {
      const retrieve = vi.fn(async () => FETCHED);
      const useForm = Flux.createForm<Params, typeof formSchema>({
        initialValues: INITIAL,
        schema: formSchema,
        name: "test",
        getCached,
        retrieve,
        update: vi.fn(),
      });
      const { result } = await renderHookSuspended(
        () => useForm({ query: { key: "123" }, ...params }),
        { wrapper: Wrapper },
      );
      return { result, retrieve };
    };

    it("should fill the form from the cache without fetching", async () => {
      const { result, retrieve } = await renderForm(() => CACHED);
      expect(result.current.form.value()).toEqual(CACHED);
      expect(retrieve).not.toHaveBeenCalled();
    });

    it("should fetch when the cache misses", async () => {
      const { result, retrieve } = await renderForm(() => undefined);
      expect(retrieve).toHaveBeenCalledTimes(1);
      expect(result.current.form.value()).toEqual(FETCHED);
    });

    it("should prefer the record it read over explicit initial values", async () => {
      const { result, retrieve } = await renderForm(() => CACHED, {
        initialValues: { key: "456", name: "Explicit", age: 50 },
      });
      expect(result.current.form.value()).toEqual(CACHED);
      expect(retrieve).not.toHaveBeenCalled();
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
        })({ query: null }),
      { wrapper: Wrapper },
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
        })({ query: null }),
      { wrapper: Wrapper },
    );
    act(() => {
      result.current.save({ signal: controller.signal });
    });
    const status = result.current.form.get("name").status;
    expect(status.variant).toEqual("success");
    await waitFor(() => {
      expect(retrieve).not.toHaveBeenCalled();
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
          })({ query: null, afterSave }),
        { wrapper: Wrapper },
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
          })({ query: null, afterSave }),
        { wrapper: Wrapper },
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
          })({ query: null, afterSave }),
        { wrapper: Wrapper },
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
          })({ query: null }),
        { wrapper: Wrapper },
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
        expect(retrieve).not.toHaveBeenCalled();
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
        })({ query: null }),
      { wrapper: Wrapper },
    );
    act(() => {
      result.current.form.set("name", "Jane Doe");
      result.current.save({ signal: controller.signal });
    });
    await waitFor(() => {
      expect(retrieve).not.toHaveBeenCalled();
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
          })({ query: null, autoSave: true }),
        { wrapper: Wrapper },
      );
      act(() => {
        result.current.form.set("name", "Jane Doe");
      });
      await waitFor(() => {
        expect(retrieve).not.toHaveBeenCalled();
        expect(update).toHaveBeenCalledTimes(1);
        expect(update).toHaveBeenCalledWith("Jane Doe");
      });
    });

    it("should not call update when initial values are resolved via retrieve", async () => {
      const update = vi.fn();
      const retrieve = vi.fn().mockResolvedValue({
        key: "123",
        name: "Apple Cat",
        age: 30,
      });
      const useForm = Flux.createForm<Params, typeof formSchema>({
        initialValues: {
          key: "",
          name: "",
          age: 0,
        },
        schema: formSchema,
        name: "test",
        retrieve,
        update: ({ value }) => update(value.name),
      });
      await renderHookSuspended(
        () => useForm({ query: { key: "123" }, autoSave: true }),
        { wrapper: Wrapper },
      );
      await waitFor(() => {
        expect(retrieve).toHaveBeenCalledTimes(1);
        expect(update).not.toHaveBeenCalled();
      });
    });

    it("should update once per change when no debounce is set", async () => {
      const update = vi.fn();
      const { result } = renderHook(
        () =>
          Flux.createForm<Params, typeof formSchema>({
            initialValues: { key: "", name: "John Doe", age: 25 },
            schema: formSchema,
            name: "test",
            update: ({ get }) => update(get("name").value),
          })({ query: null, autoSave: true }),
        { wrapper: Wrapper },
      );
      act(() => {
        result.current.form.set("name", "J");
        result.current.form.set("name", "Ja");
        result.current.form.set("name", "Jane");
      });
      await waitFor(() => expect(update).toHaveBeenCalledTimes(3));
      expect(update).toHaveBeenLastCalledWith("Jane");
    });
  });

  describe("autoSaveDebounce", () => {
    const renderDebouncedForm = () => {
      const update = vi.fn();
      const rendered = renderHook(
        () =>
          Flux.createForm<Params, typeof formSchema>({
            initialValues: { key: "", name: "John Doe", age: 25 },
            schema: formSchema,
            name: "test",
            update: ({ get }) => update(get("name").value),
          })({ query: null, autoSave: true, autoSaveDebounce: DEBOUNCE }),
        { wrapper: Wrapper },
      );
      return { ...rendered, update };
    };

    it("should collapse a burst of changes into a single update", async () => {
      const { result, update } = renderDebouncedForm();
      act(() => {
        result.current.form.set("name", "J");
        result.current.form.set("name", "Ja");
        result.current.form.set("name", "Jane");
      });
      await waitFor(() => expect(update).toHaveBeenCalledTimes(1));
      expect(update).toHaveBeenCalledWith("Jane");
      await testutil.expectAlways(
        () => expect(update).toHaveBeenCalledTimes(1),
        SETTLE,
      );
    });

    it("should flush a pending update when the form unmounts", async () => {
      const { result, update, unmount } = renderDebouncedForm();
      act(() => {
        result.current.form.set("name", "Jane Doe");
      });
      unmount();
      await waitFor(() => expect(update).toHaveBeenCalledWith("Jane Doe"));
    });
  });

  describe("abandon", () => {
    // Hands back the abandon handle the form passes to its listeners, so the
    // tests can play the record being deleted out from under the form.
    const createAbandonableForm = () => {
      const update = vi.fn();
      let abandon = (): void => {};
      const useForm = Flux.createForm<Params, typeof formSchema>({
        initialValues: { key: "", name: "John Doe", age: 25 },
        schema: formSchema,
        name: "test",
        update: ({ get }) => update(get("name").value),
        mountListeners: (params) => {
          abandon = params.abandon;
          return () => {};
        },
      });
      return { update, useForm, abandon: () => abandon() };
    };

    // Debounced so a save is genuinely queued when the record is abandoned.
    const renderAbandonableForm = () => {
      const { update, useForm, abandon } = createAbandonableForm();
      const rendered = renderHook(
        () =>
          useForm({
            query: { key: "123" },
            autoSave: true,
            autoSaveDebounce: DEBOUNCE,
          }),
        { wrapper: Wrapper },
      );
      return { ...rendered, update, abandon };
    };

    it("should autosave while the record still exists", async () => {
      const { result, update } = renderAbandonableForm();
      act(() => result.current.form.set("name", "Jane Doe"));
      await waitFor(() => expect(update).toHaveBeenCalledWith("Jane Doe"));
    });

    it("should drop a pending autosave when the record is abandoned", async () => {
      const { result, update, abandon } = renderAbandonableForm();
      act(() => result.current.form.set("name", "Jane Doe"));
      act(() => abandon());
      await testutil.expectAlways(() => expect(update).not.toHaveBeenCalled(), SETTLE);
    });

    it("should ignore later changes once the record is abandoned", async () => {
      const { result, update, abandon } = renderAbandonableForm();
      act(() => abandon());
      act(() => result.current.form.set("name", "Jane Doe"));
      await testutil.expectAlways(() => expect(update).not.toHaveBeenCalled(), SETTLE);
    });

    it("should autosave again once the form re-points at another record", async () => {
      const { update, useForm, abandon } = createAbandonableForm();
      const { result, rerender } = renderHook(
        ({ key }: { key: string }) =>
          useForm({ query: { key }, autoSave: true, autoSaveDebounce: DEBOUNCE }),
        { wrapper: Wrapper, initialProps: { key: "123" } },
      );
      act(() => abandon());
      rerender({ key: "456" });
      act(() => result.current.form.set("name", "Jane Doe"));
      await waitFor(() => expect(update).toHaveBeenCalledWith("Jane Doe"));
    });

    it("should not flush a pending autosave on unmount after abandoning", async () => {
      const { result, update, abandon, unmount } = renderAbandonableForm();
      act(() => result.current.form.set("name", "Jane Doe"));
      act(() => abandon());
      unmount();
      await testutil.expectAlways(() => expect(update).not.toHaveBeenCalled(), SETTLE);
    });
  });

  describe("listeners", () => {
    it("should correctly update the form data when the listener receives changes", async () => {
      const label = await client.labels.create({
        name: "Initial Name",
        color: color.construct("#000000"),
      });

      const initialValues = {
        key: label.key.toString(),
        name: "Initial Name",
        age: 25,
      };

      const retrieve = async () => initialValues;
      const update = vi.fn();

      const useForm = Flux.createForm<Params, typeof formSchema>({
        initialValues: {
          key: label.key.toString(),
          name: "",
          age: 0,
        },
        schema: formSchema,
        name: "test",
        retrieve,
        update,
        mountListeners: ({ client, set }) =>
          client.labels.onChange(label.key, (result) => {
            if (query.isLive(result)) set("name", result.name);
          }),
      });

      const { result } = await renderHookSuspended(
        () => useForm({ query: { key: label.key } }),
        { wrapper: Wrapper },
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
        color: color.construct("#000000"),
      });

      const initialValues = {
        key: label.key.toString(),
        name: "Initial Name",
        age: 25,
      };

      const retrieve = async () => initialValues;
      const update = vi.fn();

      const useForm = Flux.createForm<Params, typeof formSchema>({
        initialValues: {
          key: label.key.toString(),
          name: "",
          age: 0,
        },
        schema: formSchema,
        name: "test",
        retrieve,
        update,
        mountListeners: ({ client, set }) =>
          client.labels.onChange(label.key, (result) => {
            if (query.isLive(result)) set("name", result.name);
          }),
      });

      const { result } = await renderHookSuspended(
        () => useForm({ query: { key: label.key } }),
        { wrapper: Wrapper },
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
