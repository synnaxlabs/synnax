// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import ErrorStackParser from "error-stack-parser";
import { type RawSourceMap, SourceMapConsumer } from "source-map-js";

type StackFrame = ReturnType<typeof ErrorStackParser.parse>[number];

export interface ResolvedStack {
  stack: string;
  componentStack: string | null;
}

// One parsed consumer per script URL, shared by both legs, every crash, and the whole
// session, so each script and sourcemap is fetched and decoded exactly once. A
// deliberate module-level cache: the crash screen cannot rely on injected state
// because the tree around it has already failed. A script without a usable map caches
// null and its frames render unresolved.
const consumers: Record<string, Promise<SourceMapConsumer | null>> = {};

const fetchText = async (url: string): Promise<string> => {
  const res = await fetch(url);
  if (!res.ok) throw new Error(`HTTP ${res.status} retrieving ${url}`);
  return await res.text();
};

const SOURCE_MAPPING_URL_REGEX = /\/\/[#@] ?sourceMappingURL=([^\s'"]+)\s*$/gm;

// A concatenated bundle can carry several sourceMappingURL comments; the last one
// governs the file.
const findSourceMappingURL = (source: string): string | null => {
  let last: string | null = null;
  for (const match of source.matchAll(SOURCE_MAPPING_URL_REGEX)) last = match[1];
  return last;
};

const readRawMap = async (mapURL: string): Promise<string> => {
  if (!mapURL.startsWith("data:")) return await fetchText(mapURL);
  const comma = mapURL.indexOf(",");
  const payload = mapURL.slice(comma + 1);
  if (mapURL.slice(0, comma).includes("base64")) return atob(payload);
  return decodeURIComponent(payload);
};

const loadConsumer = async (scriptURL: string): Promise<SourceMapConsumer | null> => {
  const source = await fetchText(scriptURL);
  const mappingURL = findSourceMappingURL(source);
  if (mappingURL == null) return null;
  const raw = await readRawMap(new URL(mappingURL, scriptURL).toString());
  // Yield a macrotask so the crash screen paints before the one-time decode below.
  await new Promise((resolve) => setTimeout(resolve, 0));
  const consumer = new SourceMapConsumer(JSON.parse(raw) as RawSourceMap);
  // The mappings decode is lazy; pay it here, inside the cached promise, so every
  // per-frame query afterward is a sub-millisecond lookup.
  consumer.originalPositionFor({ line: 1, column: 0 });
  return consumer;
};

const consumerFor = (scriptURL: string): Promise<SourceMapConsumer | null> =>
  (consumers[scriptURL] ??= loadConsumer(scriptURL).catch((cause: unknown) => {
    console.warn(`resolveStack: no usable sourcemap for ${scriptURL}`, cause);
    return null;
  }));

/**
 * Resolves the source-map-mapped form of an Error's stack and (optionally) its React
 * component stack into clean, human-readable text. Each leg is resolved independently:
 * if a leg cannot be resolved (e.g. source maps unavailable, parser cannot recognize
 * the stack format), that leg degrades to its raw input and a warning is surfaced via
 * console.warn, while the other leg's resolved output is preserved. Silently swallowing
 * would mask real deployment problems (missing maps, broken middleware, etc.), so
 * failures are always logged.
 */
export const resolveStack = async (
  error: Error,
  componentStack: string | null,
): Promise<ResolvedStack> => {
  const [stack, resolvedComponentStack] = await Promise.all([
    resolveFromStack(error).catch((cause: unknown) => {
      console.warn("resolveStack: failed to resolve error stack", cause);
      return error.stack ?? "";
    }),
    componentStack != null
      ? resolveFromComponentStack(componentStack).catch((cause: unknown) => {
          console.warn("resolveStack: failed to resolve component stack", cause);
          return componentStack;
        })
      : null,
  ]);
  return { stack, componentStack: resolvedComponentStack };
};

const resolveFromStack = async (error: Error): Promise<string> => {
  const frames = await Promise.all(ErrorStackParser.parse(error).map(resolveFrame));
  return frames.join("\n");
};

const resolveFromComponentStack = async (componentStack: string): Promise<string> => {
  const synthetic = new Error();
  synthetic.stack = componentStack;
  const frames = await Promise.all(ErrorStackParser.parse(synthetic).map(resolveFrame));
  return frames.join("\n");
};

const resolveFrame = async (frame: StackFrame): Promise<string> => {
  const { fileName, lineNumber, columnNumber } = frame;
  if (fileName == null || lineNumber == null || columnNumber == null)
    return formatRawFrame(frame);
  const consumer = await consumerFor(fileName);
  if (consumer == null) return formatRawFrame(frame);
  // Stack trace columns are 1-based; sourcemap generated columns are 0-based.
  const pos = consumer.originalPositionFor({
    line: lineNumber,
    column: Math.max(columnNumber - 1, 0),
  });
  if (pos.source == null || pos.line == null) return formatRawFrame(frame);
  const name = pos.name ?? frame.functionName ?? "<anonymous>";
  const column = pos.column == null ? "" : `:${pos.column + 1}`;
  return `  at ${name} (${cleanFileName(pos.source)}:${pos.line}${column})`;
};

const formatRawFrame = (frame: StackFrame): string => {
  const name = frame.functionName ?? "<anonymous>";
  const location = formatRawLocation(frame);
  return location != null ? `  at ${name} (${location})` : `  at ${name}`;
};

const formatRawLocation = (frame: StackFrame): string | null => {
  if (frame.fileName == null) return null;
  const file = cleanFileName(frame.fileName);
  if (frame.lineNumber == null) return file;
  if (frame.columnNumber == null) return `${file}:${frame.lineNumber}`;
  return `${file}:${frame.lineNumber}:${frame.columnNumber}`;
};

// Strip leading "../" segments and "webpack:///" / "webpack-internal:///" prefixes that
// source maps sometimes emit, keeping frames compact.
const cleanFileName = (file: string): string =>
  file.replace(/^webpack(?:-internal)?:\/\/\/?/, "").replace(/^(?:\.\.\/)+/, "");
