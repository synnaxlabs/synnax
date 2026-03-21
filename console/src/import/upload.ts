// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { URL } from "@synnaxlabs/x";

import { type Cluster } from "@/cluster/slice";

export type ConflictStatus = "new" | "conflict" | "identical";
export type ConflictPolicy = "skip" | "replace";

export interface AnalysisItem {
  type: string;
  name: string;
  archive_key: string;
  status: ConflictStatus;
  existing_key?: string;
  details?: string;
  parent_name?: string;
  data_type?: string;
  disabled?: boolean;
}

export interface AnalyzeResponse {
  session_id: string;
  items: AnalysisItem[];
}

export interface ImportRequest {
  session_id: string;
  default_policy: ConflictPolicy;
  overrides: Record<string, ConflictPolicy>;
}

export interface ImportResponse {
  imported: number;
  replaced: number;
  skipped: number;
  identical: number;
  errors: string[];
}

const buildURL = (cluster: Cluster): URL =>
  new URL({
    host: cluster.host,
    port: Number(cluster.port),
    protocol: cluster.secure ? "https" : "http",
    pathPrefix: "api/v1",
  });

const authHeaders = (client: Client): Record<string, string> => {
  if (!client.auth.authenticated)
    throw new Error("Client is not authenticated");
  return { Authorization: `Bearer ${client.auth.token}` };
};

export interface AnalyzeParams {
  client: Client;
  cluster: Cluster;
  fileBytes: Uint8Array;
}

export const analyzeBackup = async ({
  client,
  cluster,
  fileBytes,
}: AnalyzeParams): Promise<AnalyzeResponse> => {
  const url = buildURL(cluster);
  const response = await fetch(url.child("import/analyze").toString(), {
    method: "POST",
    headers: {
      "Content-Type": "application/octet-stream",
      ...authHeaders(client),
    },
    body: fileBytes.buffer as ArrayBuffer,
  });
  if (!response.ok) {
    const body = await response.text();
    throw new Error(`Analyze failed (${response.status}): ${body}`);
  }
  const data = (await response.json()) as AnalyzeResponse;
  data.items ??= [];
  return data;
};

export interface ExecuteImportParams {
  client: Client;
  cluster: Cluster;
  request: ImportRequest;
}

export const executeImport = async ({
  client,
  cluster,
  request,
}: ExecuteImportParams): Promise<ImportResponse> => {
  const url = buildURL(cluster);
  const response = await fetch(url.child("import").toString(), {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...authHeaders(client),
    },
    body: JSON.stringify(request),
  });
  if (!response.ok) {
    const body = await response.text();
    throw new Error(`Import failed (${response.status}): ${body}`);
  }
  const data = (await response.json()) as ImportResponse;
  data.errors ??= [];
  return data;
};
