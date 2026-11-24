// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type resource, schematic } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { Schematic } from "@synnaxlabs/pluto";
import { compare, strings } from "@synnaxlabs/x";
import { type ReactElement, Suspense } from "react";

import { Modals } from "@/platform/modals";

interface ConfirmSaveParams {
  /** The symbol as the Core stores it. */
  stored: schematic.symbol.Symbol;
  /** The spec that the save stores. */
  spec: schematic.symbol.Spec;
  /** The schematics that use the symbol. */
  uses: resource.ID[];
}

const nameOf = (items: { key: string; name: string }[], key: string): string =>
  items.find((i) => i.key === key)?.name ?? key;

/** @returns what a node loses in the save, such as "the Level binding". */
const losses = (
  { states, overrides, commands }: schematic.Orphans,
  stored: schematic.symbol.Spec,
  spec: schematic.symbol.Spec,
): string[] => [
  ...states.map((k) => `the ${nameOf(stored.states, k)} binding`),
  ...overrides.map(
    ({ key, fields }) =>
      `the ${strings.naturalLanguageJoin(fields)} of ${nameOf(spec.states, key)}`,
  ),
  ...commands.map((k) => `the ${nameOf(stored.commands, k)} binding`),
];

interface UseProps extends Omit<ConfirmSaveParams, "uses"> {
  schematic: schematic.Schematic;
}

/** A schematic that uses the symbol, and what each of its nodes loses in the save. */
const Use = ({ schematic: s, stored, spec }: UseProps): ReactElement => {
  const nodes = s.nodes.flatMap((n) => {
    const cfg = n.config;
    if (cfg.variant !== "symbol") return [];
    if (cfg.spec.type !== "custom" || cfg.spec.key !== stored.key) return [];
    const lost = losses(
      schematic.orphans(cfg, spec.states, spec.commands),
      stored.data,
      spec,
    );
    if (lost.length === 0) return [];
    const label = cfg.label.label === "" ? "A node with no label" : cfg.label.label;
    return [
      { key: n.key, text: `${label} loses ${strings.naturalLanguageJoin(lost)}.` },
    ];
  });
  return (
    <Flex.Box y gap="tiny">
      <Text.Text weight={450} overflow="fade">
        <Icon.Schematic />
        {s.name}
      </Text.Text>
      {nodes.length === 0 ? (
        <Text.Text level="small" color={9}>
          Each node keeps its bindings.
        </Text.Text>
      ) : (
        nodes.map(({ key, text }) => (
          <Text.Text key={key} level="small" color={9}>
            {text}
          </Text.Text>
        ))
      )}
    </Flex.Box>
  );
};

interface UsesProps extends ConfirmSaveParams {}

/** Orders schematics by name, then by key, so equal names keep one order. */
const ORDER = compare.chain<schematic.Schematic>(
  compare.by((s) => s.name),
  compare.by((s) => s.key),
);

/** Lists the schematics that use the symbol. Suspends while they load. */
const Uses = ({ uses, ...rest }: UsesProps): ReactElement => (
  <Flex.Box y gap="medium" aria-label="Schematics">
    {Schematic.useMultiple({ keys: uses.map((u) => u.key) })
      .toSorted(ORDER)
      .map((s) => (
        <Use key={s.key} schematic={s} {...rest} />
      ))}
  </Flex.Box>
);

const ConfirmSave = ({
  close,
  ...rest
}: Modals.ContentProps<ConfirmSaveParams, boolean>): ReactElement => {
  const count = rest.uses.length;
  return (
    <Modals.ConfirmLayout
      title="Schematic.Symbol.Save"
      icon={<Icon.Schematic />}
      message={`Save the changes to ${rest.stored.name}?`}
      description={`${count} ${count === 1 ? "schematic uses" : "schematics use"} this symbol and will draw the changes.`}
      confirm={{ label: "Save", variant: "warning" }}
      onConfirm={() => close(true)}
      onCancel={() => close(false)}
    >
      <Suspense>
        <Uses {...rest} />
      </Suspense>
    </Modals.ConfirmLayout>
  );
};

/** Opens a confirm for a save of a symbol that schematics use. */
export const useConfirmSave = Modals.createPrompt<boolean, ConfirmSaveParams>(
  ConfirmSave,
);
