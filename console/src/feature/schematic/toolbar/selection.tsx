// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Schematic } from "@synnaxlabs/pluto";
import type { ReactElement } from "react";

import { Symbol } from "@/feature/schematic/symbol";
import { Session } from "@/session";

// Group boxes come only from grouping, and a locked group shields its members.
export const useChangeableSymbols = (
  selected: string[],
): Map<string, Schematic.Node.Config> => {
  const configs = Schematic.useConfigs({ keys: selected });
  const parentOf = Schematic.useParentOf();
  const shielded = Schematic.Group.shielded(
    [...configs.keys()],
    parentOf,
    Object.fromEntries(configs),
  );
  const out = new Map<string, Schematic.Node.Config>();
  configs.forEach((config, key) => {
    if (
      Schematic.Node.isConfig(config) &&
      config.variant !== Schematic.Node.GroupBox.VARIANT &&
      !shielded.has(key)
    )
      out.set(key, config);
  });
  return out;
};

export interface SymbolName {
  /** The symbol that every config draws, if they all draw the same one. */
  symbol?: string;
  /** The key of that symbol when the user made it, so that they can edit it. */
  custom?: string;
  name: string;
}

export const useSymbolName = (
  configs: Map<string, Schematic.Node.Config>,
): SymbolName => {
  const symbolKeys = new Set(Array.from(configs.values(), Schematic.Node.symbolKey));
  const shared = symbolKeys.size === 1 ? configs.values().next().value : undefined;
  const ref = shared?.variant === "symbol" ? shared.spec : undefined;
  const specKey = ref?.type === "custom" ? ref.key : null;
  const { symbol: custom } = Schematic.Symbol.useResolved(specKey);
  if (shared == null) return { name: "Mixed" };
  const symbol = Schematic.Node.symbolKey(shared);
  if (specKey != null) return { symbol, custom: specKey, name: custom?.name ?? "" };
  if (ref != null) {
    const name = ref.type === "builtin" ? Schematic.Node.builtInLabel(ref.name) : null;
    return { symbol, name: name ?? "Missing symbol" };
  }
  const spec = Schematic.Node.resolveSpec(symbol);
  return { symbol, name: spec.label ?? spec.name };
};

/**
 * The canvas menu items for the selected symbols: Change symbol, and Edit symbol when
 * they all draw one symbol that the user made. Renders nothing when no selected node
 * can change its symbol.
 */
export const SymbolMenuItems = (): ReactElement | null => {
  const key = Schematic.useKey();
  const dispatch = Session.useDispatch();
  const selected = Session.Schematic.useSelectSelected();
  const changeable = useChangeableSymbols(selected);
  const { custom } = useSymbolName(changeable);
  const openEditor = Symbol.Edit.useModal();
  if (changeable.size === 0) return null;
  const handleChange = () => {
    dispatch(Session.Schematic.setChanging({ key, changing: true }));
    dispatch(Session.Nav.showBottom({}));
  };
  return (
    <>
      <Menu.Item itemKey="change-symbol" onClick={handleChange}>
        <Icon.Sync />
        Change symbol
      </Menu.Item>
      {custom != null && (
        <Menu.Item
          itemKey="edit-symbol"
          onClick={() => openEditor({ symbolKey: custom })}
        >
          <Icon.Edit />
          Edit symbol
        </Menu.Item>
      )}
      <Menu.Divider />
    </>
  );
};
