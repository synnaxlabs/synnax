// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type secret } from "@synnaxlabs/client";
import { CERTIFICATE, createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { assert, beforeAll, describe, expect, it } from "vitest";

import { Errors } from "@/errors";
import { Flux } from "@/flux";
import { Secret } from "@/secret";
import { createAsyncSynnaxWrapper, renderHookSuspended } from "@/testutil";

const client = createTestClient();

const createSecret = async (name: string): Promise<secret.Secret> =>
  await client.secrets.set({ name, source: "stored", value: "hunter2" });

describe("queries", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("useList", () => {
    it("should list secrets by name and move a renamed one", async () => {
      const prefix = id.create();
      const b = await createSecret(`${prefix} b`);
      const a = await createSecret(`${prefix} a`);
      const { result } = renderHook(
        () => Secret.useList({ query: { key: [b.key, a.key] } }),
        { wrapper },
      );
      act(() => result.current.more());
      await waitFor(() => expect(result.current.data).toEqual([a.key, b.key]));
      expect(result.current.getItem(a.key)).toEqual(a);
      await client.secrets.rename(a.key, `${prefix} c`);
      await waitFor(() => expect(result.current.data).toEqual([b.key, a.key]));
    });

    it("should find a secret by a search term", async () => {
      const name = `${id.create()} password`;
      const created = await createSecret(name);
      const { result } = renderHook(() => Secret.useList(), { wrapper });
      act(() => result.current.search(name));
      await waitFor(() => expect(result.current.data).toContain(created.key));
    });

    it("should add a secret that matches a search when it is created", async () => {
      const prefix = id.create();
      const { result } = renderHook(() => Secret.useList(), { wrapper });
      act(() => result.current.search(prefix));
      await waitFor(() => expect(result.current.answered).toBe(true));
      const created = await createSecret(`${prefix} created`);
      await waitFor(() => expect(result.current.data).toContain(created.key));
    });

    it("should remove a secret when it is deleted", async () => {
      const created = await createSecret("deleted after list");
      const { result } = renderHook(
        () => Secret.useList({ query: { key: [created.key] } }),
        { wrapper },
      );
      act(() => result.current.more());
      await waitFor(() => expect(result.current.data).toEqual([created.key]));
      await client.secrets.delete(created.key);
      await waitFor(() => expect(result.current.data).toEqual([]));
    });
  });

  describe("use", () => {
    it("should retrieve a secret by key without its uses", async () => {
      const created = await createSecret("retrieve");
      const { result } = await renderHookSuspended(
        () => Secret.use({ key: created.key }),
        { wrapper },
      );
      expect(result.current).toEqual(created);
      expect(result.current.uses).toBeUndefined();
    });

    it("should retrieve the resources that use a secret", async () => {
      const cert = await client.certificates.set({
        name: "Uses certificate",
        source: "stored",
        pem: CERTIFICATE.pem,
        privateKeyPem: CERTIFICATE.privateKeyPem,
      });
      assert(cert.source === "stored" && cert.privateKey != null);
      const { privateKey } = cert;
      const { result } = await renderHookSuspended(
        () => Secret.use({ key: privateKey, includeUses: true }),
        { wrapper },
      );
      expect(result.current.uses).toEqual([{ type: "certificate", key: cert.key }]);
    });

    it("should update a secret read with its uses when it is renamed", async () => {
      const created = await createSecret("before rename");
      const { result } = await renderHookSuspended(
        () => Secret.use({ key: created.key, includeUses: true }),
        { wrapper },
      );
      expect(result.current.uses).toEqual([]);
      await act(async () => {
        await client.secrets.rename(created.key, "after rename");
      });
      await waitFor(() => expect(result.current.name).toEqual("after rename"));
      expect(result.current.uses).toEqual([]);
    });

    it("should throw a DeletedError when a secret read with its uses is deleted", async () => {
      const created = await createSecret("deleted with uses");
      const caught: Error[] = [];
      const Fallback = ({ error }: Errors.FallbackProps): null => {
        caught.push(error);
        return null;
      };
      const Wrapper = wrapper;
      const BoundaryWrapper = ({ children }: PropsWithChildren): ReactElement => (
        <Wrapper>
          <Errors.SuspenseBoundary loading={null} FallbackComponent={Fallback}>
            {children}
          </Errors.SuspenseBoundary>
        </Wrapper>
      );
      const { result } = await renderHookSuspended(
        () => Secret.use({ key: created.key, includeUses: true }),
        { wrapper: BoundaryWrapper, onCaughtError: () => {} },
      );
      expect(result.current.uses).toEqual([]);
      await act(async () => {
        await client.secrets.delete(created.key);
      });
      await waitFor(() => expect(caught.length).toBeGreaterThan(0));
      const error = caught[0];
      assert(Flux.DeletedError.matches(error));
      expect(error.corpseName).toEqual("deleted with uses");
    });
  });

  describe("useMultiple", () => {
    it("should retrieve the uses of several secrets in one read", async () => {
      const cert = await client.certificates.set({
        name: "Multiple uses certificate",
        source: "stored",
        pem: CERTIFICATE.pem,
        privateKeyPem: CERTIFICATE.privateKeyPem,
      });
      assert(cert.source === "stored" && cert.privateKey != null);
      const unused = await createSecret("multiple unused");
      const keys = [cert.privateKey, unused.key];
      const { result } = await renderHookSuspended(
        () => Secret.useMultiple({ key: keys, includeUses: true }),
        { wrapper },
      );
      const uses = new Map(result.current.map(({ key, uses }) => [key, uses]));
      expect(uses).toEqual(
        new Map([
          [cert.privateKey, [{ type: "certificate", key: cert.key }]],
          [unused.key, []],
        ]),
      );
    });

    it("should keep the uses of a secret when it is renamed", async () => {
      const first = await createSecret("multiple before rename");
      const second = await createSecret("multiple untouched");
      const { result } = await renderHookSuspended(
        () => Secret.useMultiple({ key: [first.key, second.key], includeUses: true }),
        { wrapper },
      );
      await act(async () => {
        await client.secrets.rename(first.key, "multiple after rename");
      });
      await waitFor(() =>
        expect(result.current.map(({ name }) => name).sort()).toEqual([
          "multiple after rename",
          "multiple untouched",
        ]),
      );
      expect(result.current.map(({ uses }) => uses)).toEqual([[], []]);
    });

    it("should drop a secret from the read when it is deleted", async () => {
      const kept = await createSecret("multiple kept");
      const deleted = await createSecret("multiple deleted");
      const { result } = await renderHookSuspended(
        () => Secret.useMultiple({ key: [kept.key, deleted.key], includeUses: true }),
        { wrapper },
      );
      await act(async () => {
        await client.secrets.delete(deleted.key);
      });
      await waitFor(() =>
        expect(result.current.map(({ key }) => key)).toEqual([kept.key]),
      );
    });
  });

  describe("useMultiple without keys", () => {
    it("should read the uses of every secret and track additions and deletions", async () => {
      const cert = await client.certificates.set({
        name: "All uses certificate",
        source: "stored",
        pem: CERTIFICATE.pem,
        privateKeyPem: CERTIFICATE.privateKeyPem,
      });
      assert(cert.source === "stored" && cert.privateKey != null);
      const deleted = await createSecret("all uses deleted");
      const { result } = await renderHookSuspended(
        () => Secret.useMultiple({ includeUses: true }),
        { wrapper },
      );
      const usesOf = (key: string) => result.current.find((s) => s.key === key)?.uses;
      expect(usesOf(cert.privateKey)).toEqual([{ type: "certificate", key: cert.key }]);
      expect(usesOf(deleted.key)).toEqual([]);
      const added = await act(async () => {
        await client.secrets.delete(deleted.key);
        return await createSecret("all uses added");
      });
      await waitFor(() => {
        expect(usesOf(deleted.key)).toBeUndefined();
        expect(usesOf(added.key)).toEqual([]);
      });
    });
  });

  describe("useCreate", () => {
    it("should create a stored secret", async () => {
      let created: secret.Secret | undefined;
      const { result } = renderHook(
        () =>
          Secret.useCreate({
            afterSuccess: ({ data }) => {
              created = data;
            },
          }),
        { wrapper },
      );
      await act(async () => {
        await result.current.updateAsync({
          name: "new secret",
          source: "stored",
          value: "hunter2",
        });
      });
      assert(created != null);
      expect(created).toMatchObject({ name: "new secret", source: "stored" });
      expect(await client.secrets.retrieve(created.key)).toEqual(created);
    });

    it("should replace the value of an existing secret", async () => {
      const original = await createSecret("replace");
      let replaced: secret.Secret | undefined;
      const { result } = renderHook(
        () =>
          Secret.useCreate({
            afterSuccess: ({ data }) => {
              replaced = data;
            },
          }),
        { wrapper },
      );
      await act(async () => {
        await result.current.updateAsync({
          key: original.key,
          name: original.name,
          source: "stored",
          value: "correct horse",
        });
      });
      assert(replaced != null);
      expect(replaced.key).toEqual(original.key);
      expect(replaced.updated.after(original.updated)).toBe(true);
    });
  });

  describe("useRename", () => {
    it("should rename a secret", async () => {
      const created = await createSecret("before rename");
      const { result } = renderHook(() => Secret.useRename(), { wrapper });
      await act(async () => {
        await result.current.updateAsync({ key: created.key, name: "after rename" });
      });
      expect(result.current.variant).toEqual("success");
      const retrieved = await createTestClient().secrets.retrieve(created.key);
      expect(retrieved.name).toEqual("after rename");
    });
  });

  describe("useDelete", () => {
    it("should delete a secret", async () => {
      const created = await createSecret("delete");
      const { result } = renderHook(() => Secret.useDelete(), { wrapper });
      await act(async () => {
        await result.current.updateAsync(created.key);
      });
      expect(result.current.variant).toEqual("success");
      await expect(createTestClient().secrets.retrieve(created.key)).rejects.toThrow(
        `secret with key ${created.key} not found`,
      );
    });

    it("should run afterOptimistic before the delete commits", async () => {
      const created = await createSecret("optimistic delete");
      const order: string[] = [];
      const { result } = renderHook(
        () =>
          Secret.useDelete({
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
