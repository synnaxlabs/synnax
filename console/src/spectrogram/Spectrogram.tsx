// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Eraser,
  Menu as PMenu,
  Spectrogram as PlutoSpectrogram,
  type telem,
} from "@synnaxlabs/pluto";
import { TimeSpan } from "@synnaxlabs/x";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";
import { useSelect } from "@/spectrogram/selectors";
import { setActiveToolbarTab } from "@/spectrogram/slice";

export const Spectrogram: Layout.Renderer = ({ layoutKey, visible }) => {
  const vis = useSelect(layoutKey);
  const dispatch = useDispatch();
  const menuProps = PMenu.useContextMenu();
  if (vis == null) return null;

  const telemSpec: telem.SeriesSourceSpec =
    vis.channel !== 0
      ? {
          type: "dynamic-series-source",
          variant: "source" as const,
          valueType: "series" as const,
          props: {
            channel: vis.channel,
            timeSpan: TimeSpan.seconds(30),
          },
        }
      : {
          type: "noop-series",
          variant: "source" as const,
          valueType: "series" as const,
          props: {},
        };

  const handleDoubleClick = (): void => {
    dispatch(setActiveToolbarTab({ key: layoutKey, tab: "data" }));
  };

  return (
    <div
      style={{ height: "100%", width: "100%", padding: "2rem" }}
      className={menuProps.className}
      onDoubleClick={handleDoubleClick}
    >
      <PMenu.ContextMenu
        {...menuProps}
        menu={() => (
          <PMenu.Menu level="small" gap="small">
            <Layout.MenuItems layoutKey={layoutKey} />
          </PMenu.Menu>
        )}
      >
        <Eraser.Eraser>
          <PlutoSpectrogram.SpectrogramC
            aetherKey={layoutKey}
            telem={telemSpec}
            sampleRate={vis.sampleRate}
            fftSize={vis.fftSize}
            windowFunction={vis.windowFunction}
            overlap={vis.overlap}
            colorMap={vis.colorMap}
            dbMin={vis.dbMin}
            dbMax={vis.dbMax}
            freqMin={vis.freqMin}
            freqMax={vis.freqMax}
            visible={visible}
          />
        </Eraser.Eraser>
      </PMenu.ContextMenu>
    </div>
  );
};
