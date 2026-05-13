"""
@file storage.py
@brief In-memory úložiště poslední přijaté DATEX II zprávy.

@author Dominik Vondruška
@ingroup riot_datex_downloader

@par Autorský podíl
- Dominik Vondruška: návrh a implementace celé funkcionality souboru.
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
