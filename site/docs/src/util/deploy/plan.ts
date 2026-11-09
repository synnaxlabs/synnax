// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NEXT, resolve } from "@/util/version";

/** Requests the newest train. */
export const LATEST = "latest";
/** Requests the train before the newest one. */
export const PREVIOUS = "previous";

/** A deployment of one docs version. */
export interface Deployment {
  /** A train such as `0.61`, or `next`. */
  version: string;
  /** The Git ref the deployment builds. */
  ref: string;
  /** The host the deployment serves, such as `v0-60.docs.synnaxlabs.com`. */
  host: string;
}

export interface Options {
  /** Builds this ref instead. It needs exactly one version. */
  ref?: string;
  /** A train that a release changed. It deploys only if it has a site. */
  released?: string;
}

export interface Lookups {
  /** Lists the trains with a docs site, newest first. */
  trains: () => Promise<string[]>;
  /** Returns the newest stable Core release on a train, such as `0.60.1`. */
  newest: (train: string) => Promise<string>;
  /** Returns whether a train has a `release/core-X.Y` branch. */
  branched: (train: string) => Promise<boolean>;
}

/**
 * Plans the deployments of docs versions. `next` builds from `main`. A train builds
 * from its release branch, which carries its docs fixes, else from its newest stable
 * Core tag.
 * @param requested - Trains such as `0.61`, `next`, `latest`, or `previous`. `previous`
 * adds nothing while only one train exists.
 * @throws {Error} if no train has a site, if a version is neither `next` nor a train
 * with a site, or if `ref` comes with more than one version.
 */
export const plan = async (
  requested: string[],
  { ref, released }: Options,
  { trains: listTrains, newest, branched }: Lookups,
): Promise<Deployment[]> => {
  const trains = await listTrains();
  if (trains.length === 0) throw new Error("No train has a docs site");
  const expand = (name: string): string[] => {
    if (name === LATEST) return trains.slice(0, 1);
    if (name === PREVIOUS) return trains.slice(1, 2);
    return [name];
  };
  const names = [...new Set(requested.flatMap(expand))];
  if (released != null && trains.includes(released) && !names.includes(released))
    names.push(released);
  if (ref != null && names.length !== 1)
    throw new Error(`A ref builds exactly one version, not ${names.length}`);
  const refOf = async (name: string): Promise<string> => {
    if (name === NEXT) return "main";
    if (await branched(name)) return `release/core-${name}`;
    return `core/v${await newest(name)}`;
  };
  return await Promise.all(
    names.map(async (name) => {
      const { url } = resolve(name, trains);
      return {
        version: name,
        ref: ref ?? (await refOf(name)),
        host: new URL(url).host,
      };
    }),
  );
};
