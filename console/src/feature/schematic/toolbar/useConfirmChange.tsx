// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { Schematic } from "@synnaxlabs/pluto";
import { deep, strings } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { Symbol } from "@/feature/schematic/symbol";
import { Modals } from "@/platform/modals";

/** A symbol node loses bindings. A node of another item loses its item settings. */
type LosingNode =
  | { key: string; config: schematic.SymbolNodeConfig; lost: schematic.Orphans }
  | { key: string; config: Schematic.Node.Config; lost: null };

/** The states and commands that a symbol takes bindings for. */
type Bindable = Pick<schematic.symbol.Spec, "states" | "commands">;

interface ConfirmChangeParams {
  /** The nodes that lose bindings or settings in the change. */
  nodes: LosingNode[];
  /** The spec of the new symbol. */
  spec: Bindable;
}

const NO_SPEC: Bindable = { states: [], commands: [] };

const nodeLabel = (config: Schematic.Node.Config): string =>
  "label" in config && config.label.label !== ""
    ? config.label.label
    : "A node with no label";

/** What a node of another item loses in the change, with the name of its item. */
const itemLoss = (config: Schematic.Node.Config): string => {
  const { name } = Schematic.Node.resolveSpec(config.variant);
  return `${nodeLabel(config)} loses its ${name} settings.`;
};

interface LossProps {
  config: schematic.SymbolNodeConfig;
  lost: schematic.Orphans;
  spec: Bindable;
}

/** What a node loses in the change, with the names that its old symbol gives. */
const Loss = ({ config, lost, spec }: LossProps): ReactElement => {
  const ref = config.spec;
  const custom = Schematic.Symbol.useResolved(ref.type === "custom" ? ref.key : null);
  const before =
    ref.type === "builtin"
      ? Schematic.Node.BUILT_INS.get(ref.name)
      : custom.symbol?.data;
  const names = Symbol.losses(lost, before ?? NO_SPEC, spec);
  return (
    <Text.Text level="small" color={9}>
      {`${nodeLabel(config)} loses ${strings.naturalLanguageJoin(names)}.`}
    </Text.Text>
  );
};

const ConfirmChange = ({
  close,
  nodes,
  spec,
}: Modals.ContentProps<ConfirmChangeParams, boolean>): ReactElement => (
  <Modals.ConfirmLayout
    title="Schematic.Symbol.Change"
    icon={<Icon.Schematic />}
    message="Change the symbol?"
    description="The change removes the bindings and settings that the new item lacks."
    confirm={{ label: "Change", variant: "warning" }}
    onConfirm={() => close(true)}
    onCancel={() => close(false)}
  >
    <Flex.Box y gap="tiny">
      {nodes.map((n) =>
        n.lost === null ? (
          <Text.Text key={n.key} level="small" color={9}>
            {itemLoss(n.config)}
          </Text.Text>
        ) : (
          <Loss key={n.key} config={n.config} lost={n.lost} spec={spec} />
        ),
      )}
    </Flex.Box>
  </Modals.ConfirmLayout>
);

const useConfirm = Modals.createPrompt<boolean, ConfirmChangeParams>(ConfirmChange);

const losing = (
  configs: Map<string, Schematic.Node.Config>,
  target: string,
  spec: Bindable,
): LosingNode[] =>
  [...configs].flatMap(([key, config]): LosingNode[] => {
    if (config.variant !== "symbol")
      return Schematic.Node.symbolKey(config) === target
        ? []
        : [{ key, config, lost: null }];
    const lost = schematic.orphans(config, spec.states, spec.commands);
    const count = lost.states.length + lost.overrides.length + lost.commands.length;
    return count > 0 ? [{ key, config, lost }] : [];
  });

const sameLosses = (a: LosingNode[], b: LosingNode[]): boolean =>
  a.length === b.length &&
  a.every(
    (n, i) =>
      n.key === b[i].key &&
      Schematic.Node.symbolKey(n.config) === Schematic.Node.symbolKey(b[i].config) &&
      deep.equal(n.lost, b[i].lost),
  );

/**
 * @returns a callback that resolves true when a change to the item that input names may
 * go ahead. read returns the configs to change, and spec is the spec of a symbol. It
 * asks first when the change removes a binding from a symbol node, or gives a node of
 * another item a different item. It asks again if what the nodes lose changes while it
 * asks. With no spec, the new item takes no bindings, so a change to it removes each
 * one.
 */
export const useConfirmChange = () => {
  const confirm = useConfirm();
  return async <V extends Schematic.Node.Variant>(
    read: () => Map<string, Schematic.Node.Config>,
    input: Schematic.Node.Input<V>,
    spec: Bindable = NO_SPEC,
  ): Promise<boolean> => {
    const target = Schematic.Node.symbolKey(Schematic.Node.createConfig(input));
    let nodes = losing(read(), target, spec);
    while (nodes.length > 0) {
      if ((await confirm({ nodes, spec })) !== true) return false;
      const now = losing(read(), target, spec);
      if (sameLosses(nodes, now)) return true;
      nodes = now;
    }
    return true;
  };
};
