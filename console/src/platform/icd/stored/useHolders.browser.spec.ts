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
import { createFirstKey } from "@/platform/icd/testutil";
import { renderHookWithConsole } from "@/testutil";

const client = createTestClient();

// The query and page size of the list that the hook reads.
const LIST = { limit: 50 };

const createMessage = (): icd.Message => ({
  key: uuid.create(),
  name: uniqueName("message"),
  payload: { format: "json", fields: [] },
});

/** @returns an ICD key that the Core lists after every key of createFirstKey. */
const createSecondKey = (): icd.Key => `${uuid.create().slice(0, 19)}8000-000000000001`;

// The hook reads every ICD of the Core, which can hold thousands, one page at a time.
const EVERY_PAGE_TIMEOUT = 45_000;

/** @returns the read properties key of a message of the record oven-1 of plant. */
const storedKey = (message: icd.MessageKey): string => `plant#oven-1#${message}`;

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
  const renderHolders = async (read: Record<string, unknown>) => {
    const fresh = createTestClient();
    onTestFinished(async () => await fresh.close());
    const list = fresh.icds.watch(LIST);
    const more = vi.spyOn(list, "more");
    const rendered = await renderHookWithConsole(() => ICD.Stored.useHolders(read), {
      client: fresh,
    });
    return { ...rendered, fresh, list, more };
  };

  it("should return an ICD past the first page that holds a stored message", async () => {
    const holder = await client.icds.create({
      key: createSecondKey(),
      name: uniqueName("holder"),
      messages: [createMessage()],
    });
    onTestFinished(async () => await client.icds.delete(holder.key));
    const { result, more } = await renderHolders({
      [storedKey(holder.messages[0].key)]: {},
    });
    await waitFor(() => expect(result.current).toEqual([holder.key]));
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
      const { result, rerender, fresh, list, more } = await renderHolders({
        [storedKey(gone.messages[0].key)]: {},
        [storedKey(later.key)]: {},
      });
      await waitFor(
        () => {
          const window = list.getCached();
          expect(window).toBeDefined();
          expect(window?.end).toBeNull();
        },
        { timeout: EVERY_PAGE_TIMEOUT },
      );
      expect(more).toHaveBeenCalled();
      const reads = more.mock.calls.length;
      const holder = await fresh.icds.create({
        name: uniqueName("holder"),
        messages: [later],
      });
      onTestFinished(async () => await client.icds.delete(holder.key));
      await waitFor(() => expect(result.current).toEqual([holder.key]));
      rerender();
      expect(more).toHaveBeenCalledTimes(reads);
    },
    2 * EVERY_PAGE_TIMEOUT,
  );

  it("should read only the first page for a key that names no message", async () => {
    const [first] = [...firstPage].sort((a, b) => compare.uuids(a.key, b.key));
    const { result, rerender, more } = await renderHolders({
      "plant#oven-1": {},
      [storedKey(first.messages[0].key)]: {},
    });
    await waitFor(() => expect(result.current).toEqual([first.key]));
    rerender();
    expect(more).not.toHaveBeenCalled();
  });
});
