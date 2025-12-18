// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Test utilities for simulating performance issues in the dashboard.
 * These are dev-only helpers to verify that performance monitoring works correctly.
 */

/**
 * Dispatches a synthetic event to simulate user interaction.
 * This helps test event attribution for long tasks.
 *
 * @param eventType - Type of event (click, keydown, etc.)
 * @param target - Optional target element (defaults to document.body)
 */
export const dispatchSyntheticEvent = (
  eventType: string,
  target: Element = document.body,
): void => {
  const event = new Event(eventType, { bubbles: true, cancelable: true });
  target.dispatchEvent(event);
  console.log(`[PerfTest] Dispatched synthetic "${eventType}" event`);
};

/**
 * Blocks the main thread for the specified duration by doing busy work.
 * This will trigger a "long task" if duration > 50ms.
 *
 * @param durationMs - How long to block the thread (in milliseconds)
 * @param eventType - Optional event type to dispatch before blocking
 */
export const simulateLongTask = (
  durationMs: number,
  eventType?: string,
): void => {
  if (eventType) {
    dispatchSyntheticEvent(eventType);
    console.log(
      `[PerfTest] Simulating long task for ${durationMs}ms after "${eventType}" event...`,
    );
  } else 
    console.log(`[PerfTest] Simulating long task for ${durationMs}ms...`);
  

  const start = performance.now();

  // Busy loop to block the main thread
  while (performance.now() - start < durationMs) 
    // Intentionally blocking work
    Math.sqrt(Math.random());
  

  const actual = performance.now() - start;
  console.log(`[PerfTest] Long task completed in ${actual.toFixed(1)}ms`);
};

/**
 * Simulates multiple long tasks with random durations.
 *
 * @param count - Number of long tasks to simulate
 * @param minMs - Minimum duration (default: 50ms)
 * @param maxMs - Maximum duration (default: 500ms)
 * @param delayMs - Delay between tasks (default: 100ms)
 */
export const simulateMultipleLongTasks = async (
  count: number,
  minMs = 50,
  maxMs = 500,
  delayMs = 100,
): Promise<void> => {
  console.log(
    `[PerfTest] Simulating ${count} long tasks (${minMs}-${maxMs}ms each)...`,
  );

  for (let i = 0; i < count; i++) {
    const duration = Math.random() * (maxMs - minMs) + minMs;
    simulateLongTask(duration);

    // Wait before next task
    if (i < count - 1) 
      await new Promise((resolve) => setTimeout(resolve, delayMs));
    
  }

  console.log(`[PerfTest] Completed ${count} long tasks`);
};

/**
 * Simulates a specific long task scenario.
 */
export const simulateLongTaskScenario = {

  lightStutter: () => simulateLongTask(100),

  mediumLag: () => simulateLongTask(250),

  heavyFreeze: () => simulateLongTask(1000),

  repeatedStutters: () => simulateMultipleLongTasks(5, 75, 150, 200),

  gradualDegradation: async () => {
    for (let i = 1; i <= 5; i++) {
      simulateLongTask(50 * i);
      await new Promise((resolve) => setTimeout(resolve, 300));
    }
  },
};

/**
 * Simulates long tasks triggered by specific event types.
 * Tests event attribution system.
 */
export const simulateEventBasedLongTask = {

  slowClick: () => simulateLongTask(150, "click"),

  slowKeydown: () => simulateLongTask(100, "keydown"),

  slowInput: () => simulateLongTask(120, "input"),

  slowDrag: () => simulateLongTask(200, "dragstart"),

  slowPaste: () => simulateLongTask(180, "paste"),

  slowScroll: () => simulateLongTask(90, "scroll"),

  slowResize: () => simulateLongTask(130, "resize"),

  slowFocus: () => simulateLongTask(110, "focus"),

  /** Test all event types - fires with small breaks to register as separate tasks (4 rounds = 32 events) */
  testAllEvents: async () => {
    const events = [
      { type: "click", duration: 75 },
      { type: "keydown", duration: 75 },
      { type: "input", duration: 75 },
      { type: "paste", duration: 75 },
      { type: "scroll", duration: 75 },
      { type: "dragstart", duration: 75 },
      { type: "focus", duration: 75 },
      { type: "resize", duration: 75 },
    ];

    console.log("[PerfTest] Testing all event types (4 rounds, 32 total)...");
    for (let round = 0; round < 4; round++) 
      for (const { type, duration } of events) {
        simulateLongTask(duration, type);
        // Small delay to ensure tasks are registered separately
        await new Promise((resolve) => setTimeout(resolve, 10));
      }
    
    console.log("[PerfTest] Completed 32 long tasks");
  },
};

export interface NetworkRequestOptions {
  url?: string;
  latencyMs?: number;
  shouldFail?: boolean;
  method?: string;
  body?: any;
}

