#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import time

import requests
from paralell import process_products
import threading
import warnings

headers = {
    "Client_id": "1fcca773908c4e6da0500a60ea393e83",
    "Client_secret": "9EC7AA4494614C25AE57f022Bc6f7Bac",
    "Referer": "https://www.ni.com/",
}

base_url = "https://www.ni.com/site-search/api/results?"

lock = threading.Lock()
updated = list()


def find_better_spec(id, prod):
    global counter
    url = prod["productSpecs"]
    if "search" in url:
        url = url[url.find("?") + 1 :]
        res = requests.get(base_url + url, headers=headers)
        if res.status_code != 200:
            warnings.warn(
                f"Failed to get specs for {prod['productTitle']} {res.status_code}"
            )
            if res.status_code == 503:
                time.sleep(1)
                return find_better_spec(id, prod)
            return prod
        j = res.json()
        res_list = j["searchResultList"]
        for item in res_list:
            if "specs.html" in item["uri"]:
                prod["productSpecs"] = item["uri"]
                with lock:
                    updated.append(prod["tileLabel"])
                break
    return prod


products = json.load(open("out/products.json", "r"))
products = process_products(products, find_better_spec)
print(f"Updated {updated} products")
json.dump(products, open("out/products.json", "w"))
