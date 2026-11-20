// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { type Listing } from "@/platform/file-browser/list";
import { inside } from "@/platform/file-browser/path";

type Kind = "root" | "home" | "secrets";

/** A folder that the sidebar offers: a root, the home folder, or the secrets folder. */
export interface Place {
  kind: Kind;
  label: string;
  path: string;
}

/** @returns the places in the top listing of a Driver, in its order. */
export const placesOf = (top: Listing | undefined): Place[] =>
  top?.entries.map(({ name, path }): Place => {
    if (top.secretsDir !== "" && path === top.secretsDir)
      return { kind: "secrets", label: "Secrets folder", path };
    if (name === path)
      return { kind: "root", label: path === "/" ? "File system" : path, path };
    return { kind: "home", label: "Home folder", path };
  }) ?? [];

const ICONS: Record<Kind, Icon.FC> = {
  root: Icon.Computer,
  home: Icon.User,
  secrets: Icon.Secret,
};

export interface PlacesProps {
  places: Place[];
  /** The open folder. The place that holds it most closely shows as active. */
  path: string;
  onGo: (path: string) => void;
}

/** The sidebar of the places of a Driver. */
export const Places = ({ places, path, onGo }: PlacesProps): ReactElement => {
  const active = places
    .filter((p) => inside(path, p.path))
    .sort((a, b) => b.path.length - a.path.length)[0]?.path;
  return (
    <Flex.Box y empty className={CSS.BE("file-browser", "places")}>
      <Text.Text level="small" color={8} className={CSS.BE("file-browser", "heading")}>
        Places
      </Text.Text>
      {places.map(({ kind, label, path: placePath }) => {
        const PlaceIcon = ICONS[kind];
        return (
          <Button.Button
            key={placePath}
            variant="text"
            size="small"
            justify="start"
            gap="medium"
            tooltip={placePath}
            tooltipLocation="right"
            className={CSS.cls(
              CSS.BE("file-browser", "place"),
              active === placePath && CSS.M("active"),
            )}
            onClick={() => onGo(placePath)}
          >
            <PlaceIcon />
            <Text.Text el="span" overflow="fade">
              {label}
            </Text.Text>
          </Button.Button>
        );
      })}
    </Flex.Box>
  );
};
