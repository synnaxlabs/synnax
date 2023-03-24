// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Direction, Location, Position } from "@synnaxlabs/x";

import { ComponentSize } from "..";

import { BEM, newBEM } from "./bem";

export interface CSSType extends BEM {
  visible: (visible: boolean) => string;
  expanded: (expanded: boolean) => string;
  loc: (location: Location) => string;
  pos: (position: Position) => string;
  dir: (direction?: Direction) => string | false;
  size: (size: ComponentSize | number) => string | false;
  sharp: (sharp?: boolean) => string | false;
  rounded: (rounded?: boolean) => string | false;
  bordered: (location?: Location | Position | boolean) => string | false;
  noSelect: string;
  selected: (selected: boolean) => string | false;
  noWrap: (noWrap: boolean) => string | false;
}

const newCSS = (prefix: string): CSSType => {
  const CSS = newBEM(prefix) as CSSType;
  CSS.visible = (visible) => CSS.M(visible ? "visible" : "hidden");
  CSS.expanded = (expanded) => CSS.M(expanded ? "expanded" : "collapsed");
  CSS.loc = (location) => CSS.M(location);
  CSS.pos = (position) => CSS.M(position);
  CSS.dir = (direction) => direction != null && CSS.M(direction);
  CSS.size = (size) => typeof size === "string" && CSS.M(size);
  CSS.sharp = (sharp) => !(sharp === false) && CSS.M("sharp");
  CSS.rounded = (rounded) => !(rounded === false) && CSS.M("rounded");
  CSS.bordered = (location) => {
    if (typeof location === "boolean") return location && CSS.M("bordered");
    return location != null ? CSS.M("bordered-" + location) : CSS.M("bordered");
  };
  CSS.selected = (selected) => selected && CSS.M("selected");
  CSS.noSelect = CSS.M("no-select");
  CSS.noWrap = (noWrap) => noWrap && CSS.M("no-wrap");
  return CSS;
};

export const CSS = newCSS("pluto");
