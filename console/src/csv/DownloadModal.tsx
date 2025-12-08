// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import {
  Button,
  Channel,
  Flex,
  Icon,
  Nav,
  Progress,
  type Select,
  Text,
} from "@synnaxlabs/pluto";
import { type CrudeTimeRange, TimeRange } from "@synnaxlabs/x";
import { useState } from "react";

import { useDownload } from "@/csv/useDownload";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";

export interface DownloadModalArgs extends Modals.BaseArgs<void> {
  timeRanges: CrudeTimeRange[];
  name: string;
}

export const DOWNLOAD_MODAL_LAYOUT_TYPE = "downloadCSV";
const NON_VIRTUAL_CHANNEL_QUERY: Partial<Channel.RetrieveMultipleQuery> = {
  virtual: false,
};
const CHANNEL_SELECT_TRIGGER_PROPS: Select.MultipleTriggerProps<channel.Key> = {
  placeholder: "Select channels to download",
};

export interface PromptDownload extends Modals.Prompt<void, DownloadModalArgs> {}

export const [useDownloadModal, DownloadModal] = Modals.createBase<
  void,
  DownloadModalArgs
>(
  "Download.CSV",
  DOWNLOAD_MODAL_LAYOUT_TYPE,
  ({ value: { timeRanges, name }, onFinish }) => {
    const [channels, setChannels] = useState<channel.Keys>([]);
    const [percentDownloaded, setPercentDownloaded] = useState(0);
    const downloadCSV = useDownload();
    const handleFinish = () =>
      downloadCSV({
        timeRanges: timeRanges.map((tr) => new TimeRange(tr)),
        keys: channels,
        fileName: name,
        afterDownload: onFinish,
        onPercentDownloadedChange: setPercentDownloaded,
      });
    const footer =
      percentDownloaded > 0 ? (
        <Flex.Box grow style={{ paddingLeft: "5rem", paddingRight: "5rem" }} x>
          <Progress.Progress value={percentDownloaded} />
        </Flex.Box>
      ) : (
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
              Download
            </Button.Button>
          </Nav.Bar.End>
        </>
      );
    return (
      <Modals.ModalContentLayout footer={footer} gap="huge">
        <Text.Text level="h4" weight={450}>
          Export data for {name} to a CSV
        </Text.Text>
        <Flex.Box y full="x">
          <Channel.SelectMultiple
            value={channels}
            onChange={setChannels}
            initialQuery={NON_VIRTUAL_CHANNEL_QUERY}
            triggerProps={CHANNEL_SELECT_TRIGGER_PROPS}
            full="x"
          />
        </Flex.Box>
      </Modals.ModalContentLayout>
    );
  },
);
