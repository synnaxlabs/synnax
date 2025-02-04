#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json


def convert_product_data(id, prods):
    for i, p in enumerate(prods):
        data = p["productData"]
        if isinstance(data, list):
            out_dict = dict()
            for item in data:
                out_dict[item["id"]] = item["value"]
            p["productData"] = out_dict


in_products = json.load(open("out/products.json", "r"))
convert_product_data(0, in_products)
json.dump(in_products, open("out/products.json", "w"), indent=4)
