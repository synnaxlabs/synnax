#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from urllib.request import getproxies, proxy_bypass

from freighter.url import URL


def resolve(url: URL, *, secure: bool) -> str | None:
    """:returns: the proxy for url from HTTPS_PROXY or HTTP_PROXY, as urllib picks it,
    or None when there is none or NO_PROXY matches url.
    """
    if proxy_bypass(f"{url.host}:{url.port}"):
        return None
    return getproxies().get("https" if secure else "http")