export const simulateNetworkRequest = async (
  options: NetworkRequestOptions = {},
): Promise<Response | null> => {
  const {
    url = "https://httpbin.org/delay/0",
    latencyMs = 0,
    shouldFail = false,
    method = "GET",
    body,
  } = options;

  try {
    if (latencyMs > 0) 
      await new Promise((resolve) => setTimeout(resolve, latencyMs));
    

    if (shouldFail) {
      await fetch("https://invalid-domain-that-does-not-exist.local");
      return null;
    }

    const response = await fetch(url, {
      method,
      body: body ? JSON.stringify(body) : undefined,
      headers: body ? { "Content-Type": "application/json" } : undefined,
      mode: "no-cors",
    });

    return response;
  } catch (_error) {
    return null;
  }
};

const HTTPBIN_ENDPOINTS = [
  "/get",
  "/post",
  "/put",
  "/delete",
  "/patch",
  "/status/200",
  "/status/201",
  "/status/400",
  "/status/404",
  "/status/500",
  "/headers",
  "/ip",
  "/user-agent",
  "/uuid",
  "/delay/0",
  "/delay/1",
  "/delay/2",
  "/base64/SFRUUEJJTiBpcyBhd2Vzb21l",
  "/json",
  "/html",
  "/xml",
  "/robots.txt",
  "/deny",
  "/cache",
  "/cache/60",
  "/bytes/1024",
  "/stream-bytes/1024",
  "/links/10",
  "/image",
  "/image/png",
  "/image/jpeg",
  "/image/svg",
];

export const simulateUniqueEndpoints = {
  allEndpoints: async (iterations = 1, delayMs = 0) => {
    for (let iter = 0; iter < iterations; iter++) 
      for (const endpoint of HTTPBIN_ENDPOINTS) {
        const method = endpoint.startsWith("/post")
          ? "POST"
          : endpoint.startsWith("/put")
            ? "PUT"
            : endpoint.startsWith("/delete")
              ? "DELETE"
              : endpoint.startsWith("/patch")
                ? "PATCH"
                : "GET";

        void simulateNetworkRequest({
          url: `https://httpbin.org${endpoint}`,
          method,
        });

        if (delayMs > 0) 
          await new Promise((resolve) => setTimeout(resolve, delayMs));
        
      }
    
  },

  randomEndpoints: async (count: number, delayMs = 0) => {
    for (let i = 0; i < count; i++) {
      const endpoint =
        HTTPBIN_ENDPOINTS[Math.floor(Math.random() * HTTPBIN_ENDPOINTS.length)];
      const method = endpoint.startsWith("/post")
        ? "POST"
        : endpoint.startsWith("/put")
          ? "PUT"
          : endpoint.startsWith("/delete")
            ? "DELETE"
            : endpoint.startsWith("/patch")
              ? "PATCH"
              : "GET";

      void simulateNetworkRequest({
        url: `https://httpbin.org${endpoint}`,
        method,
      });

      if (delayMs > 0) 
        await new Promise((resolve) => setTimeout(resolve, delayMs));
      
    }
  },

  uniqueSequential: async (count: number, delayMs = 0) => {
    for (let i = 0; i < count; i++) {
      void simulateNetworkRequest({
        url: `https://httpbin.org/anything/inject/request_${i}`,
      });
      if (delayMs > 0) 
        await new Promise((resolve) => setTimeout(resolve, delayMs));
      
    }
  },
};

// Expose to window for easy console access in development
if (typeof window !== "undefined" && import.meta.env.DEV) {
  (window as any).__perfTest = {
    simulateLongTask,
    simulateMultipleLongTasks,
    dispatchSyntheticEvent,
    scenarios: simulateLongTaskScenario,
    events: simulateEventBasedLongTask,
    network: simulateUniqueEndpoints,
    simulateNetworkRequest,
  };

  console.log(
    "[PerfTest] Performance test utils loaded. Available commands:\n\n" +
      "Long Tasks:\n" +
      "  __perfTest.simulateLongTask(durationMs, eventType?)\n" +
      "  __perfTest.simulateMultipleLongTasks(count, minMs, maxMs, delayMs)\n" +
      "  __perfTest.dispatchSyntheticEvent(eventType)\n\n" +
      "Scenarios:\n" +
      "  __perfTest.scenarios.lightStutter()\n" +
      "  __perfTest.scenarios.mediumLag()\n" +
      "  __perfTest.scenarios.heavyFreeze()\n" +
      "  __perfTest.scenarios.repeatedStutters()\n" +
      "  __perfTest.scenarios.gradualDegradation()\n\n" +
      "Event Attribution Tests:\n" +
      "  __perfTest.events.slowClick()\n" +
      "  __perfTest.events.slowKeydown()\n" +
      "  __perfTest.events.slowInput()\n" +
      "  __perfTest.events.slowPaste()\n" +
      "  __perfTest.events.slowScroll()\n" +
      "  __perfTest.events.slowDrag()\n" +
      "  __perfTest.events.slowResize()\n" +
      "  __perfTest.events.slowFocus()\n" +
      "  __perfTest.events.testAllEvents()\n\n" +
      "Network Tests:\n" +
      "  __perfTest.network.allEndpoints(iterations, delayMs)\n" +
      "  __perfTest.network.randomEndpoints(count, delayMs)\n" +
      "  __perfTest.network.uniqueSequential(count, delayMs)\n" +
      "  __perfTest.simulateNetworkRequest({ url, method, latencyMs })",
  );
}
