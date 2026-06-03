// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, NotFoundError } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { LinePlot } from "@/lineplot";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

describe("lineplot queries", () => {
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("useRetrieve", () => {
    it("should retrieve a line plot by key", async () => {
      const project = await client.projects.create({
        name: "test_project",
      });
      const plot = await client.lineplots.create(project.key, {
        name: "retrieve_test",
      });

      const { result } = renderHook(() => LinePlot.useRetrieve({ key: plot.key }), {
        wrapper,
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.data?.key).toEqual(plot.key);
      expect(result.current.data?.name).toEqual("retrieve_test");
    });

    it("should cache retrieved line plots", async () => {
      const project = await client.projects.create({
        name: "cache_project",
      });
      const plot = await client.lineplots.create(project.key, {
        name: "cached_plot",
      });

      const { result: result1 } = renderHook(
        () => LinePlot.useRetrieve({ key: plot.key }),
        { wrapper },
      );
      await waitFor(() => expect(result1.current.variant).toEqual("success"));

      const { result: result2 } = renderHook(
        () => LinePlot.useRetrieve({ key: plot.key }),
        { wrapper },
      );
      await waitFor(() => expect(result2.current.variant).toEqual("success"));
      expect(result2.current.data).toEqual(result1.current.data);
    });
  });

  describe("useCreate", () => {
    it("should create a new line plot", async () => {
      const project = await client.projects.create({
        name: "create_project",
      });

      const { result } = renderHook(() => LinePlot.useCreate(), { wrapper });

      const key = uuid.create();
      await act(async () => {
        await result.current.updateAsync({
          key,
          project: project.key,
          name: "created_plot",
        });
      });

      expect(result.current.variant).toEqual("success");
      expect(result.current.data?.name).toEqual("created_plot");
      expect(result.current.data?.project).toEqual(project.key);

      const retrieved = await client.lineplots.retrieve({ key });
      expect(retrieved.name).toEqual("created_plot");
    });

    it("should store created line plot in flux store", async () => {
      const project = await client.projects.create({
        name: "store_project",
      });

      const { result: createResult } = renderHook(() => LinePlot.useCreate(), {
        wrapper,
      });

      const key = uuid.create();
      await act(async () => {
        await createResult.current.updateAsync({
          key,
          project: project.key,
          name: "stored_plot",
        });
      });

      const { result: retrieveResult } = renderHook(
        () => LinePlot.useRetrieve({ key }),
        { wrapper },
      );
      await waitFor(() => expect(retrieveResult.current.variant).toEqual("success"));
      expect(retrieveResult.current.data?.name).toEqual("stored_plot");
    });
  });

  describe("useRename", () => {
    it("should rename a line plot", async () => {
      const project = await client.projects.create({
        name: "rename_project",
      });
      const plot = await client.lineplots.create(project.key, {
        name: "original_name",
      });

      const { result } = renderHook(
        () => {
          const retrieve = LinePlot.useRetrieve({ key: plot.key });
          const rename = LinePlot.useRename();
          return { retrieve, rename };
        },
        { wrapper },
      );

      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      expect(result.current.retrieve.data?.name).toEqual("original_name");

      await act(async () => {
        await result.current.rename.updateAsync({
          key: plot.key,
          name: "renamed_plot",
        });
      });

      const retrieved = await client.lineplots.retrieve({ key: plot.key });
      expect(retrieved.name).toEqual("renamed_plot");
    });

    it("should update cached plot after rename", async () => {
      const project = await client.projects.create({
        name: "rename_cache_project",
      });
      const plot = await client.lineplots.create(project.key, {
        name: "cache_original",
      });

      const { result } = renderHook(
        () => ({
          retrieve: LinePlot.useRetrieve({ key: plot.key }),
          rename: LinePlot.useRename(),
        }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));

      await act(async () => {
        await result.current.rename.updateAsync({
          key: plot.key,
          name: "cache_renamed",
        });
      });

      await waitFor(() => {
        expect(result.current.retrieve.data?.name).toEqual("cache_renamed");
      });
    });
  });

  describe("useDelete", () => {
    it("should delete a single line plot", async () => {
      const project = await client.projects.create({
        name: "delete_project",
      });
      const plot = await client.lineplots.create(project.key, {
        name: "delete_single",
      });

      const { result } = renderHook(() => LinePlot.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync(plot.key);
      });
      expect(result.current.variant).toEqual("success");
      await expect(client.lineplots.retrieve({ key: plot.key })).rejects.toThrow(
        NotFoundError,
      );
    });

    it("should delete multiple line plots", async () => {
      const project = await client.projects.create({
        name: "delete_multi_project",
      });
      const plot1 = await client.lineplots.create(project.key, {
        name: "delete_multi_1",
      });
      const plot2 = await client.lineplots.create(project.key, {
        name: "delete_multi_2",
      });

      const { result } = renderHook(() => LinePlot.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync([plot1.key, plot2.key]);
      });

      expect(result.current.variant).toEqual("success");

      await expect(client.lineplots.retrieve({ key: plot1.key })).rejects.toThrow(
        NotFoundError,
      );
      await expect(client.lineplots.retrieve({ key: plot2.key })).rejects.toThrow(
        NotFoundError,
      );
    });
  });
});
