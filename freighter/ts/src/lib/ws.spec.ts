import test from 'ava';

import { MsgPackEncoderDecoder } from './encoder';
import Endpoint from './endpoint';
import { BaseTypedError, EOF, registerError, TypedError } from './errors';
import { WebSocketClient } from './ws';

const ENDPOINT = new Endpoint({
  host: '127.0.0.1',
  port: 8080,
});

type Message = {
  id?: number;
  message?: string;
};

class MyCustomError extends BaseTypedError {
  code: number;

  constructor(message: string, code: number) {
    super(message, 'integration.error');
    this.code = code;
  }
}

const encodeTestError = (err: TypedError): string => {
  if (!(err instanceof MyCustomError)) {
    throw new Error('Unexpected error type');
  }
  return `${err.code},${err.message}`;
};

const decodeTestError = (encoded: string): TypedError => {
  const [code, message] = encoded.split(',');
  return new MyCustomError(message, parseInt(code, 10));
};

registerError({
  type: 'integration.error',
  encode: encodeTestError,
  decode: decodeTestError,
});

test('basic exchange', async (t) => {
  // Should exchange ten echo messages that increment the ID.
  const client = new WebSocketClient(new MsgPackEncoderDecoder(), ENDPOINT);
  const stream = await client.stream<Message, Message>('ws/echo');

  for (let i = 0; i < 10; i++) {
    stream.send({ id: i, message: 'hello' });
    const [response, error] = await stream.receive();
    t.is(error, undefined);
    t.is(response?.id, i + 1);
    t.is(response?.message, 'hello');
  }
  stream.closeSend();
  const [response, error] = await stream.receive();
  t.deepEqual(error, new EOF());
  t.is(response, undefined);
});

test('receive message after close', async (t) => {
  // Should exchange ten echo messages that increment the ID.
  const client = new WebSocketClient(new MsgPackEncoderDecoder(), ENDPOINT);
  const stream = await client.stream<Message, Message>(
    'ws/sendMessageAfterClientClose'
  );

  await stream.closeSend();
  let [response, error] = await stream.receive();
  t.is(error, undefined);
  t.is(response?.id, 0);
  t.is(response?.message, 'Close Acknowledged');
  [response, error] = await stream.receive();
  t.deepEqual(error, new EOF());
});

test('receive error', async (t) => {
  // Should exchange ten echo messages that increment the ID.
  const client = new WebSocketClient(new MsgPackEncoderDecoder(), ENDPOINT);
  const stream = await client.stream<Message, Message>(
    'ws/receiveAndExitWithErr'
  );

  stream.send({ id: 0, message: 'hello' });

  const [response, error] = await stream.receive();
  t.deepEqual(error, new MyCustomError('unexpected error', 1));
  t.is(response, undefined);
});
