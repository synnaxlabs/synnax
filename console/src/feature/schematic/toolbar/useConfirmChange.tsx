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
import { strings } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { Symbol } from "@/feature/schematic/symbol";
import { Modals } from "@/platform/modals";

interface Losing {
  key: string;
  config: schematic.SymbolNodeConfig;
}

interface ConfirmChangeParams {
  /** The symbol nodes that lose bindings in the change. */
  nodes: Losing[];
  /** The spec of the new symbol. */
  spec: schematic.symbol.Spec;
}

const NO_SPEC: Pick<schematic.symbol.Spec, "states" | "commands"> = {
  states: [],
  commands: [],
};

interface LossProps {
  config: schematic.SymbolNodeConfig;
  spec: schematic.symbol.Spec;
}

/** What a node loses in the change, with the names that its old symbol gives. */
const Loss = ({ config, spec }: LossProps): ReactElement => {
  const ref = config.spec;
  const custom = Schematic.Symbol.useResolved(ref.type === "custom" ? ref.key : null);
  const before =
    ref.type === "builtin"
      ? Schematic.Node.BUILT_INS.get(ref.name)
      : custom.symbol?.data;
  const lost = Symbol.losses(
    schematic.orphans(config, spec.states, spec.commands),
    before ?? NO_SPEC,
    spec,
  );
  const label = config.label.label === "" ? "A node with no label" : config.label.label;
  return (
    <Text.Text level="small" color={9}>
      {`${label} loses ${strings.naturalLanguageJoin(lost)}.`}
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
    description="The new symbol lacks bindings that these nodes use, so the change removes them."
    confirm={{ label: "Change", variant: "warning" }}
    onConfirm={() => close(true)}
    onCancel={() => close(false)}
  >
    <Flex.Box y gap="tiny">
      {nodes.map(({ key, config }) => (
        <Loss key={key} config={config} spec={spec} />
      ))}
    </Flex.Box>
  </Modals.ConfirmLayout>
);

const useConfirm = Modals.createPrompt<boolean, ConfirmChangeParams>(ConfirmChange);

const orphaned = (config: schematic.SymbolNodeConfig, spec: schematic.symbol.Spec) => {
  const { states, overrides, commands } = schematic.orphans(
    config,
    spec.states,
    spec.commands,
  );
  return states.length + overrides.length + commands.length > 0;
};

/**
 * @returns a callback that resolves true when a change of configs to the symbol with
 * spec may go ahead. It asks first when the change removes a binding from a node.
 */
export const useConfirmChange = () => {
  const confirm = useConfirm();
  return async (
    configs: Map<string, Schematic.Node.Config>,
    spec?: schematic.symbol.Spec,
  ): Promise<boolean> => {
    if (spec == null) return true;
    const nodes = [...configs].flatMap(([key, config]) =>
      config.variant === "symbol" && orphaned(config, spec) ? [{ key, config }] : [],
    );
    return nodes.length === 0 || (await confirm({ nodes, spec })) === true;
  };
};
