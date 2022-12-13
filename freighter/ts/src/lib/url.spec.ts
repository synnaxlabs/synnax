import test from 'ava';

import URL from './url';

test('URL - child', (t) => {
  const endpoint = new URL({
    host: 'localhost',
    port: 8080,
    protocol: 'http',
    pathPrefix: 'api',
  });
  t.is(endpoint.child('test').toString(), 'http://localhost:8080/api/test');
});

test('URL - child with trailing slash', (t) => {
  const endpoint = new URL({
    host: 'localhost',
    port: 8080,
    protocol: 'http',
    pathPrefix: 'api',
  });
  const child = endpoint.child('test/');
  t.is(child.toString(), 'http://localhost:8080/api/test');
});

test('URL - replacing protocol', (t) => {
  const endpoint = new URL({
    host: 'localhost',
    port: 8080,
    protocol: 'http',
    pathPrefix: 'api',
  });
  t.is(
    endpoint.child('test').replace({ protocol: 'https' }).toString(),
    'https://localhost:8080/api/test'
  );
});
