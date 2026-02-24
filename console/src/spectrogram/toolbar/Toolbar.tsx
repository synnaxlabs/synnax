// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Channel, Flex, Icon, Input, Select, Tabs } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { Toolbar as Base } from "@/components";
import { Layout } from "@/layout";
import { useSelect, useSelectToolbar } from "@/spectrogram/selectors";
import {
  setActiveToolbarTab,
  setChannel,
  setDisplay,
  setFFTParams,
  setSampleRate,
  type ToolbarTab,
} from "@/spectrogram/slice";

interface Tab {
  tabKey: ToolbarTab;
  name: string;
}

const TABS: Tab[] = [
  { tabKey: "data", name: "Data" },
  { tabKey: "display", name: "Display" },
];

export interface ToolbarProps {
  layoutKey: string;
}

const FFT_SIZE_KEYS = [256, 512, 1024, 2048, 4096, 8192] as const;

const COLOR_MAP_KEYS = [
  "viridis",
  "inferno",
  "magma",
  "plasma",
  "jet",
  "grayscale",
] as const;
type ColorMap = (typeof COLOR_MAP_KEYS)[number];

const WINDOW_FUNCTION_KEYS = ["hann", "blackmanHarris"] as const;
type WindowFunction = (typeof WINDOW_FUNCTION_KEYS)[number];

const DataTab = ({ layoutKey }: { layoutKey: string }): ReactElement | null => {
  const vis = useSelect(layoutKey);
  const dispatch = useDispatch();
  if (vis == null) return null;

  const handleChannelChange = (ch: channel.Key): void => {
    dispatch(setChannel({ key: layoutKey, channel: ch }));
  };

  const handleSampleRateChange: Input.Control<number>["onChange"] = (v) => {
    dispatch(setSampleRate({ key: layoutKey, sampleRate: v }));
  };

  return (
    <Flex.Box y style={{ padding: "2rem" }} gap="small">
      <Input.Item label="Channel" y grow>
        <Channel.SelectSingle
          value={vis.channel !== 0 ? vis.channel : undefined}
          onChange={handleChannelChange}
        />
      </Input.Item>
      <Input.Item label="Sample Rate (Hz)" y grow>
        <Input.Numeric value={vis.sampleRate} onChange={handleSampleRateChange} />
      </Input.Item>
    </Flex.Box>
  );
};

