// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { expect, test } from '@playwright/test';

test.describe('Basic Navigation', () => {
  test('should navigate to localhost:5173', async ({ page }) => {
    await page.goto('/');
    
    await expect(page).toHaveURL('http://localhost:5173/');
    
    const title = await page.title();
    expect(title).toBeTruthy();
    expect(title).toBe('Synnax Console');
    
    await page.waitForLoadState('networkidle');
  });

  test('should find Search & Command text', async ({ page }) => {
    await page.goto('/');
    
    await page.waitForLoadState('networkidle');
    
    const searchText = page.getByText('Search & Command');
    await expect(searchText).toBeVisible();
  });
});