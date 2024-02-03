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
for i, product in enumerate(input_products):
    prep_final(i, product)
    out[product["model"]] = product
    if "estimatedPinout" in product:
        models.append(product["model"])

json.dump(models, open('out/models.json', 'w'))

json.dump(out, open('out/products-prepped.json', 'w'))
