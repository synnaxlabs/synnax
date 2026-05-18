// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import { useState } from "react";

import { Flux } from "@/flux";
import { useAsyncEffect } from "@/hooks/useAsyncEffect";
import { useMemoDeepEqual } from "@/memo";
import { type FluxSubStore } from "@/schematic/symbol/queries";
import { Synnax } from "@/synnax";

interface CustomConfig {
  variant: "customStatic" | "customActuator";
  specKey: string;
}

const asCustomConfig = (cfg: unknown): CustomConfig | null => {
  if (typeof cfg !== "object" || cfg == null) return null;
  const c = cfg as { variant?: unknown; specKey?: unknown };
  if (
    (c.variant === "customStatic" || c.variant === "customActuator") &&
    typeof c.specKey === "string" &&
    c.specKey.length > 0
  )
    return { variant: c.variant, specKey: c.specKey };
  return null;
};

export interface CustomSymbolRefs {
  /// Unique custom-symbol keys referenced by nodes in the schematic.
  specKeys: string[];
  /// Map from a custom-symbol key to the list of node keys that reference it.
  /// Used by the bulk-resolve dialog to report how many nodes a re-link affects.
  nodesByKey: Map<string, string[]>;
}

/// collectCustomSymbolRefs walks an arbitrary configs map (as stored on a
/// schematic) and aggregates the unique custom-symbol references plus a reverse
/// map of which node keys carry each reference. Pure — safe to call inside a
/// selector.
export const collectCustomSymbolRefs = (
  configs: Record<string, unknown> | undefined,
): CustomSymbolRefs => {
  const nodesByKey = new Map<string, string[]>();
  if (configs == null) return { specKeys: [], nodesByKey };
  for (const [nodeKey, cfg] of Object.entries(configs)) {
    const custom = asCustomConfig(cfg);
    if (custom == null) continue;
    const existing = nodesByKey.get(custom.specKey);
    if (existing == null) nodesByKey.set(custom.specKey, [nodeKey]);
    else existing.push(nodeKey);
  }
  return { specKeys: [...nodesByKey.keys()], nodesByKey };
};

export interface AuditResult {
  /// Symbol keys that the cluster reports as not found. Drives the banner and the
  /// bulk-resolve dialog.
  missing: string[];
  /// Reverse map: missing symbol key -> node keys that reference it. Lets the
  /// bulk-resolve dialog show "N nodes will be re-linked" per row.
  nodesByMissingKey: Map<string, string[]>;
}

export interface UseAuditCustomSymbolsParams {
  /// All unique custom-symbol keys referenced by the schematic. Use
  /// {@link collectCustomSymbolRefs} on the schematic's configs map.
  refs: CustomSymbolRefs;
}

/// useAuditCustomSymbols drives a NotFound-tolerant fetch for each unique custom
/// symbol referenced by the schematic and produces the set of missing keys. The
/// schematicSymbols store is populated as a side effect of successful fetches —
/// nothing extra to wire for auto-heal. The audit re-runs when the set of unique
/// referenced keys changes.
export const useAuditCustomSymbols = ({
  refs,
}: UseAuditCustomSymbolsParams): AuditResult => {
  const client = Synnax.use();
  const store = Flux.useStore<FluxSubStore>();
  const memoKeys = useMemoDeepEqual(refs.specKeys);
  const [missing, setMissing] = useState<string[]>([]);
  useAsyncEffect(
    async (signal) => {
      if (client == null || memoKeys.length === 0) {
        setMissing([]);
        return;
      }
      const next: string[] = [];
      await Promise.all(
        memoKeys.map(async (specKey) => {
          if (store.schematicSymbols.get(specKey) != null) return;
          try {
            const symbol = await client.schematics.symbols.retrieve({
              key: specKey,
            });
            if (signal.aborted) return;
            store.schematicSymbols.set(specKey, symbol);
          } catch (e) {
            if (NotFoundError.matches(e)) next.push(specKey);
          }
        }),
      );
      if (signal.aborted) return;
      setMissing(next);
    },
    [memoKeys, client, store],
  );
  return { missing, nodesByMissingKey: refs.nodesByKey };
};
