// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, label } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Label } from "@/label";
import { Ontology } from "@/ontology";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

describe("queries", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({
      client,
      excludeFluxStores: [Ontology.RESOURCES_FLUX_STORE_KEY],
    });
  });
  describe("useList", () => {
    it("should return a list of label keys", async () => {
      const label1 = await client.labels.create({
        name: "label1",
        color: "#FF0000",
      });
      const label2 = await client.labels.create({
        name: "label2",
        color: "#00FF00",
      });

      const { result } = renderHook(() => Label.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data.length).toBeGreaterThanOrEqual(2);
      expect(result.current.data).toContain(label1.key);
      expect(result.current.data).toContain(label2.key);
    });

    it("should get individual labels using getItem", async () => {
      const testLabel = await client.labels.create({
        name: "testLabel",
        color: "#E774D0",
      });

      const { result } = renderHook(() => Label.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const retrievedLabel = result.current.getItem(testLabel.key);
      expect(retrievedLabel?.key).toEqual(testLabel.key);
      expect(retrievedLabel?.name).toEqual("testLabel");
      expect(retrievedLabel?.color).toEqual(color.construct("#E774D0"));
    });

    it("should filter labels by search term", async () => {
      await client.labels.create({
        name: "ordinary_label",
        color: "#FF0000",
      });
      await client.labels.create({
        name: "special_label",
        color: "#00FF00",
      });

      const { result } = renderHook(() => Label.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({ searchTerm: "special" });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data.length).toBeGreaterThanOrEqual(1);
      expect(
        result.current.data
          .map((key: label.Key) => result.current.getItem(key)?.name)
          .includes("special_label"),
      ).toBe(true);
    });

    it("should handle pagination with limit and offset", async () => {
      const labels = await Promise.all(
        Array.from({ length: 5 }).map((_, i) =>
          client.labels.create({
            name: `paginationLabel${i}`,
            color: "#0000FF",
          }),
        ),
      );

      const { result } = renderHook(() => Label.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({
          limit: 2,
          offset: 1,
          keys: labels.map((l) => l.key),
        });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toHaveLength(2);
    });

    it("should update the list when a label is created", async () => {
      const { result } = renderHook(() => Label.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      const initialLength = result.current.data.length;

      const newLabel = await client.labels.create({
        name: "newLabel",
        color: "#FFFF00",
      });

      await waitFor(() => {
        expect(result.current.data).toHaveLength(initialLength + 1);
        expect(result.current.data).toContain(newLabel.key);
      });
    });

    it("should update the list when a label is updated", async () => {
      const testLabel = await client.labels.create({
        name: "original",
        color: "#FF0000",
      });

      const { result } = renderHook(() => Label.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.getItem(testLabel.key)?.name).toEqual("original");

      const updatedLabel = await client.labels.create({
        ...testLabel,
        name: "updated",
      });

      await waitFor(() => {
        expect(result.current.getItem(updatedLabel.key)?.name).toEqual("updated");
      });
    });

    it("should remove label from list when deleted", async () => {
      const testLabel = await client.labels.create({
        name: "toDelete",
        color: "#FF0000",
      });

      const { result } = renderHook(() => Label.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.data).toContain(testLabel.key);

      await client.labels.delete(testLabel.key);

      await waitFor(() => {
        expect(result.current.data).not.toContain(testLabel.key);
      });
    });
  });

  describe("retrieveLabelsOf", () => {
    it("should retrieve labels for an ontology ID", async () => {
      const label1 = await client.labels.create({
        name: "entityLabel1",
        color: "#FF0000",
      });
      const label2 = await client.labels.create({
        name: "entityLabel2",
        color: "#00FF00",
      });
      const targetLabel = await client.labels.create({
        name: "targetEntity",
        color: "#0000FF",
      });

      await client.labels.label(label.ontologyID(targetLabel.key), [
        label1.key,
        label2.key,
      ]);

      const { result } = renderHook(
        () => Label.useRetrieveLabelsOf({ id: label.ontologyID(targetLabel.key) }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.data).toHaveLength(2);
      expect(result.current.data?.map((l) => l.key)).toContain(label1.key);
      expect(result.current.data?.map((l) => l.key)).toContain(label2.key);
    });

    it("should update when labels are added to entity", async () => {
      const targetLabel = await client.labels.create({
        name: "targetForLabeling",
        color: "#0000FF",
      });

      const { result } = renderHook(
        () =>
          Label.useRetrieveLabelsOf({
            id: label.ontologyID(targetLabel.key),
          }),
        { wrapper },
      );
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      const initialLength = result.current.data?.length ?? 0;

      const newLabel = await client.labels.create({
        name: "addedLabel",
        color: "#FFFF00",
      });
      await client.labels.label(label.ontologyID(targetLabel.key), [newLabel.key]);

      await waitFor(() => {
        expect(result.current.data).toHaveLength(initialLength + 1);
        expect(result.current.data?.map((l) => l.key)).toContain(newLabel.key);
      });
    });

    it("should update when labels are removed from entity", async () => {
      const labelToRemove = await client.labels.create({
        name: "labelToRemove",
        color: "#FF0000",
      });
      const targetLabel = await client.labels.create({
        name: "targetForRemoval",
        color: "#0000FF",
      });

      await client.labels.label(label.ontologyID(targetLabel.key), [labelToRemove.key]);

      const { result } = renderHook(
        () =>
          Label.useRetrieveLabelsOf({
            id: label.ontologyID(targetLabel.key),
          }),
        { wrapper },
      );
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.data?.map((l) => l.key)).toContain(labelToRemove.key);

      await client.labels.label(label.ontologyID(targetLabel.key), [], {
        replace: true,
      });

      await waitFor(() => {
        expect(result.current.data?.map((l) => l.key)).not.toContain(labelToRemove.key);
      });
    });

    it("should update when a label itself is updated", async () => {
      const originalLabel = await client.labels.create({
        name: "originalName",
        color: "#FF0000",
      });
      const targetLabel = await client.labels.create({
        name: "targetEntity",
        color: "#0000FF",
      });

      await client.labels.label(label.ontologyID(targetLabel.key), [originalLabel.key]);

      const { result } = renderHook(
        () =>
          Label.useRetrieveLabelsOf({
            id: label.ontologyID(targetLabel.key),
          }),
        { wrapper },
      );
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(
        result.current.data?.find((l) => l.key === originalLabel.key)?.name,
      ).toEqual("originalName");

      const updatedLabel = await client.labels.create({
        ...originalLabel,
        name: "updatedName",
      });

      await waitFor(() => {
        expect(
          result.current.data?.find((l) => l.key === updatedLabel.key)?.name,
        ).toEqual("updatedName");
      });
    });

    it("should update when a label itself is deleted", async () => {
      const labelToDelete = await client.labels.create({
        name: "labelToDelete",
        color: "#FF0000",
      });
      const targetLabel = await client.labels.create({
        name: "targetEntity",
        color: "#0000FF",
      });

      await client.labels.label(label.ontologyID(targetLabel.key), [labelToDelete.key]);

      const { result } = renderHook(
        () =>
          Label.useRetrieveLabelsOf({
            id: label.ontologyID(targetLabel.key),
          }),
        { wrapper },
      );
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.data?.map((l) => l.key)).toContain(labelToDelete.key);

      await client.labels.delete(labelToDelete.key);

      await waitFor(() => {
        expect(result.current.data?.map((l) => l.key)).not.toContain(labelToDelete.key);
      });
    });
    describe("retrieveMultiple", () => {
      describe("useDirect", () => {
        it("should retrieve multiple labels", async () => {
          const labels = await client.labels.create([
            { name: "label1", color: "#FF0000" },
            { name: "label2", color: "#00FF00" },
          ]);
          const { result } = renderHook(
            () => Label.useRetrieveMultiple({ keys: labels.map((l) => l.key) }),
            { wrapper },
          );
          await waitFor(() => expect(result.current.variant).toEqual("success"));
          expect(result.current.data).toEqual(labels);
        });
        it("should update when a label changes", async () => {
          const labels = await client.labels.create([
            { name: "label1", color: "#FF0000" },
            { name: "label2", color: "#00FF00" },
          ]);
          const { result } = renderHook(
            () => Label.useRetrieveMultiple({ keys: labels.map((l) => l.key) }),
            { wrapper },
          );
          await waitFor(() => expect(result.current.variant).toEqual("success"));
          expect(result.current.data).toEqual(labels);
          labels[0] = await client.labels.create({
            ...labels[0],
            name: "updatedLabel",
          });
          await waitFor(() => {
            expect(result.current.data).toEqual(expect.arrayContaining(labels));
          });
        });
      });
      describe("useList", () => {
        it("should retrieve labels", async () => {
          const labels = await client.labels.create([
            { name: "label1", color: "#FF0000" },
            { name: "label2", color: "#00FF00" },
          ]);
          const { result } = renderHook(
            () => Label.useList({ initialQuery: { keys: labels.map((l) => l.key) } }),
            { wrapper },
          );
          result.current.retrieve({});
          await result.current.retrieveAsync({
            keys: labels.map((l) => l.key),
          });
          await waitFor(() => expect(result.current.variant).toEqual("success"));
          const label = result.current.getItem(labels[0].key);
          expect(label).toEqual(labels[0]);
        });
      });
    });
  });

  describe("useForm", () => {
    it("should create a new label", async () => {
      const { result } = renderHook(() => Label.useForm({ query: {} }), {
        wrapper,
      });

      act(() => {
        result.current.form.set("name", "newFormLabel");
        result.current.form.set("color", "#FF00FF");
        result.current.save();
      });

      await waitFor(() => {
        expect(result.current.form.value().name).toEqual("newFormLabel");
        expect(result.current.form.value().color).toEqual(color.construct("#FF00FF"));
        expect(result.current.form.value().key).toBeDefined();
      });
    });

    it("should retrieve and edit existing label", async () => {
      const existingLabel = await client.labels.create({
        name: "existingLabel",
        color: "#00FFFF",
      });

      const { result } = renderHook(
        () => Label.useForm({ query: { key: existingLabel.key } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.form.value().name).toEqual("existingLabel");
      expect(result.current.form.value().color).toEqual(color.construct("#00FFFF"));

      act(() => {
        result.current.form.set("name", "editedLabel");
        result.current.save();
      });

      await waitFor(() => {
        expect(result.current.form.value().name).toEqual("editedLabel");
      });
    });

    it("should update form when label is updated externally", async () => {
      const testLabel = await client.labels.create({
        name: "externalUpdate",
        color: "#FF0000",
      });

      const { result } = renderHook(
        () => Label.useForm({ query: { key: testLabel.key } }),
        { wrapper },
      );
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.form.value().name).toEqual("externalUpdate");

      await client.labels.create({
        ...testLabel,
        name: "externallyUpdated",
      });

      await waitFor(() => {
        expect(result.current.form.value().name).toEqual("externallyUpdated");
      });
    });

    it("should handle form with default values", async () => {
      const { result } = renderHook(() => Label.useForm({ query: {} }), {
        wrapper,
      });

      expect(result.current.form.value().name).toEqual("");
      expect(result.current.form.value().color).toEqual(color.construct("#000000"));
    });
  });

  describe("useDelete", () => {
    it("should delete a label", async () => {
      const labelToDelete = await client.labels.create({
        name: "deleteMe",
        color: "#FF0000",
      });

      const { result } = renderHook(Label.useDelete, { wrapper });

      act(() => {
        result.current.update(labelToDelete.key);
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      await expect(
        async () => await client.labels.retrieve({ key: labelToDelete.key }),
      ).rejects.toThrow();
    });

    it("should handle delete operations in sequence", async () => {
      const label1 = await client.labels.create({
        name: "delete1",
        color: "#FF0000",
      });
      const label2 = await client.labels.create({
        name: "delete2",
        color: "#00FF00",
      });

      const { result: result1 } = renderHook(Label.useDelete, { wrapper });
      const { result: result2 } = renderHook(Label.useDelete, { wrapper });

      act(() => {
        result1.current.update(label1.key);
      });
      await waitFor(() => expect(result1.current.variant).toEqual("success"));

      act(() => {
        result2.current.update(label2.key);
      });
      await waitFor(() => expect(result2.current.variant).toEqual("success"));

      await expect(
        async () => await client.labels.retrieve({ key: label1.key }),
      ).rejects.toThrow();
      await expect(
        async () => await client.labels.retrieve({ key: label2.key }),
      ).rejects.toThrow();
    });
  });
});
