#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json

in_products = json.load(open("out/products.json", "r"))

filtered_products = list()
existing_ids = set()
count = 0
for product in in_products:
    if product["id"] not in existing_ids:
        if "controller" in product["category"] or "chassis" in product["category"]:
            continue
        existing_ids.add(product["id"])
        filtered_products.append(product)
    else:
        count += 1

print(f"Removed {count} duplicates from {len(in_products)} products")

json.dump(filtered_products, open("out/products.json", "w"), indent=4)
