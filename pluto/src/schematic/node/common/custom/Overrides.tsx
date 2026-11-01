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
import { Button } from "@synnaxlabs/lyra/button";
import { Color } from "@synnaxlabs/lyra/color";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { caseconv, deep } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useState } from "react";

import { CSS } from "@/css";
import { Region } from "@/schematic/node/common/region";
import { Symbol } from "@/schematic/symbol";

const path = deep.path<{
  specKey: string;
  stateOverrides: schematic.symbol.State[];
}>();

type RegionPath = deep.Path<schematic.symbol.Region>;

interface RegionControlsProps {
  path: RegionPath;
  svg: string;
  onReset: (path: RegionPath) => void;
  getOriginalRegion: (path: RegionPath) => schematic.symbol.Region | null;
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
  svg,
  onReset,
  getOriginalRegion,
}: RegionControlsProps): ReactElement => {
  const name = Form.useFieldValue(path.name);
  const region = Form.useFieldValue(path);
  const { set } = Form.useContext();
  const paint = Region.paint(svg, region.selectors);
  const originalRegion = getOriginalRegion(path);
  const canBeReset = !deep.equal(region, originalRegion);
  return (
    <Flex.Box x align="center">
      <Text.Text
        level="small"
        color={9}
        className={CSS.B("schematic-region-name")}
        overflow="fade"
      >
        {caseconv.capitalize(name)}
      </Text.Text>
      <Flex.Box x align="stretch" key={String(path)}>
        <Color.Swatch
          value={region.strokeColor}
          fallback={paint.stroke}
          onChange={(c) => set(path.strokeColor, c)}
        />
        <Color.Swatch
          value={region.fillColor}
          fallback={paint.fill}
          onChange={(c) => set(path.fillColor, c)}
        />
        <Button.Button
          onClick={() => onReset(path)}
          variant="text"
          size="tiny"
          className={CSS.cls(CSS.B("schematic-region-reset"), CSS.visible(canBeReset))}
        >
          <Icon.Reset />
        </Button.Button>
      </Flex.Box>
    </Flex.Box>
  );
};

interface RegionListProps {
  svg: string;
  onReset: (path: RegionPath) => void;
  getOriginalRegion: (path: RegionPath) => schematic.symbol.Region | null;
  selectedState: string;
}

const RegionList = ({
  svg,
  selectedState,
  onReset,
  getOriginalRegion,
}: RegionListProps) => {
  const list = path.stateOverrides[selectedState].regions;
  const { data: regions } = Form.useFieldList(list);
  return (
    <Flex.Box y align="stretch">
      {regions.map((region) => (
        <RegionControls
          key={region}
          svg={svg}
          onReset={onReset}
          path={list[region]}
          getOriginalRegion={getOriginalRegion}
        />
      ))}
    </Flex.Box>
  );
};

export const StateOverrideForm = (): ReactElement => {
  const specKey = Form.useFieldValue(path.specKey);
  const form = Form.useContext();
  const [originalStates, setOriginalStates] = useState<schematic.symbol.State[]>([]);
  const { data: states } = Form.useFieldList(path.stateOverrides);
  const [selectedState, setSelectedState] = useState<string | undefined>(states?.[0]);

  const applySymbol = (symbol: schematic.symbol.Symbol) => {
    if (symbol.data == null) return;
    const symbolSpec = symbol.data;
    setOriginalStates(deep.copy(symbolSpec.states));
    const currentOverrides = form.get(path.stateOverrides);
    if (currentOverrides.value?.length === 0) {
      form.set(path.stateOverrides, deep.copy(symbolSpec.states));
      setSelectedState(symbolSpec.states[0].key);
    } else {
      const syncedStates = syncStateOverrides(
        currentOverrides.value,
        symbolSpec.states,
      );
      form.set(path.stateOverrides, syncedStates);
    }
  };
  // A dangling spec key must not throw here: the owner renders a repair form for it.
  const { symbol } = Symbol.useResolved(specKey);
  useEffect(() => {
    if (symbol == null) return;
    applySymbol(symbol);
  }, [symbol, applySymbol]);

  const getOriginalRegion = (path: RegionPath): schematic.symbol.Region | null => {
    const parsedPath = String(path).split(".").slice(1).join(".");
    return deep.get(originalStates, parsedPath, { optional: true }) ?? null;
  };

  const resetRegion = (path: RegionPath) => {
    const prev = getOriginalRegion(path);
    if (prev == null) return;
    form.set(path, deep.copy(prev));
  };

  // The form can swap to a different symbol's overrides beneath the locally
  // held selection, so fall back to the first state when the key disappears.
  const shownState =
    selectedState != null && states.includes(selectedState) ? selectedState : states[0];

  return (
    <Flex.Box y align="stretch">
      {states.length > 1 && (
        <Select.Buttons value={shownState} onChange={setSelectedState} full="x">
          {states.map((state) => (
            <Select.Item key={state} itemKey={state} justify="center">
              {caseconv.capitalize(state)}
            </Select.Item>
          ))}
        </Select.Buttons>
      )}
      {shownState != null && (
        <RegionList
          svg={symbol?.data?.svg ?? ""}
          selectedState={shownState}
          onReset={resetRegion}
          getOriginalRegion={getOriginalRegion}
        />
      )}
    </Flex.Box>
  );
};
