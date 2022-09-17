import test from 'ava';

import { JSONEncoderDecoder } from './encoder';
import Endpoint from './endpoint';
import HTTPClient from './http';

const ENDPOINT = new Endpoint({
  host: '127.0.0.1',
  port: 8080,
  pathPrefix: 'http',
});

test('[http] - post echo', async (t) => {
  const client = new HTTPClient(ENDPOINT, new JSONEncoderDecoder());
  const post = client.post();
  const [response, error] = await post.send('/echo', {
    id: 1,
    message: 'hello',
  });
  t.is(error, undefined);
  t.deepEqual(response, { id: 2, message: 'hello' });
});

test('[http] - get echo', async (t) => {
  const client = new HTTPClient(ENDPOINT, new JSONEncoderDecoder());
  const get = client.get();
  const [response, error] = await get.send('/echo', {
    id: 1,
    message: 'hello',
  });
  t.is(error, undefined);
  t.deepEqual(response, { id: 2, message: 'hello' });
});

test('[http] - get not found', async (t) => {
  const client = new HTTPClient(ENDPOINT, new JSONEncoderDecoder());
  const get = client.get();
  const [response, error] = await get.send('/not-found', {});
  t.is(error?.message, 'Request failed with status code 404');
  t.is(response, undefined);
});

test('[http] - post not found', async (t) => {
  const client = new HTTPClient(ENDPOINT, new JSONEncoderDecoder());
  const post = client.post();
  const [response, error] = await post.send('/not-found', {});
  t.is(error?.message, 'Request failed with status code 404');
  t.is(response, undefined);
});
