"""
Minimalni push prijimac DATEX II pro NDIC preprocessor.
"""
from datetime import datetime, timezone
import gzip
import logging
import os
from pathlib import Path
import secrets
import xml.etree.ElementTree as ET

from fastapi import Depends, FastAPI, HTTPException, Request, Response, status
from fastapi.responses import FileResponse
from fastapi.security import HTTPBasic, HTTPBasicCredentials

from app.storage import storage

app = FastAPI()
security = HTTPBasic()
logger = logging.getLogger("datex-downloader")

USERNAME = os.getenv("DATEX_USERNAME", "")
PASSWORD = os.getenv("DATEX_PASSWORD", "")
MAX_STORED_FILES = 50

BASE_DIR = Path(__file__).resolve().parent.parent
XML_STORAGE_DIR = BASE_DIR / "data" / "ndic_messages"
XML_STORAGE_DIR.mkdir(parents=True, exist_ok=True)


def verify_credentials(credentials: HTTPBasicCredentials = Depends(security)):
    if USERNAME == "" and PASSWORD == "":
        return ""

    correct_username = secrets.compare_digest(credentials.username, USERNAME)
    correct_password = secrets.compare_digest(credentials.password, PASSWORD)
    if not (correct_username and correct_password):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Incorrect authentication credentials",
            headers={"WWW-Authenticate": "Basic"},
        )
    return credentials.username


def load_latest_message() -> None:
    latest_file = None
    for path in XML_STORAGE_DIR.glob("*.xml"):
        if latest_file is None or path.stat().st_mtime > latest_file.stat().st_mtime:
            latest_file = path

    if latest_file is None:
        logger.info("No persisted DATEX snapshot found on startup")
        return

    storage.save(latest_file.read_text(encoding="utf-8"))
    logger.info("Loaded persisted DATEX snapshot on startup: %s", latest_file.name)


@app.on_event("startup")
def startup_event():
    load_latest_message()


def persist_xml(raw_xml: str) -> Path:
    timestamp = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H-%M-%SZ")
    path = XML_STORAGE_DIR / f"{timestamp}.xml"
    counter = 1
    while path.exists():
        path = XML_STORAGE_DIR / f"{timestamp}-{counter}.xml"
        counter += 1
    path.write_text(raw_xml, encoding="utf-8")
    prune_old_files()
    return path


def prune_old_files() -> None:
    files = sorted(XML_STORAGE_DIR.glob("*.xml"), key=lambda path: path.stat().st_mtime)
    while len(files) > MAX_STORED_FILES:
        files[0].unlink(missing_ok=True)
        files.pop(0)


def decode_request_xml(raw_data: bytes, content_encoding: str) -> str:
    if content_encoding.lower() == "gzip":
        try:
            raw_data = gzip.decompress(raw_data)
        except OSError as exc:
            raise HTTPException(status_code=400, detail="Failed to decompress gzip data") from exc

    try:
        raw_xml = raw_data.decode("utf-8")
    except UnicodeDecodeError as exc:
        raise HTTPException(status_code=400, detail="Invalid XML encoding") from exc

    try:
        ET.fromstring(raw_xml)
    except ET.ParseError as exc:
        raise HTTPException(status_code=400, detail="Invalid XML") from exc

    return raw_xml


@app.post("/datex-in")
async def datex_in(request: Request, username: str = Depends(verify_credentials)):
    body = await request.body()
    content_encoding = request.headers.get("Content-Encoding", "")
    raw_xml = decode_request_xml(body, content_encoding)
    storage.save(raw_xml)
    path = persist_xml(raw_xml)
    try:
        root_tag = ET.fromstring(raw_xml).tag
    except ET.ParseError:
        root_tag = "unknown"
    logger.info(
        "Accepted DATEX push | user=%s encoding=%s bytes=%d root=%s stored=%s",
        username or "<anonymous>",
        content_encoding or "<none>",
        len(body),
        root_tag,
        path.name,
    )
    return {"status": "ok"}


@app.get("/api/latest")
async def get_latest():
    latest_raw = storage.get_latest()
    if latest_raw is None:
        logger.warning("GET /api/latest -> no data available")
        raise HTTPException(status_code=404, detail="No data available")
    logger.info("Served latest DATEX snapshot as JSON wrapper")
    return {"latest_raw": latest_raw}


@app.get("/api/latest.xml")
async def get_latest_xml():
    latest_raw = storage.get_latest()
    if latest_raw is None:
        logger.warning("GET /api/latest.xml -> no data available")
        raise HTTPException(status_code=404, detail="No data available")
    logger.info("Served latest DATEX snapshot as XML")
    return Response(content=latest_raw, media_type="application/xml")


@app.get("/download/latest.xml")
async def download_latest():
    files = sorted(XML_STORAGE_DIR.glob("*.xml"), key=lambda path: path.stat().st_mtime)
    if not files:
        logger.warning("GET /download/latest.xml -> no messages available")
        raise HTTPException(status_code=404, detail="No messages available")
    logger.info("Served latest persisted DATEX snapshot: %s", files[-1].name)
    return FileResponse(files[-1], media_type="application/xml", filename="latest.xml")


@app.get("/healthz")
async def healthz():
    return {"status": "ok"}
