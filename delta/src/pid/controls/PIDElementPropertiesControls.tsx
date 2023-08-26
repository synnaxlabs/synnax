// Copyright 2023 synnax labs, inc.
//
// use of this software is governed by the business source license included in the file
// licenses/bsl.txt.
//
// as of the change date specified in that file, in accordance with the business source
// license, use of this software will be governed by the apache license, version 2.0,
// included in the file licenses/apl.txt.

import { ReactElement } from "react";

import { Status, Color, Input, Align, PIDElement } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";

import { PIDElementInfo, useSelectSelectedPIDElementsProps } from "../store/selectors";
import { setPIDElementProps } from "../store/slice";

import "@/pid/controls/PIDElementPropertiesControls.css";

export interface PIDPropertiesProps {
  layoutKey: string;
}

export const PIDElementPropertiesControls = ({
  layoutKey,
}: PIDPropertiesProps): ReactElement => {
  const elements = useSelectSelectedPIDElementsProps(layoutKey);

  const dispatch = useDispatch();

  const handleChange = (key: string, props: any): void => {
    dispatch(setPIDElementProps({ layoutKey, key, props }));
  };

  if (elements.length === 0) {
    return (
      <Status.Text.Centered variant="disabled" hideIcon>
        Select a PID element to configure its properties.
      </Status.Text.Centered>
    );
  }

  if (elements.length > 1) {
    const groups: Record<string, PIDElementInfo[]> = {};
    elements.forEach((e) => {
      let color: Color.Color | null = null;
      if (e.type === "edge") color = new Color.Color(e.edge.color ?? Color.ZERO);
      else if ("color" in e.props) color = new Color.Color(e.props.color);
      if (color === null) return;
      const hex = color.hex;
      if (!(hex in groups)) groups[hex] = [];
      groups[hex].push(e);
    });
    return (
      <Align.Space className={CSS.B("pid-properties")} size="small">
        <Input.Label>Selection Colors</Input.Label>
        {Object.entries(groups).map(([hex, elements]) => {
          return (
            <Color.Swatch
              key={elements[0].key}
              value={hex}
              onChange={(color) => {
                elements.forEach((e) => {
                  handleChange(e.key, { color: color.hex });
                });
              }}
            />
          );
        })}
      </Align.Space>
    );
  }

  const selected = elements[0];

  if (selected.type === "edge") {
    return (
      <Color.Swatch
        value={selected.edge.color ?? Color.ZERO}
        onChange={(color) => {
          handleChange(selected.key, { color: color.hex });
        }}
      />
    );
  }

  const C = PIDElement.REGISTRY[selected.props.type];

  return (
    <Align.Space className={CSS.B("pid-properties")} size="small">
      <C.Form
        value={selected.props}
        onChange={(props) => handleChange(selected.key, props)}
      />
    </Align.Space>
  );
};
