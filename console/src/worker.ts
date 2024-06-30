// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { pluto } from "@synnaxlabs/pluto/ether";

class CustomPromise extends Promise {
  static openPromisesCount = 0;

  constructor(executor) {
    CustomPromise.openPromisesCount++;
    super((resolve, reject) => {
      executor(
        (value) => {
          CustomPromise.openPromisesCount--;
          CustomPromise.maybeLogOpenPromisesCount();
          resolve(value);
        },
        (reason) => {
          CustomPromise.openPromisesCount--;
          CustomPromise.maybeLogOpenPromisesCount();
          reject(reason);
        },
      );
    });
  }

  static maybeLogOpenPromisesCount() {
    // console.log(`Open promises count: ${CustomPromise.openPromisesCount}`);
  }

  static getOpenPromisesCount() {
    return CustomPromise.openPromisesCount;
  }
}

// Override the native Promise with the custom implementation
self.Promise = CustomPromise;

pluto.render();
