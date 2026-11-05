// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type VideoFrame as EncodedFrame } from "@synnaxlabs/x";

import { h264 } from "@/video/aether/h264";

/** Props for {@link Decoder}. */
export interface DecoderProps {
  /** Receives each decoded picture. The receiver owns the picture and must close it. */
  onPicture: (picture: VideoFrame) => void;
  /** Receives each error. The decoder then waits for the next keyframe. */
  onError: (error: Error) => void;
  /**
   * Builds the WebCodecs decoder.
   * @default the VideoDecoder constructor
   */
  createDecoder?: (init: VideoDecoderInit) => VideoDecoder;
}

// Past this many queued frames, a slow decoder drops frames up to the next keyframe, so
// it falls back to the newest picture instead of further behind.
const MAX_QUEUE = 30;

const bytesEqual = (a: Uint8Array, b: Uint8Array): boolean =>
  a.length === b.length && a.every((v, i) => v === b[i]);

/**
 * Decodes the H.264 frames of one video channel in order. A frame that does not
 * continue the decoded frames waits for the next keyframe, so a gap never shows a
 * broken picture.
 */
export class Decoder {
  private decoder: VideoDecoder | null = null;
  private sps: Uint8Array | null = null;
  // expected is the alignment that continues the decoded frames, or null while the
  // decoder waits for a keyframe.
  private expected: bigint | null = null;
  private timestamp = 0;

  constructor(private readonly props: DecoderProps) {}

  /**
   * Decodes the frame at the alignment of its sample. The picture carries the
   * timestamp, which counts up from zero when absent.
   */
  decode(frame: EncodedFrame, alignment: bigint, timestamp?: number): void {
    try {
      if (frame.keyframe) this.configure(frame.data);
      else if (this.expected !== alignment) return;
      const decoder = this.decoder!;
      if (!frame.keyframe && decoder.decodeQueueSize > MAX_QUEUE) {
        this.expected = null;
        return;
      }
      decoder.decode(
        new EncodedVideoChunk({
          type: frame.keyframe ? "key" : "delta",
          timestamp: timestamp ?? this.timestamp++,
          data: frame.data,
        }),
      );
      this.expected = alignment + 1n;
    } catch (e) {
      this.fail(this.decoder, e);
    }
  }

  /** The frames that wait to decode. */
  get pending(): number {
    return this.decoder?.decodeQueueSize ?? 0;
  }

  /** Releases the WebCodecs decoder. A later keyframe opens a new one. */
  close(): void {
    if (this.decoder?.state !== "closed") this.decoder?.close();
    this.decoder = null;
    this.expected = null;
  }

  // Configures the decoder for the stream of the keyframe. With no description, the
  // decoder reads Annex B. A new SPS configures it again, since the size or profile can
  // change. WebKit then drops the frames still queued from the old stream.
  private configure(accessUnit: Uint8Array): void {
    const sps = h264.spsOf(h264.split(accessUnit));
    if (sps == null) throw new Error("video keyframe has no SPS");
    if (this.decoder != null && this.sps != null && bytesEqual(this.sps, sps)) return;
    if (this.decoder == null) {
      const { onPicture, createDecoder = (init) => new VideoDecoder(init) } =
        this.props;
      const decoder = createDecoder({
        output: onPicture,
        error: (e) => this.fail(decoder, e),
      });
      this.decoder = decoder;
    }
    this.sps = sps.slice();
    this.decoder.configure({ codec: h264.codecOf(sps), optimizeForLatency: true });
  }

  private fail(decoder: VideoDecoder | null, e: unknown): void {
    if (decoder !== this.decoder) return;
    this.close();
    this.props.onError(e instanceof Error ? e : new Error(String(e)));
  }
}
