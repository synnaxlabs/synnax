#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
from paralell import process_products
import requests
from html.parser import HTMLParser
import threading
import time

headers = {
    "Client_id": "1fcca773908c4e6da0500a60ea393e83",
    "Client_secret": "9EC7AA4494614C25AE57f022Bc6f7Bac",
    "Referer": "https://www.ni.com/",
}


class PinoutFinder(HTMLParser):
    after_pinout: bool
    pinouts: list

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.after_pinout = False
        self.pinouts = list()

    def handle_starttag(self, tag, attrs, **kwargs):
        if tag == "img" and self.after_pinout:
            attrs = dict(attrs)
            # check if the latest pinout has an 'img' key
            if len(self.pinouts) > 0 and "img" not in self.pinouts[-1]:
                self.pinouts[-1]["img"] = attrs["src"]
            else:
                self.pinouts.append({"img": attrs["src"]})

    def handle_data(self, data):
        if "Pin" in data and self.after_pinout:
            self.pinouts.append({"title": data})

        if "Pinout" in data:
            self.after_pinout = True


def find_pinouts_in_html(html):
    parser = PinoutFinder()
    parser.feed(html)
    return parser.pinouts


def find_pinouts_at_url(url):
    response = requests.get(url, headers=headers)
    if response.status_code == 200:
        return find_pinouts_in_html(response.json()["topic_html"])
    if response.status_code == 503:
        time.sleep(1)
        return find_pinouts_at_url(url)
    return []


def from_specs(product) -> list:
    specs_url = product["productSpecs"]
    if "search" not in specs_url and "specs.html" in specs_url:
        specs_url_ext = specs_url[specs_url.find("bundle") + 7 :]
        base_url = "https://docs-be.ni.com/api/bundle/"
        response = requests.get(base_url + specs_url_ext, headers=headers)
        return find_pinouts_in_html(response.json()["topic_html"])
    return []


def from_get_started_pinout(product) -> list:
    label_lower = product["tileLabel"].lower()
    url = f"https://docs-be.ni.com/api/bundle/{label_lower}-getting-started/page/pinout.html"
    return find_pinouts_at_url(url)


def from_get_started_overview(product) -> list:
    label_lower = product["tileLabel"].lower()
    url = f"https://docs-be.ni.com/api/bundle/{label_lower}-getting-started/page/overview.html"
    return find_pinouts_at_url(url)


def from_feature_pinout(product) -> list:
    label_lower = product["tileLabel"].lower()
    url = f"https://docs-be.ni.com/api/bundle/{label_lower}-feature/page/pinout.html"
    return find_pinouts_at_url(url)


def from_feature_connecting(product) -> list:
    label_lower = product["tileLabel"].lower()
    url = f"https://docs-be.ni.com/api/bundle/{label_lower}-feature/page/connecting-device.html"
    return find_pinouts_in_html(url)


PARSERS = [
    from_specs,
    from_get_started_pinout,
    from_get_started_overview,
    from_feature_pinout,
    from_feature_connecting,
]

lock = threading.Lock()
found_pinouts = 0


def get_pinout(i, product):
    global found_pinouts
    for parser in PARSERS:
        pinouts = parser(product)
        if len(pinouts) != 0:
            with lock:
                found_pinouts += 1
            product["pinouts"] = pinouts
            return product
    return product


in_products = json.load(open("out/products.json", "r"))
out_products = process_products(in_products, get_pinout, 10)
print(f"Found {found_pinouts} pinouts for {len(out_products)} products")
json.dump(out_products, open("out/products.json", "w"), indent=4)
