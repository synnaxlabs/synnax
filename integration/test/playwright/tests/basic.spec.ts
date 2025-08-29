import { test, expect } from '@playwright/test';

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
    
    const searchText = await page.getByText('Search & Command');
    await expect(searchText).toBeVisible();
  });
});