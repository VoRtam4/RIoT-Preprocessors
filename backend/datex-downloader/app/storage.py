"""
@File: storage.py
@Author: Dominik Vondruška
@Project: Bakalářská práce — Systém pro monitorování otevřených dat v reálném čase
@Description: Jednoduché in-memory uložení poslední přijaté NDIC zprávy.
"""
from typing import Optional
from threading import Lock

class DatexStorage:
    def __init__(self):
        self._latest_raw = None
        self._lock = Lock()

    def save(self, raw_xml: str):
        with self._lock:
            self._latest_raw = raw_xml

    def get_latest(self) -> Optional[str]:
        with self._lock:
            return self._latest_raw

storage = DatexStorage()
