#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import requests
import json

url = "https://api.ni.com/e/mktg/product-catalog/1/en-us/category"

# All of the different NI devices that we need to get info for
CATEGORIES = [
    "multifunction-io",
    "voltage",
    "current",
    "digital-io",
    "temperature",
    "compactdaq-chassis",
    "pxi-chassis",
]

headers = {
    "Client_id": "1fcca773908c4e6da0500a60ea393e83",
    "Client_secret": "9EC7AA4494614C25AE57f022Bc6f7Bac",
}

# For each category, make a GET request to the API
# and print it as json

COUNT = 0

products = []


def check_has_property(product, property):
    for item in product["productData"]:
        if item["id"] == property:
            return True
    return False


for category in CATEGORIES:
    cat_url = url + "/" + category
    print("Fetching category: " + cat_url)
    response = requests.get(url + "/" + category, headers=headers)
    n_pages = response.json()["pagination"]["totalPages"]
    for page in range(1, n_pages + 1):
        print("Category: " + category + " Page: " + str(page))
        response = requests.get(
            url + "/" + category + "?&page=" + str(page) + "&getPrice=false",
            headers=headers,
        )
        j = response.json()
        l_products = j["products"]
        COUNT += len(j["products"])
        for product in l_products:
            for key in list(product.keys()):
                if key not in [
                    "id",
                    "tileLabel",
                    "productID",
                    "productData",
                    "productSpecs",
                ]:
                    del product[key]

            product["category"] = category

            if "Bundle" not in product["tileLabel"]:
                products.append(product)

for product in products:
    specs_url = product["productSpecs"]
    if "search" not in specs_url and "specs.html" in specs_url:
        specs_url = specs_url[specs_url.find("bundle") + 7 :]
        base_url = "https://docs-be.ni.com/api/bundle/"
        response = requests.get(base_url + specs_url, headers=headers)
    elif "search" in specs_url:
        print(specs_url)

# dump the products JSON file
with open("out/products-arc.json", "w") as f:
    json.dump(products, f)
