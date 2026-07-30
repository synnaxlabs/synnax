// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/common/custom/Overrides.css";

import { type schematic } from "@synnaxlabs/client";
import { caseconv, color, deep } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useState } from "react";

import { Button } from "@/button";
import { Color } from "@/color";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { type Flux } from "@/flux";
import { Form } from "@/form";
import { Icon } from "@/icon";
import { useRetrieveEffect } from "@/schematic/symbol/queries";
import { Select } from "@/select";
import { Text } from "@/text";

interface RegionControlsProps {
  path: string;
  onReset: (path: string) => void;
  getOriginalRegion: (path: string) => schematic.symbol.Region | null;
}

const syncStateOverrides = (
  currentOverrides: schematic.symbol.State[],
  specStates: schematic.symbol.State[],
): schematic.symbol.State[] => {
  let syncedStates = [...currentOverrides];

  if (currentOverrides.length !== specStates.length) {
    const currentStateMap = new Map(
      currentOverrides.map((state) => [state.key, state]),
    );
    const specStateMap = new Map(specStates.map((state) => [state.key, state]));

    syncedStates = syncedStates.filter((state) => specStateMap.has(state.key));

    for (const specState of specStates)
      if (!currentStateMap.has(specState.key)) syncedStates.push(deep.copy(specState));
  }

  const finalStates = [...syncedStates];

  for (let stateIndex = 0; stateIndex < finalStates.length; stateIndex++) {
    const currentState = finalStates[stateIndex];
    const specState = specStates.find((s) => s.key === currentState.key);

    if (specState && currentState.regions.length !== specState.regions.length) {
      const currentRegions = [...currentState.regions];

      const currentRegionMap = new Map(
        currentRegions.map((region) => [region.key, region]),
      );
      const specRegionMap = new Map(
        specState.regions.map((region) => [region.key, region]),
      );

      const syncedRegions = currentRegions.filter((region) =>
        specRegionMap.has(region.key),
      );

      for (const specRegion of specState.regions)
        if (!currentRegionMap.has(specRegion.key))
          syncedRegions.push(deep.copy(specRegion));

      finalStates[stateIndex] = {
        ...currentState,
        regions: syncedRegions,
      };
    }
  }

  return finalStates;
};

const RegionControls = ({
  path,
  onReset,
  getOriginalRegion,
}: RegionControlsProps): ReactElement => {
  const name = Form.useFieldValue<string>(`${path}.name`);
  const region = Form.useFieldValue<schematic.symbol.Region>(path);
  const originalRegion = getOriginalRegion(path);
  const canBeReset = !deep.equal(region, originalRegion);
  return (
    <Flex.Box x align="center">
      <Text.Text
        level="small"
        color={9}
        className={CSS.B("schematic-region-name")}
        overflow="ellipsis"
      >
        {caseconv.capitalize(name)}
      </Text.Text>
      <Flex.Box x align="stretch" key={path}>
        <Form.Field<string>
          path={`${path}.strokeColor`}
          showLabel={false}
          padHelpText={false}
        >
          {({ value, onChange }) => (
            <Color.Swatch value={value} onChange={(v) => onChange(color.hex(v))} />
          )}
        </Form.Field>
        <Form.Field<string>
          path={`${path}.fillColor`}
          showLabel={false}
          padHelpText={false}
        >
          {({ value, onChange }) => (
            <Color.Swatch value={value} onChange={(v) => onChange(color.hex(v))} />
          )}
        </Form.Field>
        <Button.Button
          onClick={() => onReset(path)}
          variant="text"
          size="tiny"
          className={CSS(CSS.B("schematic-region-reset"), CSS.visible(canBeReset))}
        >
          <Icon.Reset />
        </Button.Button>
      </Flex.Box>
    </Flex.Box>
  );
};

interface RegionListProps {
  onReset: (path: string) => void;
  getOriginalRegion: (path: string) => schematic.symbol.Region | null;
  selectedState: string;
}

const RegionList = ({ selectedState, onReset, getOriginalRegion }: RegionListProps) => {
  const { data: regions } = Form.useFieldList<string, schematic.symbol.Region>(
    `stateOverrides.${selectedState}.regions`,
  );
  return (
    <Flex.Box y align="stretch">
      {regions.map((region) => (
        <RegionControls
          key={region}
          onReset={onReset}
          path={`stateOverrides.${selectedState}.regions.${region}`}
          getOriginalRegion={getOriginalRegion}
        />
      ))}
    </Flex.Box>
  );
};

export const StateOverrideForm = (): ReactElement => {
  const specKey = Form.useFieldValue<string>("specKey");
  const form = Form.useContext();
  const [originalStates, setOriginalStates] = useState<schematic.symbol.State[]>([]);
  const { data: states } = Form.useFieldList<string, schematic.symbol.State>(
    "stateOverrides",
  );
  const [selectedState, setSelectedState] = useState<string | undefined>(states?.[0]);

  useRetrieveEffect({
    query: { key: specKey },
    addStatusOnFailure: false,
    onChange: useCallback(
      (res: Flux.Result<schematic.symbol.Symbol>) => {
        if (res.data?.data == null) return;
        const symbolSpec = res.data.data;
        setOriginalStates(deep.copy(symbolSpec.states));
        const currentOverrides = form.get<schematic.symbol.State[]>("stateOverrides");
        if (currentOverrides.value?.length === 0) {
          form.set("stateOverrides", deep.copy(symbolSpec.states));
          setSelectedState(symbolSpec.states[0].key);
        } else {
          const syncedStates = syncStateOverrides(
            currentOverrides.value,
            symbolSpec.states,
          );
          form.set("stateOverrides", syncedStates);
        }
      },
      [form],
    ),
  });

  const resetRegion = useCallback(
    (path: string) => {
      const parsedPath = path.split(".").slice(1).join(".");
      const prev = deep.get(originalStates, parsedPath, { optional: true });
      if (prev == null) return;
      form.set(path, deep.copy(prev));
    },
    [form.set, originalStates],
  );

  const getOriginalRegion = useCallback(
    (path: string): schematic.symbol.Region | null => {
      const parsedPath = path.split(".").slice(1).join(".");
      return deep.get(originalStates, parsedPath, { optional: true }) ?? null;
    },
    [originalStates],
  );

  return (
    <Flex.Box y align="stretch">
      {states.length > 1 && (
        <Select.Buttons
          keys={states}
          value={selectedState}
          onChange={setSelectedState}
          full="x"
        >
          {states.map((state) => (
            <Select.Button key={state} itemKey={state} justify="center">
              {caseconv.capitalize(state)}
            </Select.Button>
          ))}
        </Select.Buttons>
      )}
      {selectedState != null && (
        <RegionList
          selectedState={selectedState}
          onReset={resetRegion}
          getOriginalRegion={getOriginalRegion}
        />
      )}
    </Flex.Box>
  );
};
