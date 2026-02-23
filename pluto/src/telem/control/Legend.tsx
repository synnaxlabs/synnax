// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError } from "@synnaxlabs/client";
import { color, unique } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect, useState } from "react";

import { Aether } from "@/aether";
import { Color } from "@/color";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { type state } from "@/state";
import { control } from "@/telem/control/aether";
import { useContext } from "@/telem/control/Controller";
import { Text } from "@/text";
import { Legend as Base } from "@/vis/legend";

const SUBJECT_NAME_RE = /^(.+?)\s*\((.+)\)$/;

interface ParsedName {
  primary: string;
  secondary?: string;
}

const parseSubjectName = (name: string): ParsedName => {
  const match = SUBJECT_NAME_RE.exec(name);
  if (match == null) return { primary: name };
  return { primary: match[1], secondary: match[2] };
};

export interface LegendProps extends Omit<Base.SimpleProps, "data" | "onEntryChange"> {
  colors?: Record<string, string>;
  onColorsChange?: (colors: Record<string, string>) => void;
}

export const Legend = (props: LegendProps): ReactElement | null => {
  const { key: contextKey, needsControlOf } = useContext();
  const [, { states }, setState] = Aether.use({
    type: control.Legend.TYPE,
    schema: control.legendStateZ,
    initialState: { states: [], needsControlOf },
  });

  useEffect(() => {
    setState((state) => ({ ...state, needsControlOf }));
  }, [needsControlOf]);

  const {
    colors = {},
    onColorsChange,
    position,
    onPositionChange,
    allowVisibleChange: _,
    background = 1,
    ...rest
  } = props;

  const handleColorChange = useCallback(
    (key: string, c: color.Crude) =>
      onColorsChange?.({ ...colors, [key]: color.hex(c) }),
    [colors, onColorsChange],
  );

  const subjects = unique.unique(states.map((s) => s.subject.key));
  const data = subjects.map((s) => {
    const d = states.find((s2) => s2.subject.key === s);
    if (d == null) throw new UnexpectedError("Legend subject not found");
    return {
      key: d.subject.key,
      name: d.subject.name,
      color: colors[d.subject.key] ?? d.subjectColor,
      isSelf: d.subject.key === contextKey,
    };
  });

  data.sort((a, b) => {
    if (a.isSelf !== b.isSelf) return a.isSelf ? -1 : 1;
    return a.name.localeCompare(b.name);
  });

  const [pickerVisible, setPickerVisible] = useState(false);

  if (data.length === 0) return null;

  return (
    <Base.Container
      {...rest}
      draggable={!pickerVisible}
      value={position}
      onChange={onPositionChange}
      background={background}
      empty
    >
      {data.map((d) => (
        <LegendEntry
          key={d.key}
          entryKey={d.key}
          name={d.name}
          color={d.color}
          isSelf={d.isSelf}
          onColorChange={handleColorChange}
          onColorPickerVisibleChange={setPickerVisible}
        />
      ))}
    </Base.Container>
  );
};

interface LegendEntryProps {
  entryKey: string;
  name: string;
  color: color.Crude;
  isSelf: boolean;
  onColorChange: (key: string, color: color.Crude) => void;
  onColorPickerVisibleChange: state.Setter<boolean>;
}

const LegendEntry = ({
  entryKey,
  name,
  color: entryColor,
  isSelf,
  onColorChange,
  onColorPickerVisibleChange,
}: LegendEntryProps): ReactElement => {
  const parsed = parseSubjectName(name);
  const handleColorChange = useCallback(
    (c: color.Crude) => onColorChange(entryKey, c),
    [entryKey, onColorChange],
  );
  return (
    <Flex.Box
      align="center"
      className={CSS(CSS.B("legend-entry"))}
      gap="small"
      x
      justify="between"
      grow
    >
      <Flex.Box align="center" gap="small" x>
        <Color.Swatch
          allowChange
          draggable={false}
          onChange={handleColorChange}
          size="tiny"
          value={entryColor}
          onVisibleChange={onColorPickerVisibleChange}
        />
        {isSelf && <Icon.User style={{ fontSize: "1.2rem", opacity: 0.7 }} />}
        <Text.Text level="small" color={10} overflow="nowrap">
          {parsed.primary}
        </Text.Text>
        {parsed.secondary != null && (
          <Text.Text level="small" color={7} overflow="nowrap">
            ({parsed.secondary})
          </Text.Text>
        )}
      </Flex.Box>
    </Flex.Box>
  );
};
