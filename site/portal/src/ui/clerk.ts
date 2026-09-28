// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { $clerkStore, $isLoadedStore, $userStore } from "@clerk/astro/client";
import { useSyncExternalStore } from "react";

export type Clerk = NonNullable<ReturnType<typeof $clerkStore.get>>;
export type User = NonNullable<ReturnType<typeof $userStore.get>>;
export type Organization = Awaited<ReturnType<Clerk["getOrganization"]>>;
export type Membership = User["organizationMemberships"][number];

interface Store<T> {
  get: () => T;
  subscribe: (listener: (value: T) => void) => () => void;
}

/**
 * useStore subscribes to a Clerk store. The server snapshot is the store's unloaded
 * value, so hydration matches the server HTML and the loaded value applies right
 * after.
 */
const useStore = <T>(store: Store<T>, unloaded: T): T =>
  useSyncExternalStore(store.subscribe, store.get, () => unloaded);

/** useClerk returns the loaded Clerk client, or null before clerk-js has loaded. */
export const useClerk = (): Clerk | null => {
  const loaded = useStore($isLoadedStore, false);
  const clerk = useStore($clerkStore, null);
  return loaded ? clerk : null;
};

/** useUser returns the signed-in user, null when signed out, undefined while loading. */
export const useUser = (): User | null | undefined => useStore($userStore, undefined);

/** errorMessage reads the message to show for a failed Clerk call. */
export const errorMessage = (err: unknown): string => {
  if (err != null && typeof err === "object" && "errors" in err) {
    const errors = (err as { errors?: { longMessage?: string; message?: string }[] })
      .errors;
    const first = errors?.[0];
    if (first != null) return first.longMessage ?? first.message ?? "Request failed";
  }
  return err instanceof Error ? err.message : String(err);
};

/** ADMIN_ROLE is Clerk's role for members who manage a team. */
export const ADMIN_ROLE = "org:admin";
export const MEMBER_ROLE = "org:member";
