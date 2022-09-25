import { ZodSchema } from 'zod';

/**
 * Interface for an entity that receives a stream of responses.
 */
export interface StreamReceiver<RS> {
  /**
   * Receives a response from the stream. It's not safe to call receive
   * concurrently.
   *
   *  @returns freighter.EOF: if the server closed the stream nominally.
   *  @returns Error: if the server closed the stream abnormally,
   *  returns the error the server returned.
   *  @raises Error: if the transport fails.
   */
  receive(): Promise<[RS | undefined, Error | undefined]>;
}

/**
 * Interface for an entity that sends a stream of requests.
 */
export interface StreamSender<RQ> {
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
  send(req: RQ): Error | undefined;
}

/**
 * Extension of the StreamSender interface that allows the client to close the sending
 * direction of the stream when finished issuing requrest.
 */
export interface StreamSenderCloser<RQ> extends StreamSender<RQ> {
  /**
  * Lets the server know no more messages will be sent. If the client attempts
  * to call send() after calling closeSend(), a freighter.StreamClosed
  * exception will be raised. close_send is idempotent. If the server has
  * already closed the stream, close_send will do nothing.

  * After calling close_send, the client is responsible for calling receive()
  * to successfully receive the server's acknowledgement.
   */
  closeSend(): void;
}

/**
 * Interface for a bidirectional stream between a client and a server.
 */
export interface Stream<RQ, RS>
  extends StreamSenderCloser<RQ>,
    StreamReceiver<RS> {}

/**
 * Interface for a bidirectional stream between a client and a server.
 */
export interface StreamClient {
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
  stream<RQ, RS>(
    target: string,
    reqSchema: ZodSchema<RQ>,
    resSchema: ZodSchema<RS>
  ): Promise<Stream<RQ, RS>>;
}