const DisplayTab = ({ layoutKey }: { layoutKey: string }): ReactElement | null => {
  const vis = useSelect(layoutKey);
  const dispatch = useDispatch();
  if (vis == null) return null;

  const handleFFTSizeChange = (v: number): void => {
    dispatch(setFFTParams({ key: layoutKey, fftSize: v }));
  };

  const handleWindowFunctionChange = (v: WindowFunction): void => {
    dispatch(setFFTParams({ key: layoutKey, windowFunction: v }));
  };

  const handleOverlapChange: Input.Control<number>["onChange"] = (v) => {
    dispatch(setFFTParams({ key: layoutKey, overlap: v / 100 }));
  };

  const handleColorMapChange = (v: ColorMap): void => {
    dispatch(setDisplay({ key: layoutKey, colorMap: v }));
  };

  const handleDbMinChange: Input.Control<number>["onChange"] = (v) => {
    dispatch(setDisplay({ key: layoutKey, dbMin: v }));
  };

  const handleDbMaxChange: Input.Control<number>["onChange"] = (v) => {
    dispatch(setDisplay({ key: layoutKey, dbMax: v }));
  };

  const handleFreqMinChange: Input.Control<number>["onChange"] = (v) => {
    dispatch(setDisplay({ key: layoutKey, freqMin: v }));
  };

  const handleFreqMaxChange: Input.Control<number>["onChange"] = (v) => {
    dispatch(setDisplay({ key: layoutKey, freqMax: v }));
  };

  return (
    <Flex.Box y style={{ padding: "2rem" }} gap="small">
      <Flex.Box x gap="small">
        <Input.Item label="FFT Size" y>
          <Select.Buttons
            keys={FFT_SIZE_KEYS}
            value={vis.fftSize}
            onChange={handleFFTSizeChange}
          >
            {FFT_SIZE_KEYS.map((s) => (
              <Select.Button key={s} itemKey={s} size="small">
                {s}
              </Select.Button>
            ))}
          </Select.Buttons>
        </Input.Item>
        <Input.Item label="Overlap (%)" y grow>
          <Input.Numeric
            value={vis.overlap * 100}
            onChange={handleOverlapChange}
            bounds={{ lower: 0, upper: 95 }}
          />
        </Input.Item>
      </Flex.Box>
      <Flex.Box x gap="small">
        <Input.Item label="Window" y>
          <Select.Buttons
            keys={WINDOW_FUNCTION_KEYS}
            value={vis.windowFunction}
            onChange={handleWindowFunctionChange}
          >
            <Select.Button itemKey="hann" size="small">
              Hann
            </Select.Button>
            <Select.Button itemKey="blackmanHarris" size="small">
              Blackman-Harris
            </Select.Button>
          </Select.Buttons>
        </Input.Item>
        <Input.Item label="Color Map" y grow>
          <Select.Buttons
            keys={COLOR_MAP_KEYS}
            value={vis.colorMap}
            onChange={handleColorMapChange}
          >
            {COLOR_MAP_KEYS.map((c) => (
              <Select.Button key={c} itemKey={c} size="small">
                {c.charAt(0).toUpperCase() + c.slice(1)}
              </Select.Button>
            ))}
          </Select.Buttons>
        </Input.Item>
      </Flex.Box>
      <Flex.Box x gap="small">
        <Input.Item label="dB Min" y grow>
          <Input.Numeric value={vis.dbMin} onChange={handleDbMinChange} />
        </Input.Item>
        <Input.Item label="dB Max" y grow>
          <Input.Numeric value={vis.dbMax} onChange={handleDbMaxChange} />
        </Input.Item>
      </Flex.Box>
      <Flex.Box x gap="small">
        <Input.Item label="Freq Min (Hz)" y grow>
          <Input.Numeric value={vis.freqMin} onChange={handleFreqMinChange} />
        </Input.Item>
        <Input.Item label="Freq Max (Hz)" y grow>
          <Input.Numeric value={vis.freqMax} onChange={handleFreqMaxChange} />
        </Input.Item>
      </Flex.Box>
    </Flex.Box>
  );
};

export const Toolbar = ({ layoutKey }: ToolbarProps): ReactElement | null => {
  const { name } = Layout.useSelectRequired(layoutKey);
  const dispatch = useDispatch();
  const state = useSelectToolbar(layoutKey);

  const content = useCallback(
    ({ tabKey }: Tabs.Tab) => {
      switch (tabKey) {
        case "display":
          return <DisplayTab layoutKey={layoutKey} />;
        default:
          return <DataTab layoutKey={layoutKey} />;
      }
    },
    [layoutKey],
  );

  const handleTabSelect = useCallback(
    (tabKey: string): void => {
      dispatch(setActiveToolbarTab({ key: layoutKey, tab: tabKey as ToolbarTab }));
    },
    [dispatch, layoutKey],
  );

  const value = useMemo(
    () => ({
      tabs: TABS,
      selected: state?.activeTab,
      content,
      onSelect: handleTabSelect,
    }),
    [state?.activeTab, content, handleTabSelect],
  );

  if (state == null) return null;

  return (
    <Base.Content>
      <Tabs.Provider value={value}>
        <Base.Header>
          <Base.Title icon={<Icon.Visualize />}>{name}</Base.Title>
          <Tabs.Selector style={{ borderBottom: "none" }} />
        </Base.Header>
        <Tabs.Content />
      </Tabs.Provider>
    </Base.Content>
  );
};
