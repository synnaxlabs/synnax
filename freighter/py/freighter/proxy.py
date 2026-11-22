#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from urllib.parse import urlsplit
from urllib.request import getproxies, proxy_bypass

from freighter.url import URL


def resolve(url: URL, *, secure: bool) -> str | None:
    """Picks the proxy for url as urllib does: HTTPS_PROXY when secure, else
    HTTP_PROXY, and none when NO_PROXY matches url. With no proxy variables set, macOS
    and Windows give the system proxy settings instead.

    :returns: the proxy URL, with http:// added when it has no scheme, or None.
    :raises ValueError: if the proxy scheme is not http or https.
    """
    if proxy_bypass(f"{url.host}:{url.port}"):
        return None
    key = "https" if secure else "http"
    proxy = getproxies().get(key)
    if proxy is None:
        return None
    if "://" not in proxy:
        proxy = f"http://{proxy}"
    scheme = urlsplit(proxy).scheme
    if scheme not in ("http", "https"):
        raise ValueError(
            f"{key.upper()}_PROXY has scheme {scheme!r}, but only http and https "
            "proxies are supported"
        )
    return proxy
