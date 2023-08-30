// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Direction,
  Location,
  CrudeLocation,
  LooseDirectionT,
  LooseLocationT,
  CrudePosition,
} from "@synnaxlabs/x";

import { BEM, newBEM } from "@/css/bem";
import { applyCSSVars } from "@/css/vars";
import { ComponentSize } from "@/util/component";

import { CSSGridBuilder } from "./grid";

export interface CSSType extends BEM {
  visible: (visible: boolean) => string;
  expanded: (expanded: boolean) => string;
  loc: (location: LooseLocationT) => string;
  pos: (position: CrudePosition | "") => string;
  dir: (direction?: LooseDirectionT) => string | false;
  size: (size: ComponentSize | number) => string | false;
  sharp: (sharp?: boolean) => string | false;
  disabled: (disabled?: boolean) => string | false;
  rounded: (rounded?: boolean) => string | false;
  bordered: (location?: CrudeLocation | CrudePosition | boolean) => string | false;
  noSelect: string;
  selected: (selected: boolean) => string | false;
  editable: (editable: boolean) => string | false;
  noWrap: (noWrap: boolean) => string | false;
  applyVars: typeof applyCSSVars;
  newGridBuilder: () => CSSGridBuilder;
  dropRegion: (active: boolean) => false | string;
  px: (value: number) => string;
}

const newCSS = (prefix: string): CSSType => {
  const CSS = newBEM(prefix) as CSSType;
  CSS.visible = (visible) => CSS.M(visible ? "visible" : "hidden");
  CSS.expanded = (expanded) => CSS.M(expanded ? "expanded" : "collapsed");
  CSS.loc = (location) => CSS.M(new Location(location).valueOf());
  CSS.disabled = (disabled) => disabled === true && CSS.M("disabled");
  CSS.pos = (position) => CSS.M(position);
  CSS.dir = (direction) =>
    direction != null && CSS.M(new Direction(direction).valueOf());
  CSS.size = (size) => typeof size === "string" && CSS.M(size);
  CSS.sharp = (sharp) => !(sharp === false) && CSS.M("sharp");
  CSS.rounded = (rounded) => !(rounded === false) && CSS.M("rounded");
  CSS.bordered = (location) => {
    if (typeof location === "boolean") return location && CSS.M("bordered");
    return location != null ? CSS.M("bordered-" + location) : CSS.M("bordered");
  };
  CSS.selected = (selected) => selected && CSS.M("selected");
  CSS.editable = (editable) => editable && CSS.M("editable");
  CSS.noSelect = CSS.M("no-select");
  CSS.noWrap = (noWrap) => noWrap && CSS.M("no-wrap");
  CSS.applyVars = applyCSSVars;
  CSS.newGridBuilder = () => new CSSGridBuilder();
  CSS.dropRegion = (active) => active && CSS.B("haul-drop-region");
  CSS.px = (value: number) => `${value}px`;
  return CSS;
};

export const CSS = newCSS("pluto");
