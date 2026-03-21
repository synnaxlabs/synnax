// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { type Status } from "@synnaxlabs/pluto";
import { URL } from "@synnaxlabs/x";

import { type Cluster } from "@/cluster/slice";
import { downloadStream } from "@/runtime/download";

export interface BackupExportRequest {
  workspace_keys?: string[];
  user_keys?: string[];
  device_keys?: string[];
  task_keys?: string[];
  range_keys?: string[];
  channel_keys?: string[];
}

export interface DownloadBackupParams {
  client: Client;
  cluster: Cluster;
  request: BackupExportRequest;
  addStatus: Status.Adder;
  onDownloadStart?: () => void;
}

export const downloadBackup = async ({
  client,
  cluster,
  request,
  addStatus,
  onDownloadStart,
}: DownloadBackupParams): Promise<void> => {
  if (!client.auth.authenticated)
    throw new Error("Client is not authenticated");

  const url = new URL({
    host: cluster.host,
    port: Number(cluster.port),
    protocol: cluster.secure ? "https" : "http",
    pathPrefix: "api/v1",
  });

  const response = await fetch(url.child("export").toString(), {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${client.auth.token}`,
    },
    body: JSON.stringify(request),
  });

  if (!response.ok) {
    const body = await response.text();
    throw new Error(`Export failed (${response.status}): ${body}`);
  }

  const stream = response.body;
  if (stream == null) throw new Error("Export response has no body");

  await downloadStream({
    stream,
    name: "backup",
    extension: "sy",
    addStatus,
    onDownloadStart,
  });
};
