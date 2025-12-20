// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type z } from "zod";

import { type Transport } from "@/transport";

/**
 * Interface for an entity that receives a stream of responses.
 */
export interface StreamReceiver<RS extends z.ZodType> {
  /**
   * Receives a response from the stream. It's not safe to call receive
   * concurrently.
   *
   *  @returns freighter.EOF: if the server closed the stream nominally.
   *  @returns Error: if the server closed the stream abnormally,
   *  returns the error the server returned.
   *  @raises Error: if the transport fails.
   */
  receive: () => Promise<[z.infer<RS>, null] | [null, Error]>;

  /**
   * @returns true if the stream has received a response
   */
  received: () => boolean;
}

/**
 * Interface for an entity that sends a stream of requests.
 */
export interface StreamSender<RQ extends z.ZodType> {
  /**
  * Sends a request to the stream. It is not safe to call send concurrently
  * with closeSend or send.

  * @param req -  the request to send.
  * @returns freighter.EOF: if the server closed the stream. The caller
  * can discover the error returned by the server by calling receive().
  * @returns undefined: if the message was sent successfully.
  * @raises freighter.StreamClosed: if the client called close_send()
  * @raises Error: if the transport fails.
  */
  send: (req: z.input<RQ> | z.infer<RQ>) => Error | null;
}

/**
 * Extension of the StreamSender interface that allows the client to close the sending
 * direction of the stream when finished issuing requrest.
 */
export interface StreamSenderCloser<RQ extends z.ZodType> extends StreamSender<RQ> {
  /**
  * Lets the server know no more messages will be sent. If the client attempts
  * to call send() after calling closeSend(), a freighter.StreamClosed
  * exception will be raised. close_send is idempotent. If the server has
  * already closed the stream, close_send will do nothing.

  * After calling close_send, the client is responsible for calling receive()
  * to successfully receive the server's acknowledgement.
   */
  closeSend: () => void;
}

/**
 * Interface for a bidirectional stream between a client and a server.
 */
export interface Stream<RQ extends z.ZodType, RS extends z.ZodType = RQ>
  extends StreamSenderCloser<RQ>, StreamReceiver<RS> {}

/**
 * Interface for a bidirectional stream between a client and a server.
 */
export interface StreamClient extends Transport {
  /**
   * Dials the target and returns a stream that can be used to issue requests
   * and receive responses
   *
   * @param target - The target to dial. In some implementations, this may be
   * an endpoint path, or in others, a complete hostname or URL.
   * @param reqSchema - The schema for the request type. This is used to
   * validate the request before sending it.
   * @param resSchema - The schema for the response type. This is used to
   * validate the response before returning it.
   */
  stream: <RQ extends z.ZodType, RS extends z.ZodType = RQ>(
    target: string,
    reqSchema: RQ,
    resSchema: RS,
  ) => Promise<Stream<RQ, RS>>;
}
