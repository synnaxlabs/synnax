// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/color/PairEditor.css";

import { color } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useState } from "react";

import { Button } from "@/button";
import { Editor } from "@/color/Editor";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Select } from "@/select";
import { theme } from "@/theme";
import { Theming } from "@/theming";

type Mode = "light" | "dark";

const DARK = theme.themeZ.parse(theme.SYNNAX_DARK);

export interface PairEditorProps {
  value: theme.Pair;
  onChange: (value: theme.Pair) => void;
  /** Shown at the left of the switch between light and dark, such as a field label. */
  label?: ReactNode;
}

const Dot = ({ value }: { value: color.Crude }): ReactElement => {
  const style: CSS.VarProperties = { "--pluto-swatch-color": color.cssString(value) };
  return <span className={CSS.BE("color-pair-editor", "dot")} style={style} />;
};

/**
 * Edits the light and dark values of a color in one editor, with a switch between the
 * two, each with a swatch of its value. It starts on the mode of the current theme.
 * Auto clears the dark value. While it is absent, Auto is selected, and Dark shows the
 * color that the dark theme derives.
 */
export const PairEditor = ({
  value,
  onChange,
  label,
}: PairEditorProps): ReactElement => {
  const [mode, setMode] = useState<Mode>(Theming.use().dark ? "dark" : "light");
  const auto = value.dark == null;
  const dark = value.dark ?? theme.legible(value.light, DARK);
  const shown = mode === "light" ? value.light : dark;
  const handleChange = (c: color.Color): void =>
    onChange(mode === "light" ? { ...value, light: c } : { ...value, dark: c });
  return (
    <Flex.Box y gap="medium" className={CSS.B("color-pair-editor")}>
      <Flex.Box x align="center" gap="small">
        {label}
        <Select.Buttons<Mode>
          value={mode}
          onChange={setMode}
          x
          className={CSS.BE("color-pair-editor", "modes")}
        >
          <Select.Item itemKey="light" size="small" level="small">
            <Dot value={value.light} />
            Light
          </Select.Item>
          <Select.Item itemKey="dark" size="small" level="small">
            <Dot value={dark} />
            Dark
          </Select.Item>
        </Select.Buttons>
        <Button.Button
          variant="outlined"
          size="small"
          square
          className={CSS.cls(CSS.selected(auto))}
          onClick={() => onChange({ light: value.light })}
          tooltip="Auto: the dark theme derives the dark color"
          aria-label="Auto"
        >
          <Icon.Auto />
        </Button.Button>
      </Flex.Box>
      <Editor value={shown} onChange={handleChange} />
    </Flex.Box>
  );
};
