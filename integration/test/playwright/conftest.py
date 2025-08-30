import pytest
from playwright.sync_api import Page


@pytest.fixture(autouse=True)
def setup_timeouts(page: Page):
    """Set default timeouts for all tests"""
    # Set default timeout for actions (click, type, etc.)
    page.set_default_timeout(2000)  # 2s

    page.set_default_navigation_timeout(30000)  # 30s