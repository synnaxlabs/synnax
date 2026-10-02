// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { $clerkStore, $isLoadedStore, $userStore } from "@clerk/astro/client";
import { useEffect, useState } from "react";

export type Clerk = NonNullable<ReturnType<typeof $clerkStore.get>>;
export type User = NonNullable<ReturnType<typeof $userStore.get>>;
export type Organization = Awaited<ReturnType<Clerk["getOrganization"]>>;
export type Membership = User["organizationMemberships"][number];

interface Store<T> {
  subscribe: (listener: (value: T) => void) => () => void;
}

/**
 * useStore subscribes to a Clerk store. It returns the unloaded value until the
 * subscription delivers the store's value after mount, so hydration matches the server
 * HTML. It never reads the store during render: a read of a computed store can notify
 * subscribers, and React rejects an update scheduled while a component renders.
 */
const useStore = <T>(store: Store<T>, unloaded: T): T => {
  const [value, setValue] = useState(unloaded);
  useEffect(() => store.subscribe(setValue), [store]);
  return value;
};

/** useClerk returns the loaded Clerk client, or null before clerk-js has loaded. */
export const useClerk = (): Clerk | null => {
  const loaded = useStore($isLoadedStore, false);
  const clerk = useStore($clerkStore, null);
  return loaded ? clerk : null;
};

/** useUser returns the logged-in user, null when logged out, undefined when loading. */
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
