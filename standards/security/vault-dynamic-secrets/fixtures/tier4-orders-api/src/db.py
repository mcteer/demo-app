"""Minimal app stub for golden fixture — no static credentials on disk."""

import os


def get_database_url() -> str:
    url = os.environ.get("DATABASE_URL")
    if not url:
        raise RuntimeError("DATABASE_URL must be supplied by VSO at runtime")
    return url
