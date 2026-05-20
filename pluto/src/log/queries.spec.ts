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

import { Log } from "@/log";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

describe("log queries", () => {
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("useRetrieve", () => {
    it("should retrieve a log by key", async () => {
      const project = await client.projects.create({
        name: "test_project",
      });
      const log = await client.logs.create(project.key, {
        name: "retrieve_test",
        data: {},
      });

      const { result } = renderHook(() => Log.useRetrieve({ key: log.key }), {
        wrapper,
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.data?.key).toEqual(log.key);
      expect(result.current.data?.name).toEqual("retrieve_test");
    });

    it("should cache retrieved logs", async () => {
      const project = await client.projects.create({
        name: "cache_project",
      });
      const log = await client.logs.create(project.key, {
        name: "cached_log",
        data: {},
      });

      const { result: result1 } = renderHook(() => Log.useRetrieve({ key: log.key }), {
        wrapper,
      });
      await waitFor(() => expect(result1.current.variant).toEqual("success"));

      const { result: result2 } = renderHook(() => Log.useRetrieve({ key: log.key }), {
        wrapper,
      });
      await waitFor(() => expect(result2.current.variant).toEqual("success"));
      expect(result2.current.data).toEqual(result1.current.data);
    });
  });

  describe("useCreate", () => {
    it("should create a new log", async () => {
      const project = await client.projects.create({
        name: "create_project",
      });

      const { result } = renderHook(() => Log.useCreate(), { wrapper });

      const key = uuid.create();
      await act(async () => {
        await result.current.updateAsync({
          key,
          project: project.key,
          name: "created_log",
          data: {},
        });
      });

      expect(result.current.variant).toEqual("success");
      expect(result.current.data?.name).toEqual("created_log");
      expect(result.current.data?.project).toEqual(project.key);

      const retrieved = await client.logs.retrieve({ key });
      expect(retrieved.name).toEqual("created_log");
    });

    it("should store created log in flux store", async () => {
      const project = await client.projects.create({
        name: "store_project",
      });

      const { result: createResult } = renderHook(() => Log.useCreate(), {
        wrapper,
      });

      const key = uuid.create();
      await act(async () => {
        await createResult.current.updateAsync({
          key,
          project: project.key,
          name: "stored_log",
          data: {},
        });
      });

      const { result: retrieveResult } = renderHook(() => Log.useRetrieve({ key }), {
        wrapper,
      });
      await waitFor(() => expect(retrieveResult.current.variant).toEqual("success"));
      expect(retrieveResult.current.data?.name).toEqual("stored_log");
    });
  });

  describe("useRename", () => {
    it("should rename a log", async () => {
      const project = await client.projects.create({
        name: "rename_project",
      });
      const log = await client.logs.create(project.key, {
        name: "original_name",
        data: {},
      });

      const { result } = renderHook(
        () => {
          const retrieve = Log.useRetrieve({ key: log.key });
          const rename = Log.useRename();
          return { retrieve, rename };
        },
        { wrapper },
      );

      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      expect(result.current.retrieve.data?.name).toEqual("original_name");

      await act(async () => {
        await result.current.rename.updateAsync({
          key: log.key,
          name: "renamed_log",
        });
      });

      const retrieved = await client.logs.retrieve({ key: log.key });
      expect(retrieved.name).toEqual("renamed_log");
    });

    it("should update cached log after rename", async () => {
      const project = await client.projects.create({
        name: "rename_cache_project",
      });
      const log = await client.logs.create(project.key, {
        name: "cache_original",
        data: {},
      });

      const { result } = renderHook(
        () => ({
          retrieve: Log.useRetrieve({ key: log.key }),
          rename: Log.useRename(),
        }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));

      await act(async () => {
        await result.current.rename.updateAsync({
          key: log.key,
          name: "cache_renamed",
        });
      });

      await waitFor(() => {
        expect(result.current.retrieve.data?.name).toEqual("cache_renamed");
      });
    });
  });

  describe("useDelete", () => {
    it("should delete a single log", async () => {
      const project = await client.projects.create({
        name: "delete_project",
      });
      const log = await client.logs.create(project.key, {
        name: "delete_single",
        data: {},
      });

      const { result } = renderHook(() => Log.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync(log.key);
      });
      expect(result.current.variant).toEqual("success");
      await expect(client.logs.retrieve({ key: log.key })).rejects.toThrow(
        NotFoundError,
      );
    });

    it("should delete multiple logs", async () => {
      const project = await client.projects.create({
        name: "delete_multi_project",
      });
      const log1 = await client.logs.create(project.key, {
        name: "delete_multi_1",
        data: {},
      });
      const log2 = await client.logs.create(project.key, {
        name: "delete_multi_2",
        data: {},
      });

      const { result } = renderHook(() => Log.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync([log1.key, log2.key]);
      });

      expect(result.current.variant).toEqual("success");

      await expect(client.logs.retrieve({ key: log1.key })).rejects.toThrow(
        NotFoundError,
      );
      await expect(client.logs.retrieve({ key: log2.key })).rejects.toThrow(
        NotFoundError,
      );
    });
  });
});
