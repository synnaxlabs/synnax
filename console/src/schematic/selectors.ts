// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, UnexpectedError } from "@synnaxlabs/client";
import type { Control, Diagram, Viewport } from "@synnaxlabs/pluto";

import { useMemoSelect } from "@/hooks";
import {
  type ElementConfig,
  SLICE_NAME,
  type SliceState,
  type State,
  type StoreState,
  type ToolbarState,
} from "@/schematic/slice";

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const selectRequired = (state: StoreState, key: string): State => {
  const schematic = selectSliceState(state).schematics[key];
  if (schematic == null)
    throw new UnexpectedError(`Schematic not found for key: ${key}`);
  return schematic;
};

export const useSelectRequired = (key: string): State =>
  useMemoSelect((state: StoreState) => selectRequired(state, key), [key]);

export const selectOptional = (state: StoreState, key: string): State | undefined =>
  selectSliceState(state).schematics[key];

export const useSelectOptional = (key: string): State | undefined =>
  useMemoSelect((state: StoreState) => selectOptional(state, key), [key]);

export const selectRequiredMany = (state: StoreState, keys: string[]): State[] =>
  keys.map((key) => selectRequired(state, key));

export const useSelectRequiredMany = (keys: string[]): State[] =>
  useMemoSelect((state: StoreState) => selectRequiredMany(state, keys), [keys]);

export const selectSelectedElementsConfigs = (
  state: StoreState,
  layoutKey: string,
): [string[], ElementConfig[]] => {
  const schematic = selectRequired(state, layoutKey);
  return [schematic.selected, schematic.selected.map((key) => schematic.configs[key])];
};

export const useSelectSelectedElementsConfigs = (
  layoutKey: string,
): [string[], ElementConfig[]] =>
  useMemoSelect(
    (state: StoreState) => selectSelectedElementsConfigs(state, layoutKey),
    [layoutKey],
  );

export const selectEdge = (
  state: StoreState,
  layoutKey: string,
  key: string,
): Diagram.Edge | undefined =>
  selectOptional(state, layoutKey)?.edges.find((edge) => edge.key === key);

export const useSelectEdge = (
  layoutKey: string,
  key: string,
): Diagram.Edge | undefined =>
  useMemoSelect(
    (state: StoreState) => selectEdge(state, layoutKey, key),
    [layoutKey, key],
  );
export const selectNode = (
  state: StoreState,
  layoutKey: string,
  key: string,
): Diagram.Node | undefined =>
  selectOptional(state, layoutKey)?.nodes.find((n) => n.key === key);

export const useSelectNode = (
  layoutKey: string,
  key: string,
): Diagram.Node | undefined =>
  useMemoSelect(
    (state: StoreState) => selectNode(state, layoutKey, key),
    [layoutKey, key],
  );

export const selectSelectedElementNames = (
  state: StoreState,
  layoutKey: string,
): (string | null)[] => {
  const [, configs] = selectSelectedElementsConfigs(state, layoutKey);
  return configs.map((el) => {
    if ("label" in el) return el.label?.label ?? null;
    return null;
  });
};

export const useSelectSelectedElementNames = (layoutKey: string): (string | null)[] =>
  useMemoSelect(
    (s: StoreState) => selectSelectedElementNames(s, layoutKey),
    [layoutKey],
  );

export const selectConfig = (
  state: StoreState,
  layoutKey: string,
  elKey: string,
): ElementConfig | undefined => selectRequired(state, layoutKey).configs[elKey];

export const useSelectConfig = (
  layoutKey: string,
  elKey: string,
): ElementConfig | undefined =>
  useMemoSelect(
    (state: StoreState) => selectConfig(state, layoutKey, elKey),
    [layoutKey, elKey],
  );

export const selectRequiredConfig = (
  state: StoreState,
  layoutKey: string,
  key: string,
): ElementConfig => {
  const config = selectConfig(state, layoutKey, key);
  if (config == null) throw new NotFoundError(`Node props not found for key: ${key}`);
  return config;
};

