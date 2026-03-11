#!/usr/bin/env python3
"""
web_scrape — scrapes structured content from a URL using Playwright + BeautifulSoup.
Input:  {"url": "...", "selector": "article", "extract": "text|html|links"}
Output: {"items": [...], "count": N, "exit_code": 0}
"""

import json
import os
import sys

try:
    from playwright.sync_api import sync_playwright
    from bs4 import BeautifulSoup
except ImportError as e:
    print(json.dumps({"error": f"missing dependency: {e}", "exit_code": 1}))
    sys.exit(0)


def main():
    raw = os.environ.get("TOOL_INPUT", "{}")
    try:
        inp = json.loads(raw)
    except json.JSONDecodeError as e:
        out({"error": str(e), "exit_code": 1})
        return

    url = inp.get("url", "")
    selector = inp.get("selector", "body")
    extract = inp.get("extract", "text")

    if not url:
        out({"error": "url is required", "exit_code": 1})
        return

    os.environ.setdefault("DISPLAY", ":99")

    with sync_playwright() as pw:
        browser = pw.chromium.launch(
            executable_path="/usr/bin/chromium-browser",
            args=["--no-sandbox", "--disable-dev-shm-usage"],
        )
        page = browser.new_page()
        try:
            page.goto(url, wait_until="domcontentloaded", timeout=30_000)
            html = page.content()
        except Exception as e:
            out({"error": str(e), "exit_code": 1})
            return
        finally:
            browser.close()

    soup = BeautifulSoup(html, "lxml")
    elements = soup.select(selector)

    items = []
    for el in elements[:50]:  # Limit to 50 elements
        if extract == "html":
            items.append(str(el))
        elif extract == "links":
            items.extend(a["href"] for a in el.find_all("a", href=True))
        else:
            items.append(el.get_text(strip=True))

    out({"items": items, "count": len(items), "exit_code": 0})


def out(d: dict):
    print(json.dumps(d))


if __name__ == "__main__":
    main()
