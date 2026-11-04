// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type certificate } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Certificate } from "@/certificate";
import { createAsyncSynnaxWrapper, renderHookSuspended } from "@/testutil";

const client = createTestClient();

const createCertificate = async (name: string): Promise<certificate.Certificate> =>
  await client.certificates.create({
    name,
    source: "file",
    certificatePath: "/etc/synnax/client.crt",
  });

describe("queries", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("useList", () => {
    it("should list certificates by key", async () => {
      const first = await createCertificate("list 1");
      const second = await createCertificate("list 2");
      const { result } = renderHook(() => Certificate.useList(), { wrapper });
      await act(async () => {
        await result.current.retrieveAsync({ keys: [first.key, second.key] });
      });
      expect(result.current.data).toEqual([first.key, second.key]);
      expect(result.current.getItem(first.key)).toEqual(first);
    });

    it("should find a certificate by a search term", async () => {
      const name = `${id.create()} broker`;
      const created = await createCertificate(name);
      const { result } = renderHook(() => Certificate.useList(), { wrapper });
      await waitFor(async () => {
        await act(async () => {
          await result.current.retrieveAsync({ searchTerm: name });
        });
        expect(result.current.data).toContain(created.key);
      });
    });

    it("should add a certificate when it is created", async () => {
      const { result } = renderHook(() => Certificate.useList(), { wrapper });
      await act(async () => {
        await result.current.retrieveAsync({});
      });
      const created = await createCertificate("created after list");
      await waitFor(() => expect(result.current.data).toContain(created.key));
    });

    it("should remove a certificate when it is deleted", async () => {
      const created = await createCertificate("deleted after list");
      const { result } = renderHook(() => Certificate.useList(), { wrapper });
      await act(async () => {
        await result.current.retrieveAsync({});
      });
      expect(result.current.data).toContain(created.key);
      await client.certificates.delete(created.key);
      await waitFor(() => expect(result.current.data).not.toContain(created.key));
    });
  });

  describe("use", () => {
    it("should retrieve a certificate by key", async () => {
      const created = await createCertificate("retrieve");
      const { result } = await renderHookSuspended(
        () => Certificate.use({ key: created.key }),
        { wrapper },
      );
      expect(result.current).toEqual(created);
    });

    it("should update when the certificate is renamed", async () => {
      const created = await createCertificate("before rename");
      const { result } = await renderHookSuspended(
        () => Certificate.use({ key: created.key }),
        { wrapper },
      );
      await act(async () => {
        await client.certificates.rename(created.key, "after rename");
      });
      await waitFor(() => expect(result.current.name).toEqual("after rename"));
    });
  });

  describe("useRename", () => {
    it("should rename a certificate", async () => {
      const created = await createCertificate("before rename");
      const { result } = renderHook(() => Certificate.useRename(), { wrapper });
      await act(async () => {
        await result.current.updateAsync({ key: created.key, name: "after rename" });
      });
      expect(result.current.variant).toEqual("success");
      const retrieved = await createTestClient().certificates.retrieve(created.key);
      expect(retrieved.name).toEqual("after rename");
    });
  });

  describe("useDelete", () => {
    it("should delete a certificate", async () => {
      const created = await createCertificate("delete");
      const { result } = renderHook(() => Certificate.useDelete(), { wrapper });
      await act(async () => {
        await result.current.updateAsync(created.key);
      });
      expect(result.current.variant).toEqual("success");
      await expect(
        createTestClient().certificates.retrieve(created.key),
      ).rejects.toThrow(`Certificates with keys [${created.key}] not found`);
    });

    it("should run afterOptimistic before the delete commits", async () => {
      const created = await createCertificate("optimistic delete");
      const order: string[] = [];
      const { result } = renderHook(
        () =>
          Certificate.useDelete({
            afterOptimistic: () => {
              order.push("optimistic");
            },
            afterSuccess: () => {
              order.push("success");
            },
          }),
        { wrapper },
      );
      await act(async () => {
        await result.current.updateAsync(created.key);
      });
      expect(order).toEqual(["optimistic", "success"]);
    });
  });
});
