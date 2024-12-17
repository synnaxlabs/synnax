// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { direction, location, type spatial } from "@synnaxlabs/x/spatial";

import { type BEM, newBEM } from "@/css/bem";
import { CSSGridBuilder } from "@/css/grid";
import { applyCSSVars, removeCSSVars } from "@/css/vars";
import { type ComponentSize } from "@/util/component";

export interface CSSType extends BEM {
  visible: (visible: boolean) => string;
  expanded: (expanded: boolean) => string;
  loc: (location: location.Crude) => string;
  align: (position: spatial.Alignment | "") => string;
  dir: (direction?: direction.Crude) => string | false;
  size: (size: ComponentSize | number) => string | false;
  sharp: (sharp?: boolean) => string | false;
  disabled: (disabled?: boolean) => string | false;
  rounded: (rounded?: boolean) => string | false;
  bordered: (location?: location.Crude | spatial.Alignment | boolean) => string | false;
  noSelect: string;
  selected: (selected: boolean) => string | false;
  altColor: (secondary: boolean) => string | false;
  editable: (editable: boolean) => string | false;
  noWrap: (noWrap: boolean) => string | false;
  applyVars: typeof applyCSSVars;
  removeVars: typeof removeCSSVars;
  newGridBuilder: (prefix?: string) => CSSGridBuilder;
  inheritDims: (inherit?: boolean) => string | false;
  dropRegion: (active: boolean) => false | string;
  triggerExclude: (value: boolean) => string | false;
  px: (value: number) => string;
  shade: (value: number) => string;
  shadeVar: (value?: number) => string | undefined;
  levelSizeVar: (value: string) => string;
}

const newCSS = (prefix: string): CSSType => {
  const CSS = newBEM(prefix) as CSSType;
  CSS.visible = (visible) => CSS.M(visible ? "visible" : "hidden");
  CSS.expanded = (expanded) => CSS.M(expanded ? "expanded" : "collapsed");
  CSS.loc = (l) => CSS.M(location.construct(l));
  CSS.disabled = (disabled) => disabled === true && CSS.M("disabled");
  CSS.align = (position) => CSS.M(position);
  CSS.dir = (dir) => dir != null && CSS.M(direction.construct(dir));
  CSS.size = (size) => typeof size === "string" && CSS.M(size);
  CSS.sharp = (sharp) => !(sharp === false) && CSS.M("sharp");
  CSS.rounded = (rounded) => !(rounded === false) && CSS.M("rounded");
  CSS.bordered = (location) => {
    if (typeof location === "boolean") return location && CSS.M("bordered");
    return location != null ? CSS.M(`bordered-${location}`) : CSS.M("bordered");
  };
  CSS.selected = (selected) => selected && CSS.M("selected");
  CSS.altColor = (secondary) => secondary && CSS.M("alt-color");
  CSS.editable = (editable) => editable && CSS.M("editable");
  CSS.noSelect = CSS.M("no-select");
  CSS.noWrap = (noWrap) => noWrap && CSS.M("no-wrap");
  CSS.applyVars = applyCSSVars;
  CSS.removeVars = removeCSSVars;
  CSS.newGridBuilder = (prefix?: string) => new CSSGridBuilder(prefix);
  CSS.dropRegion = (active) => active && CSS.B("haul-drop-region");
  CSS.px = (value: number) => `${value}px`;
  CSS.inheritDims = (inherit = true) => inherit && CSS.M("inherit-dims");
  CSS.shade = (value) => CSS.M(`shade-${value}`);
  CSS.shadeVar = (value) => {
    if (value == null) return undefined;
    return `var(--${prefix}-gray-l${value})`;
  };
  CSS.levelSizeVar = (value) => `var(--${prefix}-${value}-size)`;
  return CSS;
};

export const CSS = newCSS("pluto");
