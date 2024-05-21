#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json


def prep_final(i, product):
    product["model"] = product["tileLabel"]
    product["key"] = product["id"]
    del product["tileLabel"]
    del product["id"]

    pinouts = product.get("pinouts", [])
    next_pinouts = list()
    for pinout in pinouts:
        if "info" in pinout:
            del pinout["info"]
            # strip the title of newline characters
            if "title" in pinout:
                pinout["title"] = pinout["title"].strip()
                # remove any \n or \t from the title
                pinout["title"] = pinout["title"].replace("\n", " ")
                pinout["title"] = pinout["title"].replace("\t", " ")
                # remove any spaces longer than 1
                pinout["title"] = " ".join(pinout["title"].split())
            next_pinouts.append(pinout)

    product["pinouts"] = next_pinouts


input_products = json.load(open('out/products.json', 'r'))
out = dict()
models = list()
support_table = list()
for i, product in enumerate(input_products):
    prep_final(i, product)
    out[product["model"]] = product
    if "estimatedPinout" in product:
        models.append(product["model"])
        support_table.append({
            "model": product["model"],
            "url": product["productSpecs"]
        })

json.dump(models, open('out/models.json', 'w'))
json.dump(out, open('out/products-prepped.json', 'w'))

html_table_rows = ""

for row in support_table:
    html_table_rows += f'<tr><td>{row["model"]}</td><td><a href="{row["url"]}">Datasheet</a></td></tr>'

# open a new file and write the html table
with open('out/support_table.html', 'w') as f:
    f.write(f'<table><tr><th>Model</th><th>Link</th></tr>{html_table_rows}</table>')
