// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Button, Channel, Icon, Nav, Progress, Text } from "@synnaxlabs/pluto";
import { type NumericTimeRange, TimeRange } from "@synnaxlabs/x";
import { useCallback, useState } from "react";

import { useDownload } from "@/csv/useDownload";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";

export interface SelectChannelsModalArgs extends Modals.BaseArgs<void> {
  timeRanges: NumericTimeRange[];
  fileName: string;
}

const SELECT_MODAL_LAYOUT_TYPE = "channel.select";

export interface PromptChannels extends Modals.Prompt<void, SelectChannelsModalArgs> {}

const createBaseReturns = Modals.createBase<void, SelectChannelsModalArgs>(
  "Download.CSV",
  SELECT_MODAL_LAYOUT_TYPE,
  ({ value: { timeRanges, fileName }, onFinish }) => {
    const [channels, setChannels] = useState<channel.Keys>([]);
    const [percentDownloaded, setPercentDownloaded] = useState(0);
    console.log(percentDownloaded);
    const downloadCSV = useDownload();
    const handleFinish = () =>
      downloadCSV({
        timeRanges: timeRanges.map((tr) => new TimeRange(tr)),
        keys: channels,
        fileName,
        afterDownload: onFinish,
        onPercentDownloadedChange: (p: number) => {
          console.log("calling percent downloaded", p);
          setPercentDownloaded(p);
        },
      });
    const footer = (
      <>
        <Triggers.SaveHelpText action="Download" />
        <Nav.Bar.End x align="center">
          <Button.Button
            variant="filled"
            disabled={channels.length === 0}
            onClick={handleFinish}
            trigger={Triggers.SAVE}
          >
            <Icon.Download />
          </Button.Button>
        </Nav.Bar.End>
      </>
    );
    return (
      <Modals.ModalContentLayout footer={footer} gap="huge">
        <Text.Text level="h3">Select Channels to Download</Text.Text>
        <Channel.SelectMultiple value={channels} onChange={setChannels} full="x" />
        {percentDownloaded > 0 && <Progress.Progress value={percentDownloaded} />}
      </Modals.ModalContentLayout>
    );
  },
);

export const SelectModal = createBaseReturns[1];

export const useSelectModal = (): ((
  args: SelectChannelsModalArgs,
  layoutOverrides?: Modals.LayoutOverrides,
) => Promise<void | null>) => {
  const select = createBaseReturns[0]();
  return useCallback(
    (args: SelectChannelsModalArgs) => select(args, { icon: "Range" }),
    [select],
  );
};
