// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, direction, location, type spatial } from "@synnaxlabs/x";

import { type BEM, newBEM } from "@/css/bem";
import { CSSGridBuilder } from "@/css/grid";
import { applyCSSVars, removeCSSVars } from "@/css/vars";
import { type text } from "@/text/base";

export interface CSSType extends BEM {
  visible: (visible: boolean) => string;
  expanded: (expanded: boolean) => string;
  level: (level: text.Level) => string;
  loc: (location: location.Crude) => string;
  align: (position: spatial.Alignment | "") => string;
  dir: (direction?: direction.Crude) => string | false;
  clickable: (shade?: text.Shade) => string;
  sharp: (sharp?: boolean) => string | false;
  disabled: (disabled?: boolean) => string | false;
  rounded: (rounded?: boolean) => string | false;
  bordered: (location?: location.Crude | spatial.Alignment | boolean) => string | false;
  noSelect: string;
  selected: (selected: boolean) => string | false;
  altColor: (secondary: boolean) => string | false;
  editable: (editable: boolean) => string | false;
  applyVars: typeof applyCSSVars;
  removeVars: typeof removeCSSVars;
  newGridBuilder: (prefix?: string) => CSSGridBuilder;
  inheritDims: (inherit?: boolean) => string | false;
  dropRegion: (active: boolean) => false | string;
  triggerExclude: (value: boolean) => string | false;
  px: (value: number) => string;
  shade: ((value: text.Shade) => string) & ((value?: text.Shade) => string | false);
  colorVar: (value?: false | text.Shade | color.Crude) => string | undefined;
  levelSizeVar: (value: string) => string;
}

const newCSS = (prefix: string): CSSType => {
  const CSS = newBEM(prefix) as CSSType;
  CSS.visible = (visible) => CSS.M(visible ? "visible" : "hidden");
  CSS.expanded = (expanded) => CSS.M(expanded ? "expanded" : "collapsed");
  CSS.loc = (l) => CSS.M("location", location.construct(l));
  CSS.disabled = (disabled) => disabled === true && CSS.M("disabled");
  CSS.align = (position) => CSS.M(position);
  CSS.dir = (dir) => dir != null && CSS.M("direction", direction.construct(dir));
  CSS.sharp = (sharp) => !(sharp === false) && CSS.M("sharp");
  CSS.rounded = (rounded) => !(rounded === false) && CSS.M("rounded");
  CSS.bordered = (loc) => {
    if (typeof loc === "boolean") return loc && CSS.M("bordered");
    return loc != null ? CSS.M(`bordered-${loc.toString()}`) : CSS.M("bordered");
  };
  CSS.selected = (selected) => selected && CSS.M("selected");
  CSS.altColor = (secondary) => secondary && CSS.M("alt-color");
  CSS.editable = (editable) => editable && CSS.M("editable");
  CSS.noSelect = CSS.M("no-select");
  CSS.applyVars = applyCSSVars;
  CSS.removeVars = removeCSSVars;
  CSS.newGridBuilder = (prefix?: string) => new CSSGridBuilder(prefix);
  CSS.dropRegion = (active) => active && CSS.B("haul-drop-region");
  CSS.px = (value: number) => `${value}px`;
  CSS.inheritDims = (inherit = true) => inherit && CSS.M("inherit-dims");
  CSS.shade = ((value) => value != null && CSS.M(`shade-${value}`)) as CSSType["shade"];
  CSS.colorVar = (value) => {
    if (value == null || value === false) return undefined;
    if (typeof value === "number") return `var(--${prefix}-gray-l${value})`;
    return color.cssString(value);
  };
  CSS.levelSizeVar = (value) => `var(--${prefix}-${value}-size)`;
  CSS.level = (level) => CSS.M(`level-${level}`);
  return CSS;
};

export const CSS = newCSS("pluto");
