#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from freighter import Payload
from synnax.ontology.id import OntologyID

user_ontology_type = OntologyID(type="user")


class UserPayload(Payload):
    key: str
    username: str

    def ontology_id(self) -> OntologyID:
        return OntologyID(key=self.key, type="user")
