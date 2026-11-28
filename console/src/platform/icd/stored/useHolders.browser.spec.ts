// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { icd } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { compare, uuid } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { waitFor } from "@testing-library/react";
import { afterAll, beforeAll, describe, expect, it, onTestFinished, vi } from "vitest";

import { ICD } from "@/platform/icd";
import { createFirstKey, createSecondKey } from "@/platform/icd/testutil";
import { renderHookWithConsole } from "@/testutil";

const client = createTestClient();

// The query and page size of the list that the hook reads.
const LIST = { limit: 50 };

const createMessage = (): icd.Message => ({
  key: uuid.create(),
  name: uniqueName("message"),
  payload: { format: "json", fields: [] },
});

// The hook reads every ICD of the Core, which can hold thousands, one page at a time.
const EVERY_PAGE_TIMEOUT = 45_000;

describe("useHolders", () => {
  // A full first page of ICDs that the Core lists before random ones.
  let firstPage: icd.ICD[] = [];
  beforeAll(async () => {
    firstPage = await client.icds.create(
      Array.from({ length: LIST.limit }, () => ({
        key: createFirstKey(),
        name: uniqueName("first"),
        messages: [createMessage()],
      })),
    );
  });
  afterAll(async () => await client.icds.delete(firstPage.map(({ key }) => key)));

  /** Renders the hook on a new client, whose list of ICDs starts empty. */
  const renderHolders = async (messages: icd.MessageKey[]) => {
    const fresh = createTestClient();
    onTestFinished(async () => await fresh.close());
    const list = fresh.icds.watch(LIST);
    const more = vi.spyOn(list, "more");
    const rendered = await renderHookWithConsole(
      () => ICD.Stored.useHolders(messages),
      { client: fresh },
    );
    return { ...rendered, fresh, list, more };
  };

  /** @returns the keys of the ICDs that the hook found. */
  const keysOf = ({ icds }: ICD.Stored.Holders): icd.Key[] =>
    icds.map(({ key }) => key);

  it("should return an ICD past the first page that holds a message", async () => {
    const holder = await client.icds.create({
      key: createSecondKey(),
      name: uniqueName("holder"),
      messages: [createMessage()],
    });
    onTestFinished(async () => await client.icds.delete(holder.key));
    const { result, more } = await renderHolders([holder.messages[0].key]);
    expect(result.current.searching).toBe(true);
    await waitFor(() => expect(keysOf(result.current)).toEqual([holder.key]));
    expect(result.current.searching).toBe(false);
    expect(more).toHaveBeenCalled();
  });

  it(
    "should read no more ICDs once it has read every ICD",
    async () => {
      const gone = await client.icds.create({
        name: uniqueName("gone"),
        messages: [createMessage()],
      });
      await client.icds.delete(gone.key);
      const later = createMessage();
      const { result, rerender, fresh, list, more } = await renderHolders([
        gone.messages[0].key,
        later.key,
      ]);
      await waitFor(
        () => {
          const window = list.getCached();
          expect(window).toBeDefined();
          expect(window?.end).toBeNull();
        },
        { timeout: EVERY_PAGE_TIMEOUT },
      );
      expect(more).toHaveBeenCalled();
      await waitFor(() => expect(result.current.searching).toBe(false));
      const reads = more.mock.calls.length;
      const holder = await fresh.icds.create({
        name: uniqueName("holder"),
        messages: [later],
      });
      onTestFinished(async () => await client.icds.delete(holder.key));
      await waitFor(() => expect(keysOf(result.current)).toEqual([holder.key]));
      rerender();
      expect(more).toHaveBeenCalledTimes(reads);
    },
    2 * EVERY_PAGE_TIMEOUT,
  );

  it("should read only the first page when it holds each message", async () => {
    const [first] = [...firstPage].sort((a, b) => compare.uuids(a.key, b.key));
    const { result, rerender, more } = await renderHolders([first.messages[0].key]);
    await waitFor(() => expect(keysOf(result.current)).toEqual([first.key]));
    expect(result.current.searching).toBe(false);
    rerender();
    expect(more).not.toHaveBeenCalled();
  });

  it("should read only the first page for no messages", async () => {
    const { result, rerender, list, more } = await renderHolders([]);
    await waitFor(() => expect(list.getCached()).toBeDefined());
    rerender();
    expect(result.current).toEqual({ icds: [], searching: false });
    expect(more).not.toHaveBeenCalled();
  });
});
