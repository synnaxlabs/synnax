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

interface LosingNode {
  key: string;
  config: schematic.SymbolNodeConfig;
  lost: schematic.Orphans;
}

/** The states and commands that a symbol takes bindings for. */
type Bindable = Pick<schematic.symbol.Spec, "states" | "commands">;

interface ConfirmChangeParams {
  /** The symbol nodes that lose bindings in the change. */
  nodes: LosingNode[];
  /** The spec of the new symbol. */
  spec: Bindable;
}

const NO_SPEC: Bindable = { states: [], commands: [] };

interface LossProps extends Omit<LosingNode, "key"> {
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
  const label = config.label.label === "" ? "A node with no label" : config.label.label;
  return (
    <Text.Text level="small" color={9}>
      {`${label} loses ${strings.naturalLanguageJoin(names)}.`}
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
    description="The change removes the bindings that the new symbol lacks."
    confirm={{ label: "Change", variant: "warning" }}
    onConfirm={() => close(true)}
    onCancel={() => close(false)}
  >
    <Flex.Box y gap="tiny">
      {nodes.map(({ key, config, lost }) => (
        <Loss key={key} config={config} lost={lost} spec={spec} />
      ))}
    </Flex.Box>
  </Modals.ConfirmLayout>
);

const useConfirm = Modals.createPrompt<boolean, ConfirmChangeParams>(ConfirmChange);

const losing = (
  configs: Map<string, Schematic.Node.Config>,
  spec: Bindable,
): LosingNode[] =>
  [...configs].flatMap(([key, config]) => {
    if (config.variant !== "symbol") return [];
    const lost = schematic.orphans(config, spec.states, spec.commands);
    const count = lost.states.length + lost.overrides.length + lost.commands.length;
    return count > 0 ? [{ key, config, lost }] : [];
  });

const sameLosses = (a: LosingNode[], b: LosingNode[]): boolean =>
  a.length === b.length &&
  a.every((n, i) => n.key === b[i].key && deep.equal(n.lost, b[i].lost));

/**
 * @returns a callback that resolves true when a change to the symbol with spec may go
 * ahead. read returns the configs to change. It asks first when the change removes a
 * binding from a node, and asks again if what the nodes lose changes while it asks.
 * With no spec, the new item takes no bindings, so a change to it removes each one.
 */
export const useConfirmChange = () => {
  const confirm = useConfirm();
  return async (
    read: () => Map<string, Schematic.Node.Config>,
    spec: Bindable = NO_SPEC,
  ): Promise<boolean> => {
    let nodes = losing(read(), spec);
    while (nodes.length > 0) {
      if ((await confirm({ nodes, spec })) !== true) return false;
      const now = losing(read(), spec);
      if (sameLosses(nodes, now)) return true;
      nodes = now;
    }
    return true;
  };
};