export const useSelectRequiredConfig = (
  layoutKey: string,
  key: string,
): ElementConfig =>
  useMemoSelect(
    (state: StoreState) => selectRequiredConfig(state, layoutKey, key),
    [layoutKey, key],
  );

export const selectToolbar = (
  state: StoreState,
  key: string,
): ToolbarState | undefined => selectOptional(state, key)?.toolbar;

export const useSelectToolbar = (key: string): ToolbarState | undefined =>
  useMemoSelect((state: StoreState) => selectToolbar(state, key), [key]);

export const selectEditable = (state: StoreState, key: string): boolean | undefined =>
  selectOptional(state, key)?.editable;

export const useSelectEditable = (key: string): boolean | undefined =>
  useMemoSelect((state: StoreState) => selectEditable(state, key), [key]);

export const selectRequiredViewportMode = (
  state: StoreState,
  key: string,
): Viewport.Mode => selectRequired(state, key).mode;

export const useSelectRequiredViewportMode = (key: string): Viewport.Mode =>
  useMemoSelect((state: StoreState) => selectRequiredViewportMode(state, key), [key]);

export const selectViewport = (
  state: StoreState,
  key: string,
): Diagram.Viewport | undefined => selectOptional(state, key)?.viewport;

export const useSelectViewport = (key: string): Diagram.Viewport | undefined =>
  useMemoSelect((state: StoreState) => selectViewport(state, key), [key]);

export const selectControlStatus = (
  state: StoreState,
  layoutKey: string,
): Control.Status | undefined => selectOptional(state, layoutKey)?.control;

export const useSelectControlStatus = (layoutKey: string): Control.Status | undefined =>
  useMemoSelect(
    (state: StoreState) => selectControlStatus(state, layoutKey),
    [layoutKey],
  );

export const selectVersion = (state: StoreState, key: string): string | undefined =>
  selectOptional(state, key)?.version;

export const useSelectVersion = (key: string): string | undefined =>
  useMemoSelect((state: StoreState) => selectVersion(state, key), [key]);

export const selectIsSnapshot = (state: StoreState, key: string): boolean | undefined =>
  selectOptional(state, key)?.snapshot;

export const useSelectIsSnapshot = (key: string): boolean | undefined =>
  useMemoSelect((state: StoreState) => selectIsSnapshot(state, key), [key]);

export const selectIsRemoteCreated = (
  state: StoreState,
  key: string,
): boolean | undefined => selectOptional(state, key)?.remoteCreated;

export const useSelectIsRemoteCreated = (key: string): boolean | undefined =>
  useMemoSelect((state: StoreState) => selectIsRemoteCreated(state, key), [key]);

export const selectAuthority = (state: StoreState, key: string): number | undefined =>
  selectOptional(state, key)?.authority;

export const useSelectAuthority = (key: string): number | undefined =>
  useMemoSelect((state: StoreState) => selectAuthority(state, key), [key]);

export const selectSelected = (state: StoreState, key: string): string[] =>
  selectOptional(state, key)?.selected ?? [];

export const useSelectSelected = (key: string): string[] =>
  useMemoSelect((state: StoreState) => selectSelected(state, key), [key]);

export const selectSelectedSymbolGroup = (state: StoreState, key: string): string =>
  selectRequired(state, key).toolbar.selectedSymbolGroup;

export const useSelectSelectedSymbolGroup = (key: string): string =>
  useMemoSelect((state: StoreState) => selectSelectedSymbolGroup(state, key), [key]);

export const selectLegendVisible = (
  state: StoreState,
  key: string,
): boolean | undefined => selectOptional(state, key)?.legend.visible;

export const useSelectLegendVisible = (key: string): boolean | undefined =>
  useMemoSelect((state: StoreState) => selectLegendVisible(state, key), [key]);
